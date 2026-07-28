package sdk

import (
	"context"
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
func (c *Client) Register(ctx context.Context, req RegisterRequest) error {
	return c.doPublic(ctx, "POST", "/api/user/register", nil, req, nil)
}

// Login authenticates with username/password and stores the access token on the client.
func (c *Client) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	var result LoginResult
	err := c.doPublic(ctx, "POST", "/api/user/login", nil, map[string]string{
		"username": username,
		"password": password,
	}, &result)
	if err != nil {
		return nil, err
	}
	if result.AccessToken != "" {
		c.AccessToken = result.AccessToken
	}
	return &result, nil
}

// WeChatLogin logs in or registers via WeChat verification code.
// WeChat registration does not support aff codes (server limitation).
func (c *Client) WeChatLogin(ctx context.Context, code string) (*LoginResult, error) {
	var result LoginResult
	q := mapQuery("code", code)
	err := c.doPublic(ctx, "GET", "/api/oauth/wechat", q, nil, &result)
	if err != nil {
		return nil, err
	}
	if result.AccessToken != "" {
		c.AccessToken = result.AccessToken
	}
	return &result, nil
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
