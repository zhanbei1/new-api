package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client is a lightweight management API client for New API.
type Client struct {
	BaseURL     string
	HTTPClient  *http.Client
	AccessToken string
	// UserID is required by legacy New API builds that expect New-Api-User.
	// Newer JWT session builds ignore the header; set it whenever known.
	UserID int
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

// SetAccessToken stores the bearer token used for UserAuth endpoints
// (dashboard JWT on newer servers, or user PAT on legacy session servers).
func (c *Client) SetAccessToken(token string) {
	c.AccessToken = token
}

// SetUserID stores the New API numeric user id for the New-Api-User header.
func (c *Client) SetUserID(id int) {
	c.UserID = id
}

func (c *Client) ensureCookieJar() {
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	if c.HTTPClient.Jar == nil {
		jar, err := cookiejar.New(nil)
		if err == nil {
			c.HTTPClient.Jar = jar
		}
	}
}

type apiEnvelope struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	// Code is bool on /api/usage/token/, numeric/string on some auth error responses.
	Code flexibleCode `json:"code"`
}

// flexibleCode accepts bool, number, or string JSON values for the "code" field.
type flexibleCode struct {
	Truthy bool
	Raw    json.RawMessage
}

func (c *flexibleCode) UnmarshalJSON(b []byte) error {
	c.Raw = append(json.RawMessage(nil), b...)
	c.Truthy = false
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	var asBool bool
	if err := json.Unmarshal(b, &asBool); err == nil {
		c.Truthy = asBool
		return nil
	}
	var asNum float64
	if err := json.Unmarshal(b, &asNum); err == nil {
		// Auth errors often use HTTP status codes (e.g. 401); usage API uses 1/true.
		// Only treat literal 1 as success-like truthy for the usage endpoint.
		c.Truthy = asNum == 1
		return nil
	}
	var asStr string
	if err := json.Unmarshal(b, &asStr); err == nil {
		c.Truthy = asStr == "true" || asStr == "1"
		return nil
	}
	// Unknown shape: ignore without failing the whole envelope decode.
	return nil
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
	_, err := c.doJSONCookies(ctx, method, path, query, body, authBearer, nil, out)
	return err
}

// doJSONCookies is doJSON plus optional extra Cookie header values and returned Set-Cookie list.
func (c *Client) doJSONCookies(ctx context.Context, method, path string, query url.Values, body any, authBearer string, extraCookies []*http.Cookie, out any) ([]*http.Cookie, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(b)
	}
	full := c.BaseURL + path
	if len(query) > 0 {
		full += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, full, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if authBearer != "" {
		req.Header.Set("Authorization", "Bearer "+authBearer)
	}
	if c.UserID > 0 {
		req.Header.Set("New-Api-User", strconv.Itoa(c.UserID))
	}
	for _, ck := range extraCookies {
		if ck == nil || ck.Name == "" {
			continue
		}
		req.AddCookie(ck)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.Cookies(), &APIError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(raw))}
	}
	var env apiEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return resp.Cookies(), err
	}
	// success:true (standard), code truthy (usage), message:"success" (payment gateways)
	ok := env.Success || env.Code.Truthy || env.Message == "success"
	if !ok {
		msg := paymentErrorMessage(env)
		return resp.Cookies(), &APIError{StatusCode: resp.StatusCode, Message: msg}
	}
	if out == nil || len(env.Data) == 0 || string(env.Data) == "null" {
		return resp.Cookies(), nil
	}
	return resp.Cookies(), json.Unmarshal(env.Data, out)
}

func paymentErrorMessage(env apiEnvelope) string {
	if env.Message == "error" && len(env.Data) > 0 {
		var s string
		if err := json.Unmarshal(env.Data, &s); err == nil && s != "" {
			return s
		}
	}
	if env.Message != "" && env.Message != "error" {
		return env.Message
	}
	if env.Message == "error" {
		return "payment request failed"
	}
	return "request failed"
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

// doAuthedRaw returns the raw JSON body after auth (for responses with extra top-level fields).
func (c *Client) doAuthedRaw(ctx context.Context, method, path string, query url.Values, body any) ([]byte, error) {
	if c.AccessToken == "" {
		return nil, &APIError{Message: "access token is required; call Login first"}
	}
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(b)
	}
	full := c.BaseURL + path
	if len(query) > 0 {
		full += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, full, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	if c.UserID > 0 {
		req.Header.Set("New-Api-User", strconv.Itoa(c.UserID))
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(raw))}
	}
	return raw, nil
}
