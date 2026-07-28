package sdk

import "context"

// GetAPIKey returns the user's default (oldest) API key plaintext.
// Requires the user to already have a token (e.g. GENERATE_DEFAULT_TOKEN=true on register).
func (c *Client) GetAPIKey(ctx context.Context) (*DefaultToken, error) {
	var token DefaultToken
	if err := c.doAuthed(ctx, "GET", "/api/token/default", nil, nil, &token); err != nil {
		return nil, err
	}
	return &token, nil
}
