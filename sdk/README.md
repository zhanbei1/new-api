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
| `Register` / `Login` | `/api/user/register`, `/api/user/login` |
| `DeleteSelf` | `/api/user/self` |
| `GetSelf` / `UpdateSelf` | `/api/user/self` |
| `GetAffCode` | `/api/user/aff` |
| `ListTopUps` | `/api/user/topup/self` |
| `GetTopUpInfo` / `RequestAmount` / `RequestEpay` | `/api/user/topup/info`, `/amount`, `/pay` |
| `RequestStripePay` / `RequestCreemPay` | `/api/user/stripe/pay`, `/creem/pay` |
| `WaitTopUpByTradeNo` | polls `/api/user/topup/self` |
| `ListAuthProviders` | `/api/status` |
| `WeChatLogin` | `/api/oauth/wechat` |
| `CreateOAuthState` | `/api/oauth/state` |
| `GetAPIKey` | `/api/token/default` |
| `GetTokenUsage` | `/api/usage/token/` |
| Subscription helpers | `/api/subscription/*` |
| Ticket helpers | `/api/ticket/self*` |

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
