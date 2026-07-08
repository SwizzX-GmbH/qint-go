package qint

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultBaseURL is the production Qint merchant API endpoint. A dedicated
// api.qint.ch host is planned; override it with WithBaseURL when it lands.
const DefaultBaseURL = "https://qint-api.fly.dev/api/v1"

const defaultTimeout = 30 * time.Second

// userAgent is sent on every request so the API can attribute SDK traffic.
var userAgent = "qint-go/" + Version

// Client is a typed client for the Qint merchant API. A Client is safe for
// concurrent use by multiple goroutines. Construct one with NewClient.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// Option configures a Client. Pass options to NewClient.
type Option func(*Client)

// WithBaseURL overrides the API base URL (default DefaultBaseURL). A trailing
// slash is trimmed.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(baseURL, "/")
	}
}

// WithHTTPClient sets a custom *http.Client, e.g. to configure proxies or a
// custom transport. A nil client is ignored. Overrides WithTimeout.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// WithTimeout sets the per-request timeout on the default HTTP client
// (default 30s). Ignored if WithHTTPClient supplied a client of your own.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = d
	}
}

// NewClient returns a Client authenticated with the given API key
// (qk_live_...). Every request carries "Authorization: Bearer <apiKey>".
func NewClient(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:     apiKey,
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// do performs an HTTP request against the API and decodes a successful JSON
// response into out (when non-nil). Non-2xx responses are returned as
// *QintError.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body, out any) error {
	var reqBody io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(raw)
	}

	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newQintError(resp.StatusCode, data)
	}

	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return err
		}
	}
	return nil
}
