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
