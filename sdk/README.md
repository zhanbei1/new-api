# New API Go SDK

Lightweight Go client for New API user management endpoints.

## Install

```bash
go get github.com/QuantumNous/new-api/sdk@sdk/v0.1.0
```

Module path: `github.com/QuantumNous/new-api/sdk`  
Publish tags must use the nested form: `sdk/vX.Y.Z`.

## Requirements

- Server reachable at your deployment base URL
- For automatic API key after register: set `GENERATE_DEFAULT_TOKEN=true` on the server

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/QuantumNous/new-api/sdk"
)

func main() {
	ctx := context.Background()
	c := sdk.NewClient("https://your-host")

	providers, err := c.ListAuthProviders(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("wechat enabled:", providers.WeChatEnabled)

	if err := c.Register(ctx, sdk.RegisterRequest{
		Username: "demo",
		Password: "password123",
		// AffCode: "abcd", // optional invite code
	}); err != nil {
		log.Fatal(err)
	}

	if _, err := c.Login(ctx, "demo", "password123"); err != nil {
		log.Fatal(err)
	}

	aff, _ := c.GetAffCode(ctx)
	fmt.Println("aff code:", aff)

	token, err := c.GetAPIKey(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("api key:", token.Key)

	usage, err := c.GetTokenUsage(ctx, token.Key)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("available:", usage.TotalAvailable)
}
```

## Covered APIs

| Method | Endpoint |
|--------|----------|
| `Register` / `Login` / `LoginSMS` | `/api/user/register`, `/api/user/login`, `/api/user/login/sms` |
| `SendEmailVerification` / `BindEmail` | `/api/verification`, `/api/oauth/email/bind` |
| `SendSMSVerification` / `SendSMSLoginCode` / `BindPhone` | `/api/sms/verification`, `/api/sms/login`, `/api/oauth/phone/bind` |
| `DeleteSelf` | `/api/user/self` |
| `GetSelf` / `UpdateSelf` | `/api/user/self` |
| `GetAffCode` | `/api/user/aff` |
| `ListTopUps` | `/api/user/topup/self` |
| `GetTopUpInfo` / `RequestAmount` / `RequestEpay` | `/api/user/topup/info`, `/amount`, `/pay` |
| `RequestStripePay` / `RequestCreemPay` | `/api/user/stripe/pay`, `/creem/pay` |
| `RequestAlipayPay` | `/api/user/alipay/pay` |
| `RequestSubscriptionAlipayPay` | `/api/subscription/alipay/pay` |
| `WaitTopUpByTradeNo` | polls `/api/user/topup/self` |
| `ListAuthProviders` | `/api/status` |
| `WeChatLogin` | `/api/oauth/wechat` |
| `CreateOAuthState` | `/api/oauth/state` |
| `GetAPIKey` | `/api/token/default` |
| `GetTokenUsage` | `/api/usage/token/` |
| `GetUserModels` | `/api/user/models` (UserUsableGroups; often broader than a plan) |
| `ListActiveSubscriptionModels` / `GetActiveSubscriptionModels` | `/api/subscription/self/models` |
| `GetCheckinStatus` / `DoCheckin` | `/api/user/checkin` |
| Subscription helpers | `/api/subscription/*` |
| Ticket helpers | `/api/ticket/self*` |

### Check-in (签到)

Requires the server check-in feature enabled. `DoCheckin` needs a Turnstile token
query param when `turnstile_check` is on.

```go
status, err := c.GetCheckinStatus(ctx, "") // current month
if err != nil {
	log.Fatal(err)
}
if status.Stats.CheckedInToday {
	fmt.Println("already checked in")
} else {
	res, err := c.DoCheckin(ctx, "") // or pass turnstile token
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("awarded:", res.QuotaAwarded, "date:", res.CheckinDate)
}
```

### Subscription → group → models (after purchase)

Models are not stored on the plan itself. Each active subscription has an
`upgrade_group`; the server returns models enabled on channels for that group
(`abilities`), not the full UserUsableGroups union from `/api/user/models`.

```go
// Per active subscription (subscription_id, plan_id, upgrade_group, models)
items, err := c.ListActiveSubscriptionModels(ctx)
if err != nil {
	log.Fatal(err)
}
for _, item := range items {
	fmt.Println(item.UpgradeGroup, item.Models)
}

// Deduplicated model list across all active upgrade groups
models, err := c.GetActiveSubscriptionModels(ctx)
if err != nil {
	log.Fatal(err)
}
fmt.Println(models)
```

`GetUserModels` is for playground usable-group listing and should not be used
as the source of truth for subscription unlocks.

### SMS login / phone bind

When `ListAuthProviders` reports `PhoneVerificationEnabled`:

```go
_ = c.SendSMSLoginCode(ctx, "13800138000", "")
res, err := c.LoginSMS(ctx, "13800138000", "123456")
// after login: c.BindPhone(ctx, phone, code) with SendSMSVerification
_ = res
```

Register with phone OTP (when required): set `Phone` + `PhoneVerificationCode` on `RegisterRequest`
(after `SendSMSVerification`).

### Native Alipay scan-to-pay

Unlike epay, native Alipay returns a `qr_code` string (Alipay cashier URL) to render as a QR image:

```go
info, _ := c.GetTopUpInfo(ctx)
if info.EnableAlipayTopUp {
	pay, err := c.RequestAlipayPay(ctx, sdk.AlipayPayRequest{Amount: 10})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("trade:", pay.TradeNo, "qr:", pay.QRCode)
	order, err := c.WaitTopUpByTradeNo(ctx, pay.TradeNo, sdk.WaitTopUpOptions{})
	_ = order
}

// Subscription (plan must have allow_alipay):
subPay, err := c.RequestSubscriptionAlipayPay(ctx, planID)
_ = subPay

// Or epay channel (payment_method = alipay | wxpay):
epayPay, err := c.RequestSubscriptionEpay(ctx, planID, "alipay")
_ = epayPay

// Or wallet balance redemption (plan allow_balance_pay):
_ = c.RequestSubscriptionBalancePay(ctx, planID)
```

### Scan-to-pay (Alipay / WeChat via epay)

The server does **not** return a QR image. Flow:

1. `GetTopUpInfo` — check `enable_online_topup` and `pay_methods` (`alipay` / `wxpay`)
2. `RequestAmount` — show payable money
3. `RequestEpay` — get cashier `url` + form `params`
4. Open `EpayCheckoutFormHTML(result)` in a browser (auto POST) → cashier shows QR
5. Optionally `WaitTopUpByTradeNo(result.OutTradeNo(), ...)` until `status=success`

```go
info, _ := c.GetTopUpInfo(ctx)
money, _ := c.RequestAmount(ctx, 10)
fmt.Println("payable:", money)

pay, err := c.RequestEpay(ctx, sdk.EpayPayRequest{Amount: 10, PaymentMethod: "alipay"})
if err != nil {
	log.Fatal(err)
}
html := sdk.EpayCheckoutFormHTML(pay)
_ = os.WriteFile("pay.html", []byte(html), 0644)
// open pay.html in browser for the user to scan

order, err := c.WaitTopUpByTradeNo(ctx, pay.OutTradeNo(), sdk.WaitTopUpOptions{})
if err != nil {
	log.Fatal(err)
}
fmt.Println("paid:", order.Status, order.Money)
```

This SDK does **not** wrap `/v1` chat completions. Use your preferred OpenAI-compatible client with the key from `GetAPIKey`.

## Notes

- WeChat login/register requires `wechat_login` enabled; WeChat auto-register does not attach invite codes.
- `GetAPIKey` errors if the user has no token yet.
- Management calls use JWT (`Authorization: Bearer <access_token>`). Usage calls use the API key.
- Online top-up requires payment compliance confirmed and epay configured on the server.
- `SendEmailVerification` / `DoCheckin` need a Turnstile token query param when the server enables `turnstile_check`; pass it as the respective argument.
