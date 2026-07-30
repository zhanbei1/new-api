package sdk

import (
	"context"
	"net/url"
)

// GetCheckinStatus returns check-in settings and stats for the current user.
// month is "YYYY-MM"; pass "" to use the server's current month.
// Endpoint: GET /api/user/checkin?month=...
func (c *Client) GetCheckinStatus(ctx context.Context, month string) (*CheckinStatus, error) {
	var q url.Values
	if month != "" {
		q = url.Values{"month": {month}}
	}
	var status CheckinStatus
	if err := c.doAuthed(ctx, "GET", "/api/user/checkin", q, nil, &status); err != nil {
		return nil, err
	}
	if status.Stats.Records == nil {
		status.Stats.Records = []CheckinRecord{}
	}
	return &status, nil
}

// DoCheckin performs today's check-in and awards quota.
// turnstile is required when the server has turnstile_check enabled
// (see ListAuthProviders / GetStatus); pass "" when disabled.
// Endpoint: POST /api/user/checkin?turnstile=...
func (c *Client) DoCheckin(ctx context.Context, turnstile string) (*CheckinResult, error) {
	var q url.Values
	if turnstile != "" {
		q = url.Values{"turnstile": {turnstile}}
	}
	var result CheckinResult
	if err := c.doAuthed(ctx, "POST", "/api/user/checkin", q, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
