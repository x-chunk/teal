package teal

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

// serve answers every request with body, and keeps the last request seen.
func serve(t *testing.T, body string) (*Client, func() *http.Request, func() string) {
	t.Helper()
	var last *http.Request
	var sent string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		last = r
		if b, err := io.ReadAll(r.Body); err == nil {
			sent = string(b)
		}
		io.WriteString(w, body)
	})
	return c, func() *http.Request { return last }, func() string { return sent }
}

// The example from the documentation, byte for byte.
func TestAppGetDecodesTheDocumentedAnswer(t *testing.T) {
	c, _, _ := serve(t, `{"ok":true,"data":{
		"id": 12, "name": "Support desk", "billing": "hybrid",
		"key_prefix": "aek_7Fh2Kd", "key_issued_at": "2026-09-01T10:04:11Z",
		"balance": {"credits": 4975000, "display": "$4.975"},
		"funded": {"credits": 5000000, "display": "$5.00"},
		"spent": {"credits": 25000, "display": "$0.025"},
		"requests": 1204, "last_used_at": "2026-09-06T08:22:40Z",
		"disabled": false, "created_at": "2026-09-01T10:04:11Z",
		"account_id": 1256738876}}`)

	app, _, err := c.App.Get(context.Background())
	if err != nil {
		t.Fatalf("App.Get: %v", err)
	}
	if app.ID != 12 || app.Name != "Support desk" || app.Billing != BillingHybrid {
		t.Errorf("app = %+v", app)
	}
	if app.Balance.Credits != 4_975_000 || app.Balance.Credits.String() != "$4.975" {
		t.Errorf("balance = %+v", app.Balance)
	}
	if want := time.Date(2026, 9, 1, 10, 4, 11, 0, time.UTC); !app.KeyIssuedAt.Equal(want) {
		t.Errorf("key_issued_at = %v, want %v", app.KeyIssuedAt, want)
	}
}

func TestQuotaKeepsItsWindow(t *testing.T) {
	c, _, _ := serve(t, `{"ok":true,"data":[
		{"key":"messages:stored","label":"Archived messages","unit":"messages",
		 "limit":1000000,"used":51204,"remaining":948796,"unlimited":false},
		{"key":"export:weekly","label":"Exports","unit":"exports","limit":100,
		 "used":3,"remaining":97,"unlimited":false,"window":"week",
		 "reset_at":"2026-09-08T00:00:00Z"}]}`)

	quotas, _, err := c.App.Quotas(context.Background())
	if err != nil {
		t.Fatalf("App.Quotas: %v", err)
	}
	if len(quotas) != 2 {
		t.Fatalf("got %d quotas, want 2", len(quotas))
	}
	if !quotas[0].ResetAt.IsZero() || quotas[0].Window != "" {
		t.Errorf("a ceiling should carry no window: %+v", quotas[0])
	}
	if quotas[1].Window != "week" || quotas[1].ResetAt.IsZero() {
		t.Errorf("an allowance should carry one: %+v", quotas[1])
	}
}

// The point of the pointers: zero is a value, not an absence.
func TestRetentionUpdateSendsOnlyWhatIsSet(t *testing.T) {
	for _, tc := range []struct {
		name string
		req  RetentionUpdateRequest
		want string
	}{
		{"nothing", RetentionUpdateRequest{}, `{}`},
		{"ttl off", RetentionUpdateRequest{TTLSeconds: ptr(int64(0))}, `{"ttl_seconds":0}`},
		{"mode only", RetentionUpdateRequest{Mode: ptr(RetentionRotate)}, `{"mode":"rotate"}`},
		{"in_chat false", RetentionUpdateRequest{InChat: ptr(false)}, `{"in_chat":false}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _, sent := serve(t, `{"ok":true,"data":{}}`)
			if _, _, err := c.Settings.UpdateRetention(context.Background(), tc.req); err != nil {
				t.Fatalf("UpdateRetention: %v", err)
			}
			if sent() != tc.want {
				t.Errorf("body = %s, want %s", sent(), tc.want)
			}
		})
	}
}

func TestSearchBodyLeavesOutWhatWasNotAsked(t *testing.T) {
	c, _, sent := serve(t, `{"ok":true,"data":{"total":37,"pages":4,"per_page":10}}`)

	res, _, err := c.Archive.Search(context.Background(), SearchRequest{
		Conditions: []Condition{{Field: "text", Mode: MatchContains, Value: "invoice"}},
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	want := `{"conditions":[{"field":"text","mode":"ct","value":"invoice"}]}`
	if sent() != want {
		t.Errorf("body = %s, want %s", sent(), want)
	}
	if res.Total != 37 || res.Pages != 4 {
		t.Errorf("result = %+v", res)
	}
}

func TestPathIDAndQuery(t *testing.T) {
	for _, tc := range []struct {
		name       string
		call       func(*Client) error
		path, rawQ string
	}{
		{"message by id",
			func(c *Client) error { _, _, err := c.Archive.Message(context.Background(), 90210); return err },
			"/v1/messages/90210", ""},
		{"versions",
			func(c *Client) error { _, _, err := c.Archive.Versions(context.Background(), 90210); return err },
			"/v1/messages/90210/versions", ""},
		{"negative chat id",
			func(c *Client) error {
				_, _, err := c.Insights.Portrait(context.Background(), -1001234567890)
				return err
			},
			"/v1/portraits/-1001234567890", ""},
		{"usage window",
			func(c *Client) error {
				_, _, err := c.App.Usage(context.Background(), &UsageRequest{Days: 7, ByDay: true})
				return err
			},
			"/v1/usage", "by_day=true&days=7"},
		{"usage defaults",
			func(c *Client) error { _, _, err := c.App.Usage(context.Background(), nil); return err },
			"/v1/usage", ""},
		{"chats page",
			func(c *Client) error {
				_, _, err := c.Archive.Chats(context.Background(), &ChatsRequest{Page: 2})
				return err
			},
			"/v1/chats", "page=2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, last, _ := serve(t, `{"ok":true,"data":{}}`)
			if err := tc.call(c); err != nil {
				t.Fatalf("call: %v", err)
			}
			if got := last().URL.Path; got != tc.path {
				t.Errorf("path = %q, want %q", got, tc.path)
			}
			if got := last().URL.RawQuery; got != tc.rawQ {
				t.Errorf("query = %q, want %q", got, tc.rawQ)
			}
		})
	}
}

func TestVaultDeleteCarriesThePassphraseInTheBody(t *testing.T) {
	c, last, sent := serve(t, `{"ok":true,"data":{}}`)

	meta, err := c.Vault.Delete(context.Background(), VaultDeleteRequest{Passphrase: "the pale blue dot"})
	if err != nil {
		t.Fatalf("Vault.Delete: %v", err)
	}
	if last().Method != http.MethodPost {
		t.Errorf("method = %s, want POST", last().Method)
	}
	if last().URL.RawQuery != "" {
		t.Errorf("a passphrase must not reach the query: %q", last().URL.RawQuery)
	}
	if want := `{"passphrase":"the pale blue dot"}`; sent() != want {
		t.Errorf("body = %s, want %s", sent(), want)
	}
	if meta == nil {
		t.Error("meta = nil")
	}
}

func TestExportHandsBackTheDocument(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Aether-Export-Total", "412")
		io.WriteString(w, `{"account_id":1256738876,"total":412,"filters":1,"messages":[]}`)
	})

	body, meta, err := c.Archive.Export(context.Background(), SearchRequest{})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	defer body.Close()

	var doc ExportDocument
	if err := json.NewDecoder(body).Decode(&doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if doc.Total != 412 {
		t.Errorf("total = %d, want 412", doc.Total)
	}
	if got := meta.Header.Get("X-Aether-Export-Total"); got != "412" {
		t.Errorf("X-Aether-Export-Total = %q, want 412", got)
	}
}

func TestWrongPassphraseIsNotFound(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, `{"ok":false,"error":{"code":"not_found","message":"not there"}}`)
	})

	_, _, err := c.Vault.Reveal(context.Background(), VaultRevealRequest{Passphrase: "wrong"})
	if !IsCode(err, CodeNotFound) {
		t.Fatalf("err = %v, want not_found", err)
	}
}

func ptr[T any](v T) *T { return &v }
