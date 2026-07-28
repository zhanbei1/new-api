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
| `ListAuthProviders` | `/api/status` |
| `WeChatLogin` | `/api/oauth/wechat` |
| `CreateOAuthState` | `/api/oauth/state` |
| `GetAPIKey` | `/api/token/default` |
| `GetTokenUsage` | `/api/usage/token/` |
| Subscription helpers | `/api/subscription/*` |
| Ticket helpers | `/api/ticket/self*` |

This SDK does **not** wrap `/v1` chat completions. Use your preferred OpenAI-compatible client with the key from `GetAPIKey`.

## Notes

- WeChat login/register requires `wechat_login` enabled; WeChat auto-register does not attach invite codes.
- `GetAPIKey` errors if the user has no token yet.
- Management calls use JWT (`Authorization: Bearer <access_token>`). Usage calls use the API key.
