package sdk

import "context"

// ListSubscriptionPlans returns available subscription plans.
func (c *Client) ListSubscriptionPlans(ctx context.Context) ([]SubscriptionPlanItem, error) {
	var plans []SubscriptionPlanItem
	if err := c.doAuthed(ctx, "GET", "/api/subscription/plans", nil, nil, &plans); err != nil {
		return nil, err
	}
	return plans, nil
}

// GetSubscriptionSelf returns the current user's subscriptions.
func (c *Client) GetSubscriptionSelf(ctx context.Context) (SubscriptionSelf, error) {
	var data SubscriptionSelf
	if err := c.doAuthed(ctx, "GET", "/api/subscription/self", nil, nil, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// UpdateSubscriptionPreference updates billing preference.
func (c *Client) UpdateSubscriptionPreference(ctx context.Context, preference string) error {
	return c.doAuthed(ctx, "PUT", "/api/subscription/self/preference", nil, map[string]string{
		"billing_preference": preference,
	}, nil)
}
