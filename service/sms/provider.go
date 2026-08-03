package sms

import (
	"context"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const (
	ProviderTypeAlibaba = "alibaba"
)

// Provider sends SMS messages through a specific vendor.
type Provider interface {
	Send(ctx context.Context, phone, templateCode string, params map[string]string) error
}

// GetProvider returns the configured SMS provider.
func GetProvider() (Provider, error) {
	providerType := strings.TrimSpace(common.SMSProviderType)
	if providerType == "" {
		providerType = ProviderTypeAlibaba
	}
	switch providerType {
	case ProviderTypeAlibaba:
		return newAlibabaProvider()
	default:
		return nil, fmt.Errorf("unsupported SMS provider type: %s", providerType)
	}
}

// SendVerificationCode sends a verification code using the configured template.
func SendVerificationCode(ctx context.Context, phone, code string) error {
	provider, err := GetProvider()
	if err != nil {
		return err
	}
	templateCode := strings.TrimSpace(common.SMSTemplateCode)
	if templateCode == "" {
		return fmt.Errorf("SMS template code is not configured")
	}
	return provider.Send(ctx, phone, templateCode, map[string]string{
		"code": code,
	})
}

// IsConfigured reports whether the minimum SMS settings are present.
func IsConfigured() bool {
	return strings.TrimSpace(common.SMSClientId) != "" &&
		strings.TrimSpace(common.SMSClientSecret) != "" &&
		strings.TrimSpace(common.SMSSignName) != "" &&
		strings.TrimSpace(common.SMSTemplateCode) != ""
}
