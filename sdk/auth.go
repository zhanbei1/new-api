package sdk

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// RegisterRequest is the password registration payload.
type RegisterRequest struct {
	Username         string `json:"username"`
	Password         string `json:"password"`
	Email            string `json:"email,omitempty"`
	VerificationCode string `json:"verification_code,omitempty"`
	AffCode          string `json:"aff_code,omitempty"`
}

// Register creates a new user account.
// If the server has GENERATE_DEFAULT_TOKEN=true, a default API key is created
// (not returned here; call Login then GetAPIKey).
// When the server enables email verification, set Email + VerificationCode
// (obtain the code via SendEmailVerification first).
func (c *Client) Register(ctx context.Context, req RegisterRequest) error {
	return c.doPublic(ctx, "POST", "/api/user/register", nil, req, nil)
}

// SendEmailVerification sends a registration email verification code to email.
// Endpoint: GET /api/verification?email=...&turnstile=...
// Turnstile is required when the server has turnstile_check enabled (see ListAuthProviders / GetStatus).
func (c *Client) SendEmailVerification(ctx context.Context, email, turnstile string) error {
	q := mapQuery("email", email)
	if turnstile != "" {
		q.Set("turnstile", turnstile)
	}
	return c.doPublic(ctx, "GET", "/api/verification", q, nil, nil)
}

// BindEmail binds an email to the current logged-in user using a verification code
// previously sent via SendEmailVerification (same purpose code).
func (c *Client) BindEmail(ctx context.Context, email, code string) error {
	return c.doAuthed(ctx, "POST", "/api/oauth/email/bind", nil, map[string]string{
		"email": email,
		"code":  code,
	}, nil)
}

// Login authenticates with username/password and stores the access token on the client.
//
// Supports two New API auth modes:
//  1. New (JWT session): login data contains access_token.
//  2. Legacy (cookie session): login data is a flat user object; the SDK keeps the
//     session cookie, calls GET /api/user/token to mint a PAT, and uses that as AccessToken.
func (c *Client) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	c.ensureCookieJar()
	var result LoginResult
	cookies, err := c.doJSONCookies(ctx, "POST", "/api/user/login", nil, map[string]string{
		"username": username,
		"password": password,
	}, "", nil, &result)
	if err != nil {
		return nil, err
	}
	if err := c.finalizeLogin(ctx, &result, cookies); err != nil {
		return nil, err
	}
	return &result, nil
}

// WeChatLogin logs in or registers via WeChat verification code.
// WeChat registration does not support aff codes (server limitation).
func (c *Client) WeChatLogin(ctx context.Context, code string) (*LoginResult, error) {
	c.ensureCookieJar()
	var result LoginResult
	q := mapQuery("code", code)
	cookies, err := c.doJSONCookies(ctx, "GET", "/api/oauth/wechat", q, nil, "", nil, &result)
	if err != nil {
		return nil, err
	}
	if err := c.finalizeLogin(ctx, &result, cookies); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) finalizeLogin(ctx context.Context, result *LoginResult, loginCookies []*http.Cookie) error {
	if result == nil {
		return &APIError{Message: "empty login result"}
	}
	if result.Require2FA {
		return &APIError{Message: "account requires 2FA; complete /api/user/login/2fa with flow_token"}
	}
	normalizeLoginUser(result)

	if token := strings.TrimSpace(result.AccessToken); token != "" {
		c.AccessToken = token
		if result.User.ID > 0 {
			c.UserID = result.User.ID
		}
		// Dashboard JWT access tokens expire in ~15 minutes. Prefer a long-lived
		// user access token (PAT) for OpenOcta / SDK sessions that outlive that window.
		if pat, err := c.mintUserAccessToken(ctx, loginCookies); err == nil {
			if p := strings.TrimSpace(pat); p != "" {
				result.AccessToken = p
				c.AccessToken = p
				result.AccessExpiresAt = 0
			}
		}
		return nil
	}

	// Legacy session-cookie login: mint a personal access token while the jar holds the session.
	if result.User.ID <= 0 {
		return &APIError{Message: "login succeeded but response has neither access_token nor user id (unsupported auth shape)"}
	}
	c.UserID = result.User.ID
	pat, err := c.mintUserAccessToken(ctx, loginCookies)
	if err != nil {
		return fmt.Errorf("legacy session login: mint access token: %w", err)
	}
	if strings.TrimSpace(pat) == "" {
		return &APIError{Message: "legacy session login: empty access token from /api/user/token"}
	}
	result.AccessToken = pat
	if result.TokenType == "" {
		result.TokenType = "Bearer"
	}
	c.AccessToken = pat
	return nil
}

func normalizeLoginUser(result *LoginResult) {
	if result.User.ID > 0 {
		return
	}
	if result.ID <= 0 {
		return
	}
	result.User = User{
		ID:          result.ID,
		Username:    result.Username,
		DisplayName: result.DisplayName,
		Role:        result.Role,
		Status:      result.Status,
		Group:       result.Group,
	}
}

// mintUserAccessToken calls GET /api/user/token (GenerateAccessToken).
// Auth may be the short-lived dashboard JWT (Authorization) and/or login session cookies.
func (c *Client) mintUserAccessToken(ctx context.Context, loginCookies []*http.Cookie) (string, error) {
	var token string
	auth := strings.TrimSpace(c.AccessToken)
	// Explicitly forward Set-Cookie from login: Go's cookiejar skips Secure cookies on HTTP,
	// and some deployments mark the gin session cookie Secure even behind plain http://.
	if _, err := c.doJSONCookies(ctx, "GET", "/api/user/token", nil, nil, auth, loginCookies, &token); err != nil {
		return "", err
	}
	return strings.TrimSpace(token), nil
}

// CreateOAuthState creates an OAuth flow token for browser redirect flows (GitHub, etc.).
func (c *Client) CreateOAuthState(ctx context.Context, provider, intent, aff string) (string, error) {
	body := map[string]string{
		"provider": provider,
		"intent":   intent,
	}
	if aff != "" {
		body["aff"] = aff
	}
	var data struct {
		FlowToken string `json:"flow_token"`
	}
	err := c.doPublic(ctx, "POST", "/api/oauth/state", nil, body, &data)
	if err != nil {
		return "", err
	}
	return data.FlowToken, nil
}
