package sdk

import (
	"context"
	"strings"
)

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

// ListActiveSubscriptionModels maps each active subscription's upgrade_group
// to the models currently available to the user for that group.
// Subscriptions without upgrade_group are skipped. Groups the user cannot use
// yet return an empty Models slice (server-side filter).
func (c *Client) ListActiveSubscriptionModels(ctx context.Context) ([]SubscriptionGroupModels, error) {
	var data struct {
		Subscriptions []struct {
			Subscription *struct {
				ID           int    `json:"id"`
				PlanID       int    `json:"plan_id"`
				UpgradeGroup string `json:"upgrade_group"`
			} `json:"subscription"`
		} `json:"subscriptions"`
	}
	if err := c.doAuthed(ctx, "GET", "/api/subscription/self", nil, nil, &data); err != nil {
		return nil, err
	}

	byGroup := make(map[string][]string)
	out := make([]SubscriptionGroupModels, 0)
	for _, item := range data.Subscriptions {
		if item.Subscription == nil {
			continue
		}
		group := strings.TrimSpace(item.Subscription.UpgradeGroup)
		if group == "" {
			continue
		}
		models, ok := byGroup[group]
		if !ok {
			var err error
			models, err = c.GetUserModels(ctx, group)
			if err != nil {
				return nil, err
			}
			byGroup[group] = models
		}
		out = append(out, SubscriptionGroupModels{
			SubscriptionID: item.Subscription.ID,
			PlanID:         item.Subscription.PlanID,
			UpgradeGroup:   group,
			Models:         models,
		})
	}
	return out, nil
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
