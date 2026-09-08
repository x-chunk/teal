package teal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Request is one call to the API, before it becomes an *http.Request.
type Request struct {
	Method string     // http.MethodGet, http.MethodPost, …
	Path   string     // relative to the base URL, e.g. "v1/messages/search"
	Query  url.Values // optional
	Body   any        // marshalled as JSON when not nil

	// Idempotent says that sending this request twice does what sending it
	// once does, which is what decides whether it may be retried after a
	// failure on the server's side. Reads set it; so do the archive's
	// queries, which are POSTs only because a query does not fit in a URL.
	//
	// A write leaves it false and is never retried on a 500: the request
	// may well have taken effect before the process failed, and a second
	// vault entry is worse than an error. A rate refusal is retried either
	// way, because nothing was done.
	Idempotent bool
}

// Do sends req and decodes the envelope's data field into a T.
//
// The *Meta is nil only when the request never reached the API — a transport
// failure, or a body that would not marshal. Every answer, refusal included,
// carries one.
//
// T is in the result and not in the arguments, so it is always given
// explicitly:
//
//	app, meta, err := c.Do[Application](ctx, Request{Method: http.MethodGet, Path: "v1/app"})
//
// Services reach for the get/post/patch/del helpers on base instead, which
// are this with the boilerplate folded away.
func (c *Client) Do[T any](ctx context.Context, req Request) (T, *Meta, error) {
	var out T
	resp, meta, err := c.send(ctx, req)
	if err != nil {
		return out, meta, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return out, meta, fmt.Errorf("teal: reading %s %s: %w", req.Method, req.Path, err)
	}

	var env struct {
		OK   bool            `json:"ok"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return out, meta, fmt.Errorf("teal: decoding %s %s: %w", req.Method, req.Path, err)
	}
	// A refusal normally arrives with a status to match, and send has turned
	// it into an *Error already. One that arrives with a 2xx would otherwise
	// read as an empty answer, which is worse than an error.
	if !env.OK {
		return out, meta, errorFromBody(raw, resp.StatusCode, meta)
	}
	if len(env.Data) == 0 || string(env.Data) == "null" {
		return out, meta, nil
	}
	if err := json.Unmarshal(env.Data, &out); err != nil {
		return out, meta, fmt.Errorf("teal: decoding %s %s: %w", req.Method, req.Path, err)
	}
	return out, meta, nil
}

// DoRaw sends req and hands back the undecoded body, for the one endpoint
// that answers with a document instead of the envelope — the export. The
// caller closes it.
func (c *Client) DoRaw(ctx context.Context, req Request) (io.ReadCloser, *Meta, error) {
	resp, meta, err := c.send(ctx, req)
	if err != nil {
		return nil, meta, err
	}
	return resp.Body, meta, nil
}

// send performs the request, retrying what is worth retrying, and turns a
// refusal into an *Error. On a nil error the body is open and undrained.
func (c *Client) send(ctx context.Context, req Request) (*http.Response, *Meta, error) {
	var body []byte
	if req.Body != nil {
		var err error
		if body, err = json.Marshal(req.Body); err != nil {
			return nil, nil, fmt.Errorf("teal: encoding %s %s: %w", req.Method, req.Path, err)
		}
	}

	for attempt := 0; ; attempt++ {
		httpReq, err := c.newRequest(ctx, req, body)
		if err != nil {
			return nil, nil, err
		}

		resp, err := c.http.Do(httpReq)
		if err != nil {
			// No answer at all, so nothing was charged.
			err = fmt.Errorf("teal: %s %s: %w", req.Method, req.Path, err)
			if attempt >= c.maxRetries || ctx.Err() != nil {
				return nil, nil, err
			}
			if werr := wait(ctx, c.backoff(attempt, 0)); werr != nil {
				return nil, nil, werr
			}
			continue
		}

		meta := metaFrom(resp)
		if resp.StatusCode < 300 {
			return resp, meta, nil
		}

		apiErr := decodeError(resp, meta)
		if attempt >= c.maxRetries || !apiErr.retryable(req.Idempotent) {
			return nil, meta, apiErr
		}
		if err := wait(ctx, c.backoff(attempt, apiErr.RetryAfter)); err != nil {
			return nil, meta, err
		}
	}
}

func (c *Client) newRequest(ctx context.Context, req Request, body []byte) (*http.Request, error) {
	u := c.baseURL.JoinPath(req.Path)
	if len(req.Query) > 0 {
		u.RawQuery = req.Query.Encode()
	}

	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, u.String(), r)
	if err != nil {
		return nil, fmt.Errorf("teal: building %s %s: %w", req.Method, req.Path, err)
	}

	switch c.auth {
	case AuthBare:
		httpReq.Header.Set("Authorization", c.apiKey)
	case AuthCustomHeader:
		httpReq.Header.Set("X-Aether-Key", c.apiKey)
	default:
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	return httpReq, nil
}

// backoff prefers what the server asked for, and doubles its own wait
// otherwise.
func (c *Client) backoff(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		return retryAfter
	}
	if c.retryBase <= 0 {
		return 0
	}
	return time.Duration(float64(c.retryBase) * math.Pow(2, float64(attempt)))
}

func wait(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// BillingMode is the mode an application is on, as the API reports it.
type BillingMode string

const (
	BillingShared  BillingMode = "shared"  // the plan pays, nothing is charged
	BillingCredits BillingMode = "credits" // the application's balance pays
	BillingHybrid  BillingMode = "hybrid"  // the plan pays, credits cover the overflow
)

// Credits are millionths of a US dollar: 1000000 credits is one dollar and
// 10000 is one cent.
type Credits int64

// Dollars renders credits as the amount of money they are.
func (c Credits) Dollars() float64 { return float64(c) / 1_000_000 }

func (c Credits) String() string { return "$" + strconv.FormatFloat(c.Dollars(), 'f', -1, 64) }

// Meta is what a response cost, read from the X-Aether-* headers, so a client
// can meter itself without a second request. It is returned even when the
// call is refused: the API refunds before it answers, so the balance here is
// the balance actually left.
//
// A method returns a nil *Meta only when the request never reached the API.
type Meta struct {
	StatusCode int

	Billing BillingMode // the mode this request was served under
	Cost    Credits     // zero under shared limits and for free operations

	// Balance is what is left on the application's balance. It is absent
	// under shared limits, where there is no balance in play — HasBalance
	// says whether the header was there at all.
	Balance    Credits
	HasBalance bool

	Ops        []string      // the operations this request was billed for
	QuotaReset time.Time     // when a refusing quota's window turns
	RetryAfter time.Duration // how long to wait, on a rate refusal

	// Header is the whole response header, for anything not lifted above —
	// X-Aether-Export-Total, for one.
	Header http.Header
}

func metaFrom(resp *http.Response) *Meta {
	h := resp.Header
	m := &Meta{
		StatusCode: resp.StatusCode,
		Billing:    BillingMode(h.Get("X-Aether-Billing")),
		Header:     h,
	}
	if n, err := strconv.ParseInt(h.Get("X-Aether-Cost"), 10, 64); err == nil {
		m.Cost = Credits(n)
	}
	if n, err := strconv.ParseInt(h.Get("X-Aether-Balance"), 10, 64); err == nil {
		m.Balance, m.HasBalance = Credits(n), true
	}
	for _, op := range strings.Split(h.Get("X-Aether-Ops"), ",") {
		if op = strings.TrimSpace(op); op != "" {
			m.Ops = append(m.Ops, op)
		}
	}
	if n, err := strconv.ParseInt(h.Get("X-Aether-Quota-Reset"), 10, 64); err == nil {
		m.QuotaReset = time.Unix(n, 0).UTC()
	}
	m.RetryAfter = parseRetryAfter(h.Get("Retry-After"))
	return m
}

// parseRetryAfter reads the header in both of the forms RFC 9110 allows: a
// count of seconds, and an HTTP-date.
func parseRetryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}
	if n, err := strconv.Atoi(v); err == nil {
		if n < 0 {
			return 0
		}
		return time.Duration(n) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

// maxBody caps what is read from one answer that is not the export.
const maxBody = 8 << 20
