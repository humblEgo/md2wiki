// Package confluence provides a typed client that wraps the Confluence Cloud REST API.
package confluence

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is a Confluence Cloud REST client.
type Client struct {
	baseURL    string
	email      string
	token      string
	httpClient *http.Client
}

// Option configures a Client. Options are applied by New in the order they are passed.
type Option func(*Client)

// WithHTTPClient sets the http.Client the Client uses, which is useful for injecting a
// test server's client or for overriding the default request timeout.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.httpClient = h }
}

// New creates a client that authenticates with HTTP Basic auth (email plus API token).
// baseURL is the Confluence Cloud site root, for example https://acme.atlassian.net;
// any trailing slash is trimmed so path building stays consistent.
func New(baseURL, email, token string, opts ...Option) *Client {
	c := &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		email:      email,
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// APIError represents a non-2xx HTTP response from the Confluence API, capturing the
// status code and the raw response body for diagnostics.
type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("confluence: HTTP %d: %s", e.Status, e.Body)
}

func (c *Client) v1(p string) string { return c.baseURL + "/wiki/rest/api" + p }
func (c *Client) v2(p string) string { return c.baseURL + "/wiki/api/v2" + p }

func (c *Client) basicAuth() string {
	return base64.StdEncoding.EncodeToString([]byte(c.email + ":" + c.token))
}

// do attaches the Basic auth header, sends the request, and returns the response body,
// status code, and an error. A non-2xx status is reported as an *APIError (with the body
// still returned so callers that want it do not have to re-read the response).
func (c *Client) do(req *http.Request) ([]byte, int, error) {
	req.Header.Set("Authorization", "Basic "+c.basicAuth())
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("confluence: request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("confluence: read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return data, resp.StatusCode, &APIError{Status: resp.StatusCode, Body: string(data)}
	}
	return data, resp.StatusCode, nil
}

// doJSON sends a request whose body (if reqBody is non-nil) is JSON-encoded and decodes the
// response into out. Decoding is skipped when out is nil or the response body is empty. The
// returned int is the HTTP status code, which is reported even when an error is returned so
// callers can branch on it (for example, treating 404 as "not found").
func (c *Client) doJSON(ctx context.Context, method, urlStr string, reqBody, out any) (int, error) {
	var rdr io.Reader
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return 0, fmt.Errorf("confluence: encode: %w", err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, urlStr, rdr)
	if err != nil {
		return 0, fmt.Errorf("confluence: new request: %w", err)
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	data, status, err := c.do(req)
	if err != nil {
		return status, err
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return status, fmt.Errorf("confluence: decode: %w", err)
		}
	}
	return status, nil
}
