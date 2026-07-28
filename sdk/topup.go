package sdk

import (
	"context"
	"net/url"
	"strconv"
)

// ListTopUpsParams controls pagination for recharge history.
type ListTopUpsParams struct {
	Page     int
	PageSize int
	Keyword  string
}

// ListTopUps returns the current user's top-up / payment records.
func (c *Client) ListTopUps(ctx context.Context, params ListTopUpsParams) (*Page[TopUp], error) {
	q := url.Values{}
	if params.Page > 0 {
		q.Set("p", strconv.Itoa(params.Page))
	}
	if params.PageSize > 0 {
		q.Set("page_size", strconv.Itoa(params.PageSize))
	}
	if params.Keyword != "" {
		q.Set("keyword", params.Keyword)
	}
	var page Page[TopUp]
	if err := c.doAuthed(ctx, "GET", "/api/user/topup/self", q, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}
