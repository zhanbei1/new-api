package sdk

import "context"

// GetAffCode returns the user's invite code, generating one on the server if empty.
func (c *Client) GetAffCode(ctx context.Context) (string, error) {
	var code string
	if err := c.doAuthed(ctx, "GET", "/api/user/aff", nil, nil, &code); err != nil {
		return "", err
	}
	return code, nil
}
