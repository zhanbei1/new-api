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
