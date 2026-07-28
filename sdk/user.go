package sdk

import "context"

// GetSelf returns the current user profile.
func (c *Client) GetSelf(ctx context.Context) (*User, error) {
	var user User
	if err := c.doAuthed(ctx, "GET", "/api/user/self", nil, nil, &user); err != nil {
		return nil, err
	}
	return &user, nil
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
