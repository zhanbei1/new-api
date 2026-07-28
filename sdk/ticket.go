package sdk

import (
	"context"
	"net/url"
	"strconv"
)

func mapQuery(k, v string) url.Values {
	q := url.Values{}
	if v != "" {
		q.Set(k, v)
	}
	return q
}

// ListTicketsParams filters the current user's tickets.
type ListTicketsParams struct {
	Page     int
	PageSize int
	Status   string
}

// CreateTicket creates a support ticket for the current user.
func (c *Client) CreateTicket(ctx context.Context, title, content string) (*Ticket, error) {
	var ticket Ticket
	err := c.doAuthed(ctx, "POST", "/api/ticket/", nil, map[string]string{
		"title":   title,
		"content": content,
	}, &ticket)
	if err != nil {
		return nil, err
	}
	return &ticket, nil
}

// ListTickets lists the current user's tickets.
func (c *Client) ListTickets(ctx context.Context, params ListTicketsParams) (*Page[Ticket], error) {
	q := url.Values{}
	if params.Page > 0 {
		q.Set("p", strconv.Itoa(params.Page))
	}
	if params.PageSize > 0 {
		q.Set("page_size", strconv.Itoa(params.PageSize))
	}
	if params.Status != "" {
		q.Set("status", params.Status)
	}
	var page Page[Ticket]
	if err := c.doAuthed(ctx, "GET", "/api/ticket/self", q, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// GetTicket returns ticket detail including replies.
func (c *Client) GetTicket(ctx context.Context, id int) (*TicketDetail, error) {
	var detail TicketDetail
	path := "/api/ticket/self/" + strconv.Itoa(id)
	if err := c.doAuthed(ctx, "GET", path, nil, nil, &detail); err != nil {
		return nil, err
	}
	return &detail, nil
}

// ReplyTicketRequest is a user reply payload.
type ReplyTicketRequest struct {
	Content  string `json:"content"`
	ParentID *int   `json:"parent_id,omitempty"`
}

// ReplyTicket appends a user reply to a ticket.
func (c *Client) ReplyTicket(ctx context.Context, id int, req ReplyTicketRequest) (*TicketReply, error) {
	var reply TicketReply
	path := "/api/ticket/self/" + strconv.Itoa(id) + "/replies"
	if err := c.doAuthed(ctx, "POST", path, nil, req, &reply); err != nil {
		return nil, err
	}
	return &reply, nil
}

// CloseTicket closes the current user's ticket.
func (c *Client) CloseTicket(ctx context.Context, id int) error {
	path := "/api/ticket/self/" + strconv.Itoa(id) + "/close"
	return c.doAuthed(ctx, "PUT", path, nil, nil, nil)
}

// DeleteTicket deletes the current user's ticket and its replies.
func (c *Client) DeleteTicket(ctx context.Context, id int) error {
	path := "/api/ticket/self/" + strconv.Itoa(id)
	return c.doAuthed(ctx, "DELETE", path, nil, nil, nil)
}
