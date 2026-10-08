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

	resp, err := c.App.Get(context.Background()).Value()
	if err != nil {
		t.Fatalf("App.Get: %v", err)
	}
	app := resp.Data
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

// The example from the documentation, byte for byte.
func TestAccountSaysWhenItWasMeasured(t *testing.T) {
	c, _, _ := serve(t, `{"ok":true,"data":{
		"account_id": 1256738876,
		"plan": {
			"tier": "max",
			"name": "Max",
			"price_cents": 1999,
			"until": "2026-10-01T00:00:00Z",
			"permissions": ["glance:allow_insights", "plugin:allow_use"]
		},
		"balance_cents": 1204,
		"quotas": [
			{"key": "search:daily", "label": "Searches", "unit": "searches", "limit": 2000,
			 "used": 41, "remaining": 1959, "unlimited": false, "window": "day",
			 "reset_at": "2026-09-07T00:00:00Z"}
		],
		"measured_at": 1757160000}}`)

	resp, err := c.App.Account(context.Background()).Value()
	if err != nil {
		t.Fatalf("App.Account: %v", err)
	}
	acc := resp.Data
	if acc.Plan.Tier != "max" || len(acc.Plan.Permissions) != 2 {
		t.Errorf("plan = %+v", acc.Plan)
	}
	if want := time.Unix(1757160000, 0).UTC(); !acc.MeasuredAt.At().Equal(want) {
		t.Errorf("measured_at = %v, want %v", acc.MeasuredAt.At(), want)
	}
	if len(acc.Quotas) != 1 || acc.Quotas[0].Remaining != 1959 {
		t.Errorf("quotas = %+v", acc.Quotas)
	}
}

// An unlimited quota has no ceiling to state, and says so with null.
func TestUnlimitedQuotaDecodes(t *testing.T) {
	c, _, _ := serve(t, `{"ok":true,"data":[
		{"key":"search:daily","label":"Searches","unit":"searches",
		 "limit":null,"used":41,"remaining":null,"unlimited":true,"window":"day"}]}`)

	resp, err := c.App.Quotas(context.Background()).Value()
	if err != nil {
		t.Fatalf("App.Quotas: %v", err)
	}
	quotas := resp.Data
	if len(quotas) != 1 || !quotas[0].Unlimited || quotas[0].Used != 41 {
		t.Errorf("quotas = %+v", quotas)
	}
}

func TestUsageBreaksDownByDayAndOperation(t *testing.T) {
	c, _, _ := serve(t, `{"ok":true,"data":{
		"from": "2026-08-08", "to": "2026-09-06", "calls": 40,
		"credits": {"credits": 25000, "display": "$0.025"},
		"ops": [
			{"op": "search:query", "calls": 20, "units": 20, "credits": {"credits": 20000, "display": "$0.02"}},
			{"op": "messages:read", "calls": 20, "units": 20, "credits": {"credits": 5000, "display": "$0.005"}}
		],
		"days": [
			{"op": "search:query", "calls": 12, "units": 12, "credits": {"credits": 12000, "display": "$0.012"}, "day": "2026-09-06"},
			{"op": "search:query", "calls": 8, "units": 8, "credits": {"credits": 8000, "display": "$0.008"}, "day": "2026-09-05"}
		]}}`)

	resp, err := c.App.Usage(context.Background(), &UsageRequest{ByDay: true}).Value()
	if err != nil {
		t.Fatalf("App.Usage: %v", err)
	}
	usage := resp.Data
	if len(usage.Ops) != 2 || usage.Ops[0].Day != "" {
		t.Errorf("ops = %+v", usage.Ops)
	}
	if len(usage.Days) != 2 || usage.Days[0].Day != "2026-09-06" || usage.Days[1].Calls != 8 {
		t.Errorf("days = %+v", usage.Days)
	}
}

// The example from the documentation, byte for byte — and a rate that is not
// a whole number, which the API is free to configure.
func TestPricesDecodesEveryRate(t *testing.T) {
	c, _, _ := serve(t, `{"ok":true,"data":{
		"prices": [
			{"op": "search:query", "label": "Archive query", "unit": "query",
			 "price": {"credits": 1000, "display": "$0.001"}, "limit": "search:daily",
			 "charging": "always", "description": "One search of the archive, or the count behind one."}
		],
		"credits_per_cent": 10000,
		"rates": {
			"shared":  {"per_second": 10, "burst": 20},
			"hybrid":  {"per_second": 10, "burst": 20},
			"credits": {"per_second": 50, "burst": 100}
		},
		"account_rate": {"per_second": 60, "burst": 120},
		"global_rate": {"per_second": 300, "burst": 600},
		"quota_ttl_seconds": 30}}`)

	resp, err := c.App.Prices(context.Background()).Value()
	if err != nil {
		t.Fatalf("App.Prices: %v", err)
	}
	prices := resp.Data
	if prices.Rates[BillingCredits].PerSecond != 50 || prices.Prices[0].Charging != ChargingAlways {
		t.Errorf("prices = %+v", prices)
	}
	if prices.AccountRate == nil || prices.AccountRate.Burst != 120 || prices.GlobalRate == nil {
		t.Errorf("account_rate = %+v, global_rate = %+v", prices.AccountRate, prices.GlobalRate)
	}

	c, _, _ = serve(t, `{"ok":true,"data":{"rates":{"shared":{"per_second":0.5,"burst":2}}}}`)
	resp, err = c.App.Prices(context.Background()).Value()
	if err != nil {
		t.Fatalf("App.Prices with a fractional rate: %v", err)
	}
	prices = resp.Data
	if prices.Rates[BillingShared].PerSecond != 0.5 || prices.AccountRate != nil {
		t.Errorf("prices = %+v", prices)
	}
}

func TestQuotaKeepsItsWindow(t *testing.T) {
	c, _, _ := serve(t, `{"ok":true,"data":[
		{"key":"messages:stored","label":"Archived messages","unit":"messages",
		 "limit":1000000,"used":51204,"remaining":948796,"unlimited":false},
		{"key":"export:weekly","label":"Exports","unit":"exports","limit":100,
		 "used":3,"remaining":97,"unlimited":false,"window":"week",
		 "reset_at":"2026-09-08T00:00:00Z"}]}`)

	resp, err := c.App.Quotas(context.Background()).Value()
	if err != nil {
		t.Fatalf("App.Quotas: %v", err)
	}
	quotas := resp.Data
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

	resp, err := c.Archive.Search(context.Background(), SearchRequest{
		Conditions: []Condition{{Field: "text", Mode: MatchContains, Value: "invoice"}},
	}).Value()
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	res := resp.Data
	want := `{"conditions":[{"field":"text","mode":"ct","value":"invoice"}]}`
	if sent() != want {
		t.Errorf("body = %s, want %s", sent(), want)
	}
	if res.Total != 37 || res.Pages != 4 {
		t.Errorf("result = %+v", res)
	}
}

// Every value is text on the wire, and a mode left out is the API's eq.
func TestConditionValueIsAlwaysText(t *testing.T) {
	c, _, sent := serve(t, `{"ok":true,"data":{"total":412}}`)

	chat := int64(-1001234567890)
	err := c.Archive.Count(context.Background(), SearchRequest{
		Chat:       &chat,
		Conditions: []Condition{{Field: "deleted", Value: "true"}},
	}).Error()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	want := `{"chat":-1001234567890,"conditions":[{"field":"deleted","value":"true"}]}`
	if sent() != want {
		t.Errorf("body = %s, want %s", sent(), want)
	}
}

// The example from the documentation, byte for byte.
func TestSearchPageTurnsByTheQueryID(t *testing.T) {
	c, last, sent := serve(t, `{"ok":true,"data":{
		"messages": [
			{"id": 90190, "message_id": 4402, "chat": -1001234567890, "sender": 1256738876,
			 "receiver": 42, "text": "invoice for August", "deleted": false,
			 "created_at": "2026-08-30T17:40:11Z", "versions": 1}
		],
		"total": 37,
		"page": 1,
		"pages": 4,
		"per_page": 10,
		"query_id": "5tQx0bYFvHh1nC2kq9Lr3w"}}`)

	resp, err := c.Archive.SearchPage(context.Background(), SearchPageRequest{
		QueryID: "5tQx0bYFvHh1nC2kq9Lr3w",
		Page:    1,
	}).Value()
	if err != nil {
		t.Fatalf("SearchPage: %v", err)
	}
	res := resp.Data
	if last().Method != http.MethodPost || last().URL.Path != "/v1/messages/search/page" {
		t.Errorf("request = %s %s", last().Method, last().URL.Path)
	}
	if want := `{"query_id":"5tQx0bYFvHh1nC2kq9Lr3w","page":1}`; sent() != want {
		t.Errorf("body = %s, want %s", sent(), want)
	}
	if res.QueryID != "5tQx0bYFvHh1nC2kq9Lr3w" || res.Page != 1 || len(res.Messages) != 1 {
		t.Errorf("result = %+v", res)
	}
	if res.Messages[0].ID != 90190 || res.Messages[0].Versions != 1 {
		t.Errorf("message = %+v", res.Messages[0])
	}
}

// A reply carries what it answers, and the chat's names travel with it.
func TestMessageCarriesItsReplyAndNames(t *testing.T) {
	c, _, _ := serve(t, `{"ok":true,"data":{
		"id": 90210, "message_id": 4471, "chat": -1001234567890, "sender": 1256738876,
		"receiver": 42, "text": "the invoice is attached", "username": "ann",
		"first_name": "Ann", "last_name": "Weber", "sender_username": "ann",
		"sender_name": "Ann", "deleted": true, "deleted_at": 1757145760,
		"media_type": "document", "file_id": "BQACAgIAAx…",
		"created_at": "2026-09-01T09:14:02Z", "reply_to_message_id": 4470}}`)

	resp, err := c.Archive.Message(context.Background(), 90210).Value()
	if err != nil {
		t.Fatalf("Message: %v", err)
	}
	msg := resp.Data
	if msg.FirstName != "Ann" || msg.LastName != "Weber" || msg.SenderName != "Ann" {
		t.Errorf("names = %+v", msg)
	}
	if !msg.Deleted || msg.DeletedAt.At().Unix() != 1757145760 {
		t.Errorf("deletion = %v at %v", msg.Deleted, msg.DeletedAt.At())
	}
	if msg.ReplyToID != 0 || msg.ReplyToMessageID != 4470 {
		t.Errorf("reply = %d / %d", msg.ReplyToID, msg.ReplyToMessageID)
	}
}

// The example from the documentation, byte for byte.
func TestVaultStoreReturnsTheEntryID(t *testing.T) {
	c, _, _ := serve(t, `{"ok":true,"data":{"entry_id": 31, "recovery_codes": ["7K3QMX9TBA4F8ZRQW2NCHD6PVJ", "Q2W8ERT4YA9M3PKZX7CVBN5H1D", "H6J4K2NM8NP9Q3RS5T7VW1X0YZ", "A1B2C3D4E5F6G7H8J9KM0NPQRS"]}}`)

	resp, err := c.Vault.Store(context.Background(), VaultStoreRequest{Passphrase: "the pale blue dot", Plaintext: "AKIA…"}).Value()
	if err != nil {
		t.Fatalf("Vault.Store: %v", err)
	}
	codes := resp.Data
	if codes.EntryID != 31 || len(codes.RecoveryCodes) != 4 {
		t.Errorf("codes = %+v", codes)
	}
}

// The new passphrase is required now, and a required field is a value sent
// whatever it holds: refusing an empty one is the API's decision to make.
func TestVaultRecoverAlwaysSendsTheNewPassphrase(t *testing.T) {
	c, _, sent := serve(t, `{"ok":true,"data":{}}`)

	if err := c.Vault.Recover(context.Background(), VaultRecoverRequest{Code: "7K3QM-X9TBA"}).Error(); err != nil {
		t.Fatalf("Vault.Recover: %v", err)
	}
	if want := `{"code":"7K3QM-X9TBA","new_passphrase":""}`; sent() != want {
		t.Errorf("body = %s, want %s", sent(), want)
	}
}

// The example from the documentation, byte for byte.
func TestActionUpdateSwitchesArgsRequired(t *testing.T) {
	c, last, sent := serve(t, `{"ok":true,"data":{"id": 3, "name": "order", "body": "Order [[ARG1]] is ready, [[ARG2]].",
 "min_args": 2, "args_required": true, "uses": 42}}`)

	resp, err := c.Actions.Update(context.Background(), 3, ActionUpdateRequest{
		Body:         ptr("Order [[ARG1]] is ready, [[ARG2]]."),
		ArgsRequired: ptr(true),
	}).Value()
	if err != nil {
		t.Fatalf("Actions.Update: %v", err)
	}
	action := resp.Data
	if last().Method != http.MethodPatch || last().URL.Path != "/v1/actions/3" {
		t.Errorf("request = %s %s", last().Method, last().URL.Path)
	}
	if want := `{"body":"Order [[ARG1]] is ready, [[ARG2]].","args_required":true}`; sent() != want {
		t.Errorf("body = %s, want %s", sent(), want)
	}
	if action.MinArgs != 2 || !action.ArgsRequired {
		t.Errorf("action = %+v", action)
	}
}

func TestPortraitCarriesThePlanSections(t *testing.T) {
	c, _, _ := serve(t, `{"ok":true,"data":{
		"chat_id": -1001234567890,
		"archetype": {"key": "the_essayist", "label": "The Essayist",
		              "description": "Long, careful sentences.", "fit": 0.81, "terms": ["draft"]},
		"traits": [{"key": "verbosity", "name": "Verbosity", "level": "high",
		            "note": "well above most", "score": 0.78, "z": 1.24}],
		"activity": {"messages": 4471, "first": "2026-01-04T08:00:00Z", "last": "2026-09-06T08:00:00Z"},
		"similar": [{"title": "Bob", "score": 0.72}],
		"signals": [{"kind": "drift", "key": "style_shift", "text": "The style shifted.", "severity": "notice"}],
		"confidence": {"level": "high", "score": 0.86}}}`)

	resp, err := c.Insights.Portrait(context.Background(), -1001234567890).Value()
	if err != nil {
		t.Fatalf("Insights.Portrait: %v", err)
	}
	p := resp.Data
	if p.Archetype.Description == "" || len(p.Archetype.Terms) != 1 || p.Traits[0].Note == "" {
		t.Errorf("archetype = %+v, traits = %+v", p.Archetype, p.Traits)
	}
	if p.Activity.First.IsZero() || len(p.Similar) != 1 || p.Signals[0].Severity != "notice" {
		t.Errorf("portrait = %+v", p)
	}
}

func TestSettingsCarryTheirChoices(t *testing.T) {
	c, _, _ := serve(t, `{"ok":true,"data":{"mode": "rotate", "ttl_seconds": 2592000, "in_chat": false,
		"can_ttl": true, "can_in_chat": true,
		"ttl_choices": [0, 86400, 604800, 2592000, 7776000, 15552000, 31536000]}}`)
	r, _, err := c.Settings.Retention(context.Background())
	if err != nil {
		t.Fatalf("Settings.Retention: %v", err)
	}
	if len(r.TTLChoices) != 7 || r.TTLChoices[0] != 0 || r.TTLChoices[3] != r.TTLSeconds {
		t.Errorf("retention = %+v", r)
	}

	c, _, _ = serve(t, `{"ok":true,"data":{"reveal_ttl_seconds": 60, "auto_deletes": true, "can_reveal_ttl": true, "reveal_ttl_choices": [15, 30, 60, 180, 300, 600, 1800]}}`)
	v, _, err := c.Settings.Vault(context.Background())
	if err != nil {
		t.Fatalf("Settings.Vault: %v", err)
	}
	if len(v.RevealTTLChoices) != 7 || v.RevealTTLChoices[2] != v.RevealTTLSeconds {
		t.Errorf("vault settings = %+v", v)
	}
}

func TestPathIDAndQuery(t *testing.T) {
	for _, tc := range []struct {
		name       string
		call       func(*Client) error
		path, rawQ string
	}{
		{"message by id",
			func(c *Client) error { return c.Archive.Message(context.Background(), 90210).Error() },
			"/v1/messages/90210", ""},
		{"versions",
			func(c *Client) error { return c.Archive.Versions(context.Background(), 90210).Error() },
			"/v1/messages/90210/versions", ""},
		{"negative chat id",
			func(c *Client) error {
				return c.Insights.Portrait(context.Background(), -1001234567890).Error()
			},
			"/v1/portraits/-1001234567890", ""},
		{"usage window",
			func(c *Client) error {
				return c.App.Usage(context.Background(), &UsageRequest{Days: 7, ByDay: true}).Error()
			},
			"/v1/usage", "by_day=true&days=7"},
		{"usage defaults",
			func(c *Client) error { return c.App.Usage(context.Background(), nil).Error() },
			"/v1/usage", ""},
		{"chats page",
			func(c *Client) error {
				return c.Archive.Chats(context.Background(), &ChatsRequest{Page: 2}).Error()
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

	resp, err := c.Vault.Delete(context.Background(), VaultDeleteRequest{Passphrase: "the pale blue dot"}).Value()
	if err != nil {
		t.Fatalf("Vault.Delete: %v", err)
	}
	meta := resp.Meta
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

	resp, err := c.Archive.Export(context.Background(), SearchRequest{}).Value()
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	body, meta := resp.Data, resp.Meta
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

	err := c.Vault.Reveal(context.Background(), VaultRevealRequest{Passphrase: "wrong"}).Error()
	if !IsCode(err, CodeNotFound) {
		t.Fatalf("err = %v, want not_found", err)
	}
}

func ptr[T any](v T) *T { return &v }
