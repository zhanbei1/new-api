package sdk

// Page holds a paginated list response.
type Page[T any] struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
	Items    []T `json:"items"`
}

// User is the dashboard self user DTO.
type User struct {
	ID              int    `json:"id"`
	Username        string `json:"username"`
	DisplayName     string `json:"display_name"`
	Role            int    `json:"role"`
	Status          int    `json:"status"`
	Email           string `json:"email"`
	GitHubID        string `json:"github_id"`
	DiscordID       string `json:"discord_id"`
	OidcID          string `json:"oidc_id"`
	WeChatID        string `json:"wechat_id"`
	TelegramID      string `json:"telegram_id"`
	LinuxDoID       string `json:"linux_do_id"`
	Group           string `json:"group"`
	Quota           int    `json:"quota"`
	UsedQuota       int    `json:"used_quota"`
	RequestCount    int    `json:"request_count"`
	AffCode         string `json:"aff_code"`
	AffCount        int    `json:"aff_count"`
	AffQuota        int    `json:"aff_quota"`
	AffHistoryQuota int    `json:"aff_history_quota"`
	InviterID       int    `json:"inviter_id"`
}

// LoginResult is returned by password or WeChat login.
//
// Newer New API builds return access_token + nested user.
// Legacy session builds (e.g. v1.0.0-rc.10) return flat user fields and set a
// session cookie instead — the SDK upgrades that to a PAT via GET /api/user/token.
type LoginResult struct {
	AccessToken     string `json:"access_token"`
	TokenType       string `json:"token_type"`
	AccessExpiresAt int64  `json:"access_expires_at"`
	User            User   `json:"user"`
	Require2FA      bool   `json:"require_2fa,omitempty"`
	FlowToken       string `json:"flow_token,omitempty"`

	// Legacy flat user fields (pre JWT-session login response).
	ID          int    `json:"id,omitempty"`
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Role        int    `json:"role,omitempty"`
	Status      int    `json:"status,omitempty"`
	Group       string `json:"group,omitempty"`
}

// DefaultToken is returned by GetAPIKey.
type DefaultToken struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Key  string `json:"key"`
}

// TokenUsage matches GET /api/usage/token/ data.
type TokenUsage struct {
	Object             string          `json:"object"`
	Name               string          `json:"name"`
	TotalGranted       int             `json:"total_granted"`
	TotalUsed          int             `json:"total_used"`
	TotalAvailable     int             `json:"total_available"`
	UnlimitedQuota     bool            `json:"unlimited_quota"`
	ModelLimits        map[string]bool `json:"model_limits"`
	ModelLimitsEnabled bool            `json:"model_limits_enabled"`
	ExpiresAt          int64           `json:"expires_at"`
}

// TopUp is a recharge / payment record.
type TopUp struct {
	ID              int     `json:"id"`
	UserID          int     `json:"user_id"`
	Amount          int64   `json:"amount"`
	Money           float64 `json:"money"`
	TradeNo         string  `json:"trade_no"`
	PaymentMethod   string  `json:"payment_method"`
	PaymentProvider string  `json:"payment_provider"`
	CreateTime      int64   `json:"create_time"`
	CompleteTime    int64   `json:"complete_time"`
	Status          string  `json:"status"`
}

// AuthProviders summarizes which login integrations are enabled.
type AuthProviders struct {
	WeChatEnabled            bool
	GitHubEnabled            bool
	DiscordEnabled           bool
	OIDCEnabled              bool
	TelegramEnabled          bool
	LinuxDoEnabled           bool
	PasskeyEnabled           bool
	RegisterEnabled          bool
	EmailVerificationEnabled bool
	CustomProviders          []CustomOAuthProvider
}

// CustomOAuthProvider is a custom OAuth app from /api/status.
type CustomOAuthProvider struct {
	ID                    int    `json:"id"`
	Name                  string `json:"name"`
	Slug                  string `json:"slug"`
	Icon                  string `json:"icon"`
	ClientID              string `json:"client_id"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	Scopes                string `json:"scopes"`
}

// Ticket statuses.
const (
	TicketStatusOpen    = "open"
	TicketStatusReplied = "replied"
	TicketStatusClosed  = "closed"
)

// Ticket is a support ticket.
type Ticket struct {
	ID        int    `json:"id"`
	UserID    int    `json:"user_id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// TicketReply is a message on a ticket thread.
type TicketReply struct {
	ID        int    `json:"id"`
	TicketID  int    `json:"ticket_id"`
	UserID    int    `json:"user_id"`
	IsAdmin   bool   `json:"is_admin"`
	Content   string `json:"content"`
	ParentID  *int   `json:"parent_id"`
	CreatedAt int64  `json:"created_at"`
}

// TicketDetail includes replies.
type TicketDetail struct {
	Ticket
	Replies []TicketReply `json:"replies"`
}

// SubscriptionSelf is GET /api/subscription/self data (kept loosely typed).
type SubscriptionSelf map[string]any

// SubscriptionPlanItem is one entry from GET /api/subscription/plans.
type SubscriptionPlanItem map[string]any

// SubscriptionGroupModels is one active subscription paired with models
// available under its upgrade_group for the current user.
type SubscriptionGroupModels struct {
	SubscriptionID int      `json:"subscription_id"`
	PlanID         int      `json:"plan_id"`
	UpgradeGroup   string   `json:"upgrade_group"`
	Models         []string `json:"models"`
}
