package service

import (
	"fmt"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/setting"
	"github.com/smartwalle/alipay/v3"
)

var (
	alipayClientMu sync.Mutex
	alipayClient   *alipay.Client
	alipayClientFP string
)

func alipayConfigFingerprint() string {
	return strings.Join([]string{
		setting.AlipayAppId,
		setting.AlipayPrivateKey,
		setting.AlipayAppPublicCert,
		setting.AlipayRootCert,
		setting.AlipayPublicCert,
		fmt.Sprintf("%v", setting.AlipayIsProduction),
		setting.AlipayEncryptKey,
	}, "|")
}

// GetAlipayClient builds (or reuses) an Alipay client from current settings.
func GetAlipayClient() (*alipay.Client, error) {
	fp := alipayConfigFingerprint()
	alipayClientMu.Lock()
	defer alipayClientMu.Unlock()
	if alipayClient != nil && alipayClientFP == fp {
		return alipayClient, nil
	}

	appId := strings.TrimSpace(setting.AlipayAppId)
	privateKey := strings.TrimSpace(setting.AlipayPrivateKey)
	if appId == "" || privateKey == "" {
		return nil, fmt.Errorf("Alipay AppId or private key is not configured")
	}
	client, err := alipay.New(appId, privateKey, setting.AlipayIsProduction)
	if err != nil {
		return nil, fmt.Errorf("create Alipay client: %w", err)
	}
	if err := client.LoadAppCertPublicKey(strings.TrimSpace(setting.AlipayAppPublicCert)); err != nil {
		return nil, fmt.Errorf("load Alipay app cert: %w", err)
	}
	if err := client.LoadAliPayRootCert(strings.TrimSpace(setting.AlipayRootCert)); err != nil {
		return nil, fmt.Errorf("load Alipay root cert: %w", err)
	}
	if err := client.LoadAlipayCertPublicKey(strings.TrimSpace(setting.AlipayPublicCert)); err != nil {
		return nil, fmt.Errorf("load Alipay public cert: %w", err)
	}
	if key := strings.TrimSpace(setting.AlipayEncryptKey); key != "" {
		if err := client.SetEncryptKey(key); err != nil {
			return nil, fmt.Errorf("set Alipay encrypt key: %w", err)
		}
	}
	alipayClient = client
	alipayClientFP = fp
	return client, nil
}

// IsAlipayConfigured reports whether certificate-mode Alipay credentials are present.
func IsAlipayConfigured() bool {
	return strings.TrimSpace(setting.AlipayAppId) != "" &&
		strings.TrimSpace(setting.AlipayPrivateKey) != "" &&
		strings.TrimSpace(setting.AlipayAppPublicCert) != "" &&
		strings.TrimSpace(setting.AlipayRootCert) != "" &&
		strings.TrimSpace(setting.AlipayPublicCert) != ""
}
