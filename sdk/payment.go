package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"
)

// Top-up order statuses (server-side).
const (
	TopUpStatusPending = "pending"
	TopUpStatusSuccess = "success"
	TopUpStatusFailed  = "failed"
	TopUpStatusExpired = "expired"
)

// PayMethod describes an enabled online payment method from topup info.
type PayMethod struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // e.g. alipay, wxpay
	Color    string `json:"color,omitempty"`
	MinTopup string `json:"min_topup,omitempty"`
	Icon     string `json:"icon,omitempty"`
}

// TopUpInfo is GET /api/user/topup/info.
type TopUpInfo struct {
	EnableOnlineTopUp       bool               `json:"enable_online_topup"`
	EnableStripeTopUp       bool               `json:"enable_stripe_topup"`
	EnableCreemTopUp        bool               `json:"enable_creem_topup"`
	EnableAlipayTopUp       bool               `json:"enable_alipay_topup"`
	EnableWaffoTopUp        bool               `json:"enable_waffo_topup"`
	EnableWaffoPancakeTopUp bool               `json:"enable_waffo_pancake_topup"`
	PayMethods              []PayMethod        `json:"pay_methods"`
	MinTopup                int                `json:"min_topup"`
	StripeMinTopup          int                `json:"stripe_min_topup"`
	AlipayMinTopup          int                `json:"alipay_min_topup"`
	AmountOptions           []int              `json:"amount_options"`
	Discount                map[string]float64 `json:"discount"`
	EnableRedemption        bool               `json:"enable_redemption"`
}

// GetTopUpInfo returns recharge configuration and enabled pay methods (alipay/wxpay for scan pay).
func (c *Client) GetTopUpInfo(ctx context.Context) (*TopUpInfo, error) {
	var info TopUpInfo
	if err := c.doAuthed(ctx, "GET", "/api/user/topup/info", nil, nil, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// RequestAmount asks the server for the payable money string for a quota amount (epay).
func (c *Client) RequestAmount(ctx context.Context, amount int64) (string, error) {
	var money string
	if err := c.doAuthed(ctx, "POST", "/api/user/amount", nil, map[string]int64{
		"amount": amount,
	}, &money); err != nil {
		return "", err
	}
	return money, nil
}

// EpayPayRequest creates an epay (alipay/wxpay) checkout order.
type EpayPayRequest struct {
	Amount        int64  `json:"amount"`
	PaymentMethod string `json:"payment_method"` // alipay | wxpay | custom type from PayMethods
}

// EpayPayResult is returned by RequestEpay.
// The server does not return a QR image URL; open SubmitURL with a form POST of Params
// to the epay cashier, which then shows the Alipay/WeChat QR code.
type EpayPayResult struct {
	URL    string            `json:"url"`  // cashier submit URL (.../submit.php)
	Params map[string]string `json:"data"` // form fields including out_trade_no, money, sign, ...
}

// OutTradeNo returns the merchant trade number (same as TopUp.TradeNo).
func (r *EpayPayResult) OutTradeNo() string {
	if r == nil || r.Params == nil {
		return ""
	}
	return r.Params["out_trade_no"]
}

// RequestEpay creates an online top-up order via epay for scan-to-pay (alipay/wxpay).
func (c *Client) RequestEpay(ctx context.Context, req EpayPayRequest) (*EpayPayResult, error) {
	raw, err := c.doAuthedRaw(ctx, "POST", "/api/user/pay", nil, req)
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
		URL     string          `json:"url"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	if envelope.Message != "success" {
		msg := "payment request failed"
		var errText string
		if err := json.Unmarshal(envelope.Data, &errText); err == nil && errText != "" {
			msg = errText
		} else if envelope.Message != "" && envelope.Message != "error" {
			msg = envelope.Message
		}
		return nil, &APIError{Message: msg}
	}
	params := map[string]string{}
	if len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		if err := json.Unmarshal(envelope.Data, &params); err != nil {
			return nil, err
		}
	}
	return &EpayPayResult{URL: envelope.URL, Params: params}, nil
}

// EpayCheckoutFormHTML builds an auto-submit HTML form that opens the epay cashier
// (where the user scans Alipay/WeChat QR). Write this to a file or serve it in a browser.
func EpayCheckoutFormHTML(result *EpayPayResult) string {
	if result == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>Redirecting to payment</title></head><body>")
	b.WriteString("<form id=\"pay\" method=\"POST\" action=\"")
	b.WriteString(html.EscapeString(result.URL))
	b.WriteString("\">")
	for k, v := range result.Params {
		b.WriteString("<input type=\"hidden\" name=\"")
		b.WriteString(html.EscapeString(k))
		b.WriteString("\" value=\"")
		b.WriteString(html.EscapeString(v))
		b.WriteString("\">")
	}
	b.WriteString("</form><script>document.getElementById('pay').submit();</script>")
	b.WriteString("<p>Redirecting to payment...</p></body></html>")
	return b.String()
}

// WaitTopUpOptions controls polling ListTopUps for a trade_no.
type WaitTopUpOptions struct {
	Interval time.Duration
	Timeout  time.Duration
}

// WaitTopUpByTradeNo polls until the top-up with tradeNo reaches success/failed/expired or timeout.
func (c *Client) WaitTopUpByTradeNo(ctx context.Context, tradeNo string, opts WaitTopUpOptions) (*TopUp, error) {
	if tradeNo == "" {
		return nil, &APIError{Message: "trade_no is required"}
	}
	interval := opts.Interval
	if interval <= 0 {
		interval = 2 * time.Second
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	deadline := time.Now().Add(timeout)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		page, err := c.ListTopUps(ctx, ListTopUpsParams{Page: 1, PageSize: 20, Keyword: tradeNo})
		if err != nil {
			return nil, err
		}
		for i := range page.Items {
			item := &page.Items[i]
			if item.TradeNo != tradeNo {
				continue
			}
			switch item.Status {
			case TopUpStatusSuccess, TopUpStatusFailed, TopUpStatusExpired:
				return item, nil
			}
		}
		if time.Now().After(deadline) {
			return nil, &APIError{Message: fmt.Sprintf("timeout waiting for trade_no %s", tradeNo)}
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

// StripePayRequest creates a Stripe Checkout session.
type StripePayRequest struct {
	Amount        int64  `json:"amount"`
	PaymentMethod string `json:"payment_method"` // usually "stripe"
	SuccessURL    string `json:"success_url,omitempty"`
	CancelURL     string `json:"cancel_url,omitempty"`
}

// StripePayResult contains the hosted checkout link.
type StripePayResult struct {
	PayLink string `json:"pay_link"`
}

// RequestStripePay creates a Stripe checkout session and returns pay_link.
func (c *Client) RequestStripePay(ctx context.Context, req StripePayRequest) (*StripePayResult, error) {
	if req.PaymentMethod == "" {
		req.PaymentMethod = "stripe"
	}
	var result StripePayResult
	if err := c.doAuthed(ctx, "POST", "/api/user/stripe/pay", nil, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreemPayRequest purchases a Creem product.
type CreemPayRequest struct {
	ProductID     string `json:"product_id"`
	PaymentMethod string `json:"payment_method"` // usually "creem"
}

// CreemPayResult contains Creem checkout URL.
type CreemPayResult struct {
	CheckoutURL string `json:"checkout_url"`
	OrderID     string `json:"order_id"`
}

// RequestCreemPay creates a Creem checkout session.
func (c *Client) RequestCreemPay(ctx context.Context, req CreemPayRequest) (*CreemPayResult, error) {
	if req.PaymentMethod == "" {
		req.PaymentMethod = "creem"
	}
	var result CreemPayResult
	if err := c.doAuthed(ctx, "POST", "/api/user/creem/pay", nil, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// AlipayPayRequest creates a native Alipay scan-to-pay top-up order.
type AlipayPayRequest struct {
	Amount int64 `json:"amount"`
}

// AlipayPayResult contains the merchant trade number and Alipay cashier QR URL.
// Render QRCode as a QR image (unlike epay, which returns an HTML form redirect).
type AlipayPayResult struct {
	TradeNo  string `json:"trade_no"`
	QRCode   string `json:"qr_code"`
	ExpireAt int64  `json:"expire_at,omitempty"`
}

// RequestAlipayPay creates a native Alipay TradePreCreate order and returns qr_code.
func (c *Client) RequestAlipayPay(ctx context.Context, req AlipayPayRequest) (*AlipayPayResult, error) {
	var result AlipayPayResult
	if err := c.doAuthed(ctx, "POST", "/api/user/alipay/pay", nil, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
