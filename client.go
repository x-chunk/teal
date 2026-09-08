package teal

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to one deployment with one application key. It is safe for
// concurrent use and is meant to be made once and kept.
type Client struct {
	baseURL    *url.URL
	apiKey     string
	auth       AuthHeader
	http       *http.Client
	userAgent  string
	maxRetries int
	retryBase  time.Duration
	timeout    time.Duration

	// The API, grouped as the documentation groups it.
	App      *AppService
	Archive  *ArchiveService
	Vault    *VaultService
	Actions  *ActionsService
	Insights *InsightsService
	Settings *SettingsService
}

// New builds a client for the given application key — the one the bot showed
// once, when the application was created.
func New(apiKey string, opts ...Option) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("teal: empty api key")
	}

	c := &Client{
		apiKey:     apiKey,
		auth:       AuthBearer,
		http:       &http.Client{Transport: defaultTransport()},
		userAgent:  "teal-go/" + Version,
		maxRetries: 2,
		retryBase:  500 * time.Millisecond,
	}
	if err := WithBaseURL(DefaultBaseURL)(c); err != nil {
		return nil, err
	}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	if c.timeout > 0 {
		// Copied rather than set, so that a caller's own http.Client is not
		// mutated behind its back and the order of the options cannot matter.
		hc := *c.http
		hc.Timeout = c.timeout
		c.http = &hc
	}

	c.App = &AppService{base{c}}
	c.Archive = &ArchiveService{base{c}}
	c.Vault = &VaultService{base{c}}
	c.Actions = &ActionsService{base{c}}
	c.Insights = &InsightsService{base{c}}
	c.Settings = &SettingsService{base{c}}
	return c, nil
}

// defaultTransport bounds the parts of a request that can hang without
// bounding the whole of it. There is deliberately no Timeout on the
// http.Client: it would cover reading the body as well, and an export can be
// 50 MB. Give a call a deadline through its context, or WithTimeout.
func defaultTransport() http.RoundTripper {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.ResponseHeaderTimeout = 30 * time.Second
	t.ExpectContinueTimeout = 1 * time.Second
	return t
}

// Option configures a Client. Options are applied in order by New.
type Option func(*Client) error

// WithBaseURL points the client at a deployment. A path in raw is kept, so a
// host behind a prefix ("https://example.com/aether") works.
func WithBaseURL(raw string) Option {
	return func(c *Client) error {
		u, err := url.Parse(strings.TrimRight(raw, "/") + "/")
		if err != nil {
			return fmt.Errorf("teal: bad base url %q: %w", raw, err)
		}
		if u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("teal: base url %q needs a scheme and a host", raw)
		}
		c.baseURL = u
		return nil
	}
}

// WithHTTPClient replaces the underlying *http.Client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) error {
		if h == nil {
			return fmt.Errorf("teal: nil http client")
		}
		c.http = h
		return nil
	}
}

// WithTimeout puts a deadline on the whole of every request, the reading of
// the body included. It is off by default, because an export is a stream
// whose length is not known in advance and a deadline would tear it in half.
// A per-call context is the finer instrument.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) error {
		if d < 0 {
			return fmt.Errorf("teal: negative timeout %s", d)
		}
		c.timeout = d
		return nil
	}
}

// WithUserAgent prepends a caller's own identifier to the User-Agent.
func WithUserAgent(ua string) Option {
	return func(c *Client) error {
		c.userAgent = strings.TrimSpace(ua) + " " + c.userAgent
		return nil
	}
}

// WithRetry sets how many times a request is sent again after a rate refusal
// or a failure on the server's side. Zero disables retrying.
func WithRetry(attempts int, base time.Duration) Option {
	return func(c *Client) error {
		if attempts < 0 {
			return fmt.Errorf("teal: negative retry count %d", attempts)
		}
		c.maxRetries, c.retryBase = attempts, base
		return nil
	}
}

// WithHeaderAuth chooses how the key is sent. The API accepts a bearer token,
// a bare key and an X-Aether-Key header; they are the same credential.
func WithHeaderAuth(h AuthHeader) Option {
	return func(c *Client) error {
		c.auth = h
		return nil
	}
}

// AuthHeader names one of the ways the API accepts a key.
type AuthHeader int

const (
	// AuthBearer sends "Authorization: Bearer <key>". The default.
	AuthBearer AuthHeader = iota
	// AuthBare sends "Authorization: <key>".
	AuthBare
	// AuthCustomHeader sends "X-Aether-Key: <key>", for callers whose
	// configuration screen will not let them set an Authorization header.
	AuthCustomHeader
)
