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
		http:       &http.Client{Timeout: 30 * time.Second},
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

	c.App = &AppService{c: c}
	c.Archive = &ArchiveService{c: c}
	c.Vault = &VaultService{c: c}
	c.Actions = &ActionsService{c: c}
	c.Insights = &InsightsService{c: c}
	c.Settings = &SettingsService{c: c}
	return c, nil
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
