package sdk

import (
	"context"
	"encoding/json"
)

// GetStatus returns the raw /api/status payload.
func (c *Client) GetStatus(ctx context.Context) (map[string]any, error) {
	var data map[string]any
	if err := c.doPublic(ctx, "GET", "/api/status", nil, nil, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// ListAuthProviders extracts enabled auth integrations from /api/status.
func (c *Client) ListAuthProviders(ctx context.Context) (*AuthProviders, error) {
	data, err := c.GetStatus(ctx)
	if err != nil {
		return nil, err
	}
	out := &AuthProviders{
		WeChatEnabled:            asBool(data["wechat_login"]),
		GitHubEnabled:            asBool(data["github_oauth"]),
		DiscordEnabled:           asBool(data["discord_oauth"]),
		OIDCEnabled:              asBool(data["oidc_enabled"]),
		TelegramEnabled:          asBool(data["telegram_oauth"]),
		LinuxDoEnabled:           asBool(data["linuxdo_oauth"]),
		PasskeyEnabled:           asBool(data["passkey_login"]),
		RegisterEnabled:          asBool(data["register_enabled"]),
		EmailVerificationEnabled: asBool(data["email_verification"]),
		PhoneVerificationEnabled: asBool(data["phone_verification"]),
	}
	if raw, ok := data["custom_oauth_providers"]; ok && raw != nil {
		b, err := json.Marshal(raw)
		if err == nil {
			_ = json.Unmarshal(b, &out.CustomProviders)
		}
	}
	return out, nil
}

func asBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case string:
		return t == "true" || t == "1"
	default:
		return false
	}
}
