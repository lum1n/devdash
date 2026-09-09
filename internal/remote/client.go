package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to another devdash serve over HTTP.
type Client struct {
	Base string
	HTTP *http.Client
}

// New points at a remote API (http://host:port).
func New(base string) *Client {
	return &Client{
		Base: NormalizeURL(base),
		HTTP: &http.Client{Timeout: 20 * time.Second},
	}
}

// NormalizeURL adds http:// when missing.
func NormalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return strings.TrimRight(raw, "/")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	return u.String()
}

// Healthy reports /api/health ok.
func (c *Client) Healthy(ctx context.Context) bool {
	return c.Ping(ctx) == nil
}

// Ping hits /api/health with a short timeout.
func (c *Client) Ping(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("remote url is empty")
	}
	probe := *c
	probe.HTTP = &http.Client{Timeout: 2 * time.Second}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
	}
	var out map[string]any
	return probe.Do(ctx, http.MethodGet, "/api/health", nil, &out)
}

// Do sends JSON to path.
func (c *Client) Do(ctx context.Context, method, path string, body, dest any) error {
	if c == nil || c.Base == "" {
		return fmt.Errorf("remote url is empty")
	}
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.Base+path, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode >= 300 {
		var payload struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &payload)
		if payload.Error != "" {
			return fmt.Errorf("%s", payload.Error)
		}
		return fmt.Errorf("remote %s %s: %s", method, path, res.Status)
	}
	if dest == nil || len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, dest)
}
