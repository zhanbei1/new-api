package sdk

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

// Client is a lightweight management API client for New API.
type Client struct {
	BaseURL     string
	HTTPClient  *http.Client
	AccessToken string
}

// NewClient creates a client pointing at baseURL (e.g. https://example.com).
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetAccessToken stores the JWT used for UserAuth endpoints.
func (c *Client) SetAccessToken(token string) {
	c.AccessToken = token
}

type apiEnvelope struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	Code    bool            `json:"code"` // used by /api/usage/token/
}

// APIError is returned when the server responds with success=false or HTTP error.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("http status %d", e.StatusCode)
}

func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, body any, authBearer string, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	full := c.BaseURL + path
	if len(query) > 0 {
		full += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, full, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if authBearer != "" {
		req.Header.Set("Authorization", "Bearer "+authBearer)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(raw))}
	}
	var env apiEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return err
	}
	// usage endpoint uses code:true; others use success:true
	ok := env.Success || env.Code
	if !ok {
		msg := env.Message
		if msg == "" {
			msg = "request failed"
		}
		return &APIError{StatusCode: resp.StatusCode, Message: msg}
	}
	if out == nil || len(env.Data) == 0 || string(env.Data) == "null" {
		return nil
	}
	return json.Unmarshal(env.Data, out)
}

func (c *Client) doAuthed(ctx context.Context, method, path string, query url.Values, body, out any) error {
	if c.AccessToken == "" {
		return &APIError{Message: "access token is required; call Login first"}
	}
	return c.doJSON(ctx, method, path, query, body, c.AccessToken, out)
}

func (c *Client) doPublic(ctx context.Context, method, path string, query url.Values, body, out any) error {
	return c.doJSON(ctx, method, path, query, body, "", out)
}
