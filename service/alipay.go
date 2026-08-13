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
	privateKey := normalizeAlipayPEMMaterial(setting.AlipayPrivateKey)
	if appId == "" || privateKey == "" {
		return nil, fmt.Errorf("Alipay AppId or private key is not configured")
	}
	client, err := alipay.New(appId, privateKey, setting.AlipayIsProduction)
	if err != nil {
		return nil, fmt.Errorf("create Alipay client: %w", wrapAlipayPrivateKeyError(privateKey, err))
	}
	appCert := normalizeAlipayPEMMaterial(setting.AlipayAppPublicCert)
	if err := client.LoadAppCertPublicKey(appCert); err != nil {
		return nil, fmt.Errorf("load Alipay app cert: %w", wrapAlipayCertError("appCertPublicKey_*.crt", appCert, err))
	}
	rootCert := normalizeAlipayPEMMaterial(setting.AlipayRootCert)
	if err := client.LoadAliPayRootCert(rootCert); err != nil {
		return nil, fmt.Errorf("load Alipay root cert: %w", err)
	}
	alipayCert := normalizeAlipayPEMMaterial(setting.AlipayPublicCert)
	if err := client.LoadAlipayCertPublicKey(alipayCert); err != nil {
		return nil, fmt.Errorf("load Alipay public cert: %w", wrapAlipayCertError("alipayCertPublicKey_RSA2.crt", alipayCert, err))
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

func normalizeAlipayPEMNewlines(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return s
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\\n", "\n")
	return strings.TrimSpace(s)
}

func stripAlipayBase64Whitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == ' ' || r == '\n' || r == '\t' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// normalizeAlipayPEMMaterial accepts PEM, raw base64, wrapped lines, and literal
// "\n" sequences from JSON/env pastes for Alipay keys and certificates.
func normalizeAlipayPEMMaterial(raw string) string {
	s := normalizeAlipayPEMNewlines(raw)
	if s == "" || strings.HasPrefix(s, "-") {
		return s
	}
	return stripAlipayBase64Whitespace(s)
}

func wrapAlipayPrivateKeyError(privateKey string, err error) error {
	if err == nil {
		return nil
	}
	upper := strings.ToUpper(privateKey)
	if strings.Contains(upper, "BEGIN PUBLIC KEY") ||
		strings.Contains(upper, "BEGIN RSA PUBLIC KEY") ||
		strings.Contains(upper, "BEGIN CERTIFICATE") ||
		strings.Contains(err.Error(), "asn1:") {
		return fmt.Errorf("application private key is not a valid PKCS#1/PKCS#8 RSA private key; paste the Alipay key tool 应用私钥, not 应用公钥, 支付宝公钥, or a certificate: %w", err)
	}
	return err
}

func wrapAlipayCertError(filename, pem string, err error) error {
	if err == nil {
		return nil
	}
	upper := strings.ToUpper(pem)
	if strings.Contains(upper, "BEGIN PUBLIC KEY") ||
		strings.Contains(upper, "BEGIN RSA PUBLIC KEY") ||
		strings.Contains(upper, "BEGIN PRIVATE KEY") ||
		strings.Contains(upper, "BEGIN RSA PRIVATE KEY") ||
		strings.Contains(err.Error(), "malformed serial number") {
		return fmt.Errorf("value is not an X.509 certificate; paste the downloaded %s content (BEGIN CERTIFICATE), not 应用公钥/私钥: %w", filename, err)
	}
	return err
}
