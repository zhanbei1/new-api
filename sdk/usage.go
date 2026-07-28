package sdk

import (
	"context"
	"strings"
)

// GetTokenUsage queries quota usage for an API key (sk-...).
func (c *Client) GetTokenUsage(ctx context.Context, apiKey string) (*TokenUsage, error) {
	key := strings.TrimSpace(apiKey)
	var usage TokenUsage
	if err := c.doJSON(ctx, "GET", "/api/usage/token/", nil, nil, key, &usage); err != nil {
		return nil, err
	}
	return &usage, nil
}
