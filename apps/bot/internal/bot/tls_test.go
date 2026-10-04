package bot_test

import (
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/marcos-vinicius14/meu-dinheiro/apps/bot/internal/bot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSelfSignedCertWithIP(t *testing.T) {
	tlsCert, certPEM, err := bot.GenerateSelfSignedCert("192.168.1.100")
	require.NoError(t, err)
	assert.NotEmpty(t, certPEM)
	assert.NotEmpty(t, tlsCert.Certificate)

	block, _ := pem.Decode(certPEM)
	require.NotNil(t, block)
	assert.Equal(t, "CERTIFICATE", block.Type)

	parsed, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	assert.Equal(t, "192.168.1.100", parsed.Subject.CommonName)
	require.Len(t, parsed.IPAddresses, 1)
	assert.Equal(t, "192.168.1.100", parsed.IPAddresses[0].String())
}

func TestGenerateSelfSignedCertWithHostname(t *testing.T) {
	tlsCert, certPEM, err := bot.GenerateSelfSignedCert("meudinheiro.local")
	require.NoError(t, err)
	assert.NotEmpty(t, certPEM)
	assert.NotEmpty(t, tlsCert.Certificate)

	block, _ := pem.Decode(certPEM)
	require.NotNil(t, block)
	parsed, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	assert.Equal(t, "meudinheiro.local", parsed.Subject.CommonName)
	require.Contains(t, parsed.DNSNames, "meudinheiro.local")
}
