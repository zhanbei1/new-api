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

// ListActiveSubscriptionModels returns models for each active subscription's
// upgrade_group (channels enabled for that group).
// Endpoint: GET /api/subscription/self/models
// This is NOT the same as GetUserModels, which unions UserUsableGroups.
func (c *Client) ListActiveSubscriptionModels(ctx context.Context) ([]SubscriptionGroupModels, error) {
	var items []SubscriptionGroupModels
	if err := c.doAuthed(ctx, "GET", "/api/subscription/self/models", nil, nil, &items); err != nil {
		return nil, err
	}
	if items == nil {
		return []SubscriptionGroupModels{}, nil
	}
	for i := range items {
		if items[i].Models == nil {
			items[i].Models = []string{}
		}
	}
	return items, nil
}

// GetActiveSubscriptionModels returns a deduplicated list of models unlocked by
// all active subscriptions' upgrade groups (post-purchase only).
func (c *Client) GetActiveSubscriptionModels(ctx context.Context) ([]string, error) {
	items, err := c.ListActiveSubscriptionModels(ctx)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	models := make([]string, 0)
	for _, item := range items {
		for _, name := range item.Models {
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			models = append(models, name)
		}
	}
	return models, nil
}

// RequestSubscriptionAlipayPay creates a native Alipay scan-to-pay order for a subscription plan.
// The plan must have allow_alipay enabled. Returns the same AlipayPayResult shape as wallet top-up.
func (c *Client) RequestSubscriptionAlipayPay(ctx context.Context, planID int) (*AlipayPayResult, error) {
	var result AlipayPayResult
	if err := c.doAuthed(ctx, "POST", "/api/subscription/alipay/pay", nil, map[string]int{
		"plan_id": planID,
	}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
