package teal

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testClient(t *testing.T, h http.HandlerFunc, opts ...Option) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	c, err := New("aek_test", append([]Option{WithBaseURL(srv.URL), WithRetry(0, 0)}, opts...)...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestDoDecodesEnvelopeAndCostHeaders(t *testing.T) {
	var got *http.Request
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
		w.Header().Set("X-Aether-Billing", "credits")
		w.Header().Set("X-Aether-Cost", "1000")
		w.Header().Set("X-Aether-Balance", "4999000")
		w.Header().Set("X-Aether-Ops", "search:query, messages:read")
		io.WriteString(w, `{"ok":true,"data":{"total":37}}`)
	})

	type count struct {
		Total int `json:"total"`
	}
	out, meta, err := c.Do[count](context.Background(), Request{
		Method: http.MethodPost,
		Path:   "v1/messages/count",
		Body:   map[string]any{"conditions": []any{}},
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}

	if want := "/v1/messages/count"; got.URL.Path != want {
		t.Errorf("path = %q, want %q", got.URL.Path, want)
	}
	if want := "Bearer aek_test"; got.Header.Get("Authorization") != want {
		t.Errorf("Authorization = %q, want %q", got.Header.Get("Authorization"), want)
	}
	if out.Total != 37 {
		t.Errorf("total = %d, want 37", out.Total)
	}
	if meta.Billing != BillingCredits || meta.Cost != 1000 {
		t.Errorf("meta = %+v", meta)
	}
	if !meta.HasBalance || meta.Balance != 4_999_000 {
		t.Errorf("balance = %d (has %v)", meta.Balance, meta.HasBalance)
	}
	if len(meta.Ops) != 2 || meta.Ops[1] != "messages:read" {
		t.Errorf("ops = %v", meta.Ops)
	}
}

func TestRefusalBecomesError(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		io.WriteString(w, `{"ok":false,"error":{"code":"quota_exhausted","message":"spent",
			"limit":"search:daily","used":2000,"limit_value":2000,"reset_at":1757203200}}`)
	})

	_, _, err := c.Do[none](context.Background(), Request{Method: http.MethodGet, Path: "v1/app"})
	if !IsCode(err, CodeQuotaExhausted) {
		t.Fatalf("err = %v, want quota_exhausted", err)
	}

	e, _ := AsError(err)
	if e.Limit != "search:daily" || e.Used != 2000 {
		t.Errorf("error = %+v", e)
	}
	if want := time.Unix(1757203200, 0).UTC(); !e.ResetAt.Equal(want) {
		t.Errorf("reset_at = %v, want %v", e.ResetAt, want)
	}
}

func TestRetriesRateRefusalButNotASpentQuota(t *testing.T) {
	for _, tc := range []struct {
		code  string
		calls int
	}{
		{CodeRateLimited, 2},
		{CodeQuotaExhausted, 1},
	} {
		t.Run(tc.code, func(t *testing.T) {
			var calls int
			c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if calls > 1 {
					io.WriteString(w, `{"ok":true,"data":{}}`)
					return
				}
				w.WriteHeader(http.StatusTooManyRequests)
				io.WriteString(w, `{"ok":false,"error":{"code":"`+tc.code+`"}}`)
			}, WithRetry(2, time.Millisecond))

			c.Do[none](context.Background(), Request{Method: http.MethodGet, Path: "v1/app"})
			if calls != tc.calls {
				t.Errorf("calls = %d, want %d", calls, tc.calls)
			}
		})
	}
}

// A refusal that arrives with a success status would otherwise read as an
// empty answer, which is worse than an error.
func TestRefusalWithASuccessStatusIsStillAnError(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"ok":false,"error":{"code":"forbidden","message":"the plan does not open this"}}`)
	})

	_, _, err := c.App.Get(context.Background())
	if !IsCode(err, CodeForbidden) {
		t.Fatalf("err = %v, want forbidden", err)
	}
}

// A write may have taken effect before the process failed, so it is not sent
// again. A read is.
func TestServerFailureIsRetriedOnlyWhenItIsSafe(t *testing.T) {
	for _, tc := range []struct {
		name  string
		call  func(*Client) error
		calls int
	}{
		{"read", func(c *Client) error { _, _, err := c.App.Get(context.Background()); return err }, 3},
		{"query", func(c *Client) error {
			_, _, err := c.Archive.Count(context.Background(), SearchRequest{})
			return err
		}, 3},
		{"delete", func(c *Client) error { _, err := c.Actions.Delete(context.Background(), 3); return err }, 3},
		{"write", func(c *Client) error {
			_, _, err := c.Vault.Store(context.Background(), VaultStoreRequest{Passphrase: "p", Plaintext: "s"})
			return err
		}, 1},
		{"patch", func(c *Client) error {
			_, _, err := c.Settings.UpdateLanguage(context.Background(), LanguageUpdateRequest{Language: "en"})
			return err
		}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls int
			c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.WriteHeader(http.StatusInternalServerError)
				io.WriteString(w, `{"ok":false,"error":{"code":"internal","message":"boom"}}`)
			}, WithRetry(2, time.Millisecond))

			if err := tc.call(c); !IsCode(err, CodeInternal) {
				t.Fatalf("err = %v, want internal", err)
			}
			if calls != tc.calls {
				t.Errorf("calls = %d, want %d", calls, tc.calls)
			}
		})
	}
}

// A rate refusal is retried whatever the request does, because nothing was
// done.
func TestRateRefusalIsRetriedEvenOnAWrite(t *testing.T) {
	var calls int
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls > 1 {
			io.WriteString(w, `{"ok":true,"data":{}}`)
			return
		}
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
		io.WriteString(w, `{"ok":false,"error":{"code":"rate_limited"}}`)
	}, WithRetry(2, time.Millisecond))

	if _, _, err := c.Vault.Store(context.Background(), VaultStoreRequest{}); err != nil {
		t.Fatalf("Store: %v", err)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}

func TestParseRetryAfter(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want time.Duration
	}{
		{"", 0},
		{"5", 5 * time.Second},
		{"-3", 0},
		{"gibberish", 0},
		{time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat), 0},
	} {
		if got := parseRetryAfter(tc.in); got != tc.want {
			t.Errorf("parseRetryAfter(%q) = %s, want %s", tc.in, got, tc.want)
		}
	}

	// An HTTP-date in the future comes back as the wait until then.
	in := time.Now().Add(90 * time.Second).UTC().Format(http.TimeFormat)
	if got := parseRetryAfter(in); got < 80*time.Second || got > 90*time.Second {
		t.Errorf("parseRetryAfter(%q) = %s, want about 90s", in, got)
	}
}

func TestWithTimeoutDoesNotMutateTheCallersClient(t *testing.T) {
	mine := &http.Client{}
	c, err := New("aek_test", WithTimeout(time.Second), WithHTTPClient(mine))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if mine.Timeout != 0 {
		t.Errorf("caller's client was mutated: %s", mine.Timeout)
	}
	if c.http.Timeout != time.Second {
		t.Errorf("timeout = %s, want 1s — options must not depend on order", c.http.Timeout)
	}
}
