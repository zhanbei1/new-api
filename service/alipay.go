package service

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
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
		setting.AlipayPublicKey,
		setting.AlipayAppPublicCert,
		setting.AlipayRootCert,
		setting.AlipayPublicCert,
		fmt.Sprintf("%v", setting.AlipayIsProduction),
		setting.AlipayEncryptKey,
	}, "|")
}

func alipayCertModeConfigured() bool {
	return strings.TrimSpace(setting.AlipayAppPublicCert) != "" &&
		strings.TrimSpace(setting.AlipayRootCert) != "" &&
		strings.TrimSpace(setting.AlipayPublicCert) != ""
}

func alipayPublicKeyModeConfigured() bool {
	return strings.TrimSpace(setting.AlipayPublicKey) != ""
}

// GetAlipayClient builds (or reuses) an Alipay client from current settings.
// Prefers public-key mode when AlipayPublicKey is set; otherwise uses certificate mode.
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

	switch {
	case alipayPublicKeyModeConfigured():
		// Public-key mode and certificate mode are mutually exclusive in the SDK.
		// Prefer explicit 支付宝公钥 when set so RSA-key users are not blocked by
		// leftover mis-pasted "cert" fields that still contain 应用公钥 material.
		publicKey, err := prepareAlipayPlatformPublicKey(setting.AlipayPublicKey)
		if err != nil {
			return nil, fmt.Errorf("load Alipay public key: %w", err)
		}
		if err := client.LoadAliPayPublicKey(publicKey); err != nil {
			return nil, fmt.Errorf("load Alipay public key: %w", wrapAlipayPublicKeyError(publicKey, err))
		}
	case alipayCertModeConfigured():
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
	default:
		return nil, fmt.Errorf("Alipay credentials incomplete: configure either certificate mode (app/root/alipay certs) or public-key mode (支付宝公钥)")
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

// IsAlipayConfigured reports whether certificate-mode or public-key-mode Alipay credentials are present.
func IsAlipayConfigured() bool {
	if strings.TrimSpace(setting.AlipayAppId) == "" || strings.TrimSpace(setting.AlipayPrivateKey) == "" {
		return false
	}
	return alipayCertModeConfigured() || alipayPublicKeyModeConfigured()
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

func stripAlipayKeyLabel(raw string) string {
	s := normalizeAlipayPEMNewlines(raw)
	for _, prefix := range []string{
		"支付宝公钥：", "支付宝公钥:", "支付宝公钥",
		"应用公钥：", "应用公钥:", "应用公钥",
		"应用私钥：", "应用私钥:", "应用私钥",
		"公钥：", "公钥:", "公钥",
		"私钥：", "私钥:", "私钥",
	} {
		if strings.HasPrefix(s, prefix) {
			s = strings.TrimSpace(strings.TrimPrefix(s, prefix))
		}
	}
	return s
}

func decodeAlipayKeyDER(raw string) ([]byte, error) {
	s := normalizeAlipayPEMMaterial(stripAlipayKeyLabel(raw))
	if s == "" {
		return nil, fmt.Errorf("empty key material")
	}
	if strings.HasPrefix(s, "-") {
		block, _ := pem.Decode([]byte(s))
		if block == nil {
			return nil, fmt.Errorf("invalid PEM block")
		}
		return block.Bytes, nil
	}
	if m := len(s) % 4; m != 0 {
		s += strings.Repeat("=", 4-m)
	}
	der, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 key material: %w", err)
	}
	return der, nil
}

// prepareAlipayPlatformPublicKey accepts Alipay key-tool / console public keys in
// raw PKCS#1 or PKIX base64 (with or without PEM headers) and returns PKIX PEM
// that smartwalle/alipay LoadAliPayPublicKey can parse.
func prepareAlipayPlatformPublicKey(raw string) (string, error) {
	der, err := decodeAlipayKeyDER(raw)
	if err != nil {
		return "", wrapAlipayPublicKeyError(raw, err)
	}
	if _, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return "", fmt.Errorf("value is an RSA private key; paste 支付宝公钥 from Open Platform, not 应用私钥")
	}
	if _, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		return "", fmt.Errorf("value is a private key; paste 支付宝公钥 from Open Platform, not 应用私钥")
	}
	if pub, err := x509.ParsePKIXPublicKey(der); err == nil {
		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return "", fmt.Errorf("Alipay public key must be an RSA public key")
		}
		return encodeAlipayPKIXPublicKey(rsaPub), nil
	}
	if pub, err := x509.ParsePKCS1PublicKey(der); err == nil {
		return encodeAlipayPKIXPublicKey(pub), nil
	}
	return "", wrapAlipayPublicKeyError(raw, fmt.Errorf("asn1: unrecognized RSA public key encoding"))
}

func encodeAlipayPKIXPublicKey(pub *rsa.PublicKey) string {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return ""
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
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

func wrapAlipayPublicKeyError(publicKey string, err error) error {
	if err == nil {
		return nil
	}
	upper := strings.ToUpper(publicKey)
	if strings.Contains(upper, "BEGIN CERTIFICATE") ||
		strings.Contains(upper, "BEGIN PRIVATE KEY") ||
		strings.Contains(upper, "BEGIN RSA PRIVATE KEY") ||
		strings.Contains(err.Error(), "asn1:") ||
		strings.Contains(err.Error(), "invalid") ||
		strings.Contains(err.Error(), "unrecognized") {
		return fmt.Errorf("value is not a usable RSA public key string; paste 支付宝公钥 from Open Platform (接口加签方式), not 应用私钥 and not a .crt certificate. Raw base64 without BEGIN/END is OK: %w", err)
	}
	return err
}

func wrapAlipayCertError(filename, certPEM string, err error) error {
	if err == nil {
		return nil
	}
	upper := strings.ToUpper(certPEM)
	if strings.Contains(upper, "BEGIN PUBLIC KEY") ||
		strings.Contains(upper, "BEGIN RSA PUBLIC KEY") ||
		strings.Contains(upper, "BEGIN PRIVATE KEY") ||
		strings.Contains(upper, "BEGIN RSA PRIVATE KEY") ||
		strings.Contains(err.Error(), "malformed serial number") {
		return fmt.Errorf("value is not an X.509 certificate; paste the downloaded %s content (BEGIN CERTIFICATE), not 应用公钥/私钥. Or clear the three cert fields and use public-key mode with 支付宝公钥: %w", filename, err)
	}
	return err
}
