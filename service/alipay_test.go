package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/setting"
	"github.com/smartwalle/alipay/v3"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAlipayPrivateKey(t *testing.T) {
	t.Parallel()

	pemKey := strings.TrimSpace(`
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEA
-----END RSA PRIVATE KEY-----
`)
	require.Equal(t, pemKey, normalizeAlipayPEMMaterial("  "+strings.ReplaceAll(pemKey, "\n", `\n`)+"  "))

	raw := "MIIEowIBAAKCAQEA00M40KY4rqqx"
	require.Equal(t, raw, normalizeAlipayPEMMaterial("  "+raw[:8]+"\n"+raw[8:]+" \t"))
}

func TestAlipayClientAcceptsWrappedPKCS1PrivateKey(t *testing.T) {
	t.Parallel()

	key := mustAlipayTestKey(t)
	raw := base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PrivateKey(key))
	var wrapped strings.Builder
	for i := 0; i < len(raw); i += 64 {
		end := i + 64
		if end > len(raw) {
			end = len(raw)
		}
		wrapped.WriteString(raw[i:end])
		wrapped.WriteByte('\n')
	}

	_, err := alipay.New("2021000000000000", normalizeAlipayPEMMaterial(wrapped.String()), false)
	require.NoError(t, err)
}

func TestWrapAlipayPrivateKeyErrorForPublicKey(t *testing.T) {
	t.Parallel()

	pubPEM := mustAlipayTestPublicKeyPEM(t)
	_, err := alipay.New("2021000000000000", pubPEM, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "asn1:")

	wrapped := wrapAlipayPrivateKeyError(pubPEM, err)
	require.ErrorContains(t, wrapped, "应用私钥")
	require.ErrorContains(t, wrapped, "asn1:")
}

func TestWrapAlipayCertErrorForPublicKey(t *testing.T) {
	t.Parallel()

	key := mustAlipayTestKey(t)
	pubPEM := mustAlipayTestPublicKeyPEM(t)
	client, err := alipay.New("2021000000000000", string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})), false)
	require.NoError(t, err)

	err = client.LoadAppCertPublicKey(pubPEM)
	require.Error(t, err)
	require.Contains(t, err.Error(), "malformed serial number")

	wrapped := wrapAlipayCertError("appCertPublicKey_*.crt", pubPEM, err)
	require.ErrorContains(t, wrapped, "BEGIN CERTIFICATE")
	require.ErrorContains(t, wrapped, "malformed serial number")
}

func TestAlipayLoadAppCertAcceptsLiteralNewlines(t *testing.T) {
	t.Parallel()

	key := mustAlipayTestKey(t)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	client, err := alipay.New("2021000000000000", string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})), false)
	require.NoError(t, err)
	require.NoError(t, client.LoadAppCertPublicKey(normalizeAlipayPEMMaterial(strings.ReplaceAll(string(certPEM), "\n", `\n`))))
}

func TestPrepareAlipayPlatformPublicKeyAcceptsPKCS1AndPKIX(t *testing.T) {
	t.Parallel()

	key := mustAlipayTestKey(t)
	pkixDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	pkcs1DER := x509.MarshalPKCS1PublicKey(&key.PublicKey)

	pkixRaw := base64.StdEncoding.EncodeToString(pkixDER)
	pkcs1Raw := base64.StdEncoding.EncodeToString(pkcs1DER)

	for _, raw := range []string{pkixRaw, pkcs1Raw, "支付宝公钥：" + pkcs1Raw} {
		prepared, err := prepareAlipayPlatformPublicKey(raw)
		require.NoError(t, err, raw[:16])
		client, err := alipay.New("2021000000000000", string(pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		})), false)
		require.NoError(t, err)
		require.NoError(t, client.LoadAliPayPublicKey(prepared))
	}
}

func TestPrepareAlipayPlatformPublicKeyRejectsPrivateKey(t *testing.T) {
	t.Parallel()

	key := mustAlipayTestKey(t)
	raw := base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PrivateKey(key))
	_, err := prepareAlipayPlatformPublicKey(raw)
	require.ErrorContains(t, err, "private key")
}

func TestGetAlipayClientPrefersPublicKeyMode(t *testing.T) {
	origAppID := setting.AlipayAppId
	origPrivateKey := setting.AlipayPrivateKey
	origPublicKey := setting.AlipayPublicKey
	origAppCert := setting.AlipayAppPublicCert
	origRootCert := setting.AlipayRootCert
	origPublicCert := setting.AlipayPublicCert
	origProduction := setting.AlipayIsProduction
	origEncryptKey := setting.AlipayEncryptKey
	t.Cleanup(func() {
		setting.AlipayAppId = origAppID
		setting.AlipayPrivateKey = origPrivateKey
		setting.AlipayPublicKey = origPublicKey
		setting.AlipayAppPublicCert = origAppCert
		setting.AlipayRootCert = origRootCert
		setting.AlipayPublicCert = origPublicCert
		setting.AlipayIsProduction = origProduction
		setting.AlipayEncryptKey = origEncryptKey
		alipayClientMu.Lock()
		alipayClient = nil
		alipayClientFP = ""
		alipayClientMu.Unlock()
	})

	appKey := mustAlipayTestKey(t)
	alipayKey := mustAlipayTestKey(t)
	pubDER, err := x509.MarshalPKIXPublicKey(&alipayKey.PublicKey)
	require.NoError(t, err)
	alipayPubPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))

	setting.AlipayAppId = "2021000000000000"
	setting.AlipayPrivateKey = string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(appKey),
	}))
	setting.AlipayPublicKey = alipayPubPEM
	// Mis-pasted application public key in cert fields must not block public-key mode.
	setting.AlipayAppPublicCert = mustAlipayTestPublicKeyPEM(t)
	setting.AlipayRootCert = "not-a-cert"
	setting.AlipayPublicCert = "not-a-cert"
	setting.AlipayIsProduction = false
	setting.AlipayEncryptKey = ""

	require.True(t, IsAlipayConfigured())
	client, err := GetAlipayClient()
	require.NoError(t, err)
	require.NotNil(t, client)
}

func mustAlipayTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	require.NoError(t, err)
	return key
}

func mustAlipayTestPublicKeyPEM(t *testing.T) string {
	t.Helper()
	key := mustAlipayTestKey(t)
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))
}
