package sdk

import (
	"context"
	"net/url"
)

// GetSelf returns the current user profile.
func (c *Client) GetSelf(ctx context.Context) (*User, error) {
	var user User
	if err := c.doAuthed(ctx, "GET", "/api/user/self", nil, nil, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserModels returns enabled models for playground UserUsableGroups.
// Pass a non-empty group to filter to that group (empty if not usable).
// Pass "" to union models across all usable groups — this is often broader
// than subscription unlocks. For models of purchased plans' upgrade_group,
// use ListActiveSubscriptionModels / GetActiveSubscriptionModels instead.
func (c *Client) GetUserModels(ctx context.Context, group string) ([]string, error) {
	var q url.Values
	if group != "" {
		q = url.Values{"group": {group}}
	}
	var models []string
	if err := c.doAuthed(ctx, "GET", "/api/user/models", q, nil, &models); err != nil {
		return nil, err
	}
	if models == nil {
		return []string{}, nil
	}
	return models, nil
}

// UpdateSelfRequest updates profile fields.
type UpdateSelfRequest struct {
	Username         string `json:"username,omitempty"`
	DisplayName      string `json:"display_name,omitempty"`
	Password         string `json:"password,omitempty"`
	OriginalPassword string `json:"original_password,omitempty"`
}

// UpdateSelf updates the current user profile.
func (c *Client) UpdateSelf(ctx context.Context, req UpdateSelfRequest) error {
	return c.doAuthed(ctx, "PUT", "/api/user/self", nil, req, nil)
}

// DeleteSelf soft-deletes (注销) the current user account.
func (c *Client) DeleteSelf(ctx context.Context) error {
	return c.doAuthed(ctx, "DELETE", "/api/user/self", nil, nil, nil)
}
