package platforms

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeFallbackIPs_ValidIPv4(t *testing.T) {
	ips := NormalizeFallbackIPs([]string{"149.154.167.220"})
	require.Len(t, ips, 1)
	assert.Equal(t, "149.154.167.220", ips[0])
}

func TestNormalizeFallbackIPs_Empty(t *testing.T) {
	ips := NormalizeFallbackIPs([]string{})
	assert.Nil(t, ips)
}

func TestNormalizeFallbackIPs_BlankEntries(t *testing.T) {
	ips := NormalizeFallbackIPs([]string{"", "  ", "149.154.167.220"})
	require.Len(t, ips, 1)
	assert.Equal(t, "149.154.167.220", ips[0])
}

func TestNormalizeFallbackIPs_InvalidIP(t *testing.T) {
	ips := NormalizeFallbackIPs([]string{"not-an-ip"})
	assert.Nil(t, ips)
}

func TestNormalizeFallbackIPs_IPv6Rejected(t *testing.T) {
	ips := NormalizeFallbackIPs([]string{"2001:db8::1"})
	assert.Nil(t, ips)
}

func TestNormalizeFallbackIPs_PrivateRejected(t *testing.T) {
	ips := NormalizeFallbackIPs([]string{"192.168.1.1"})
	assert.Nil(t, ips)
}

func TestNormalizeFallbackIPs_LoopbackRejected(t *testing.T) {
	ips := NormalizeFallbackIPs([]string{"127.0.0.1"})
	assert.Nil(t, ips)
}

func TestNormalizeFallbackIPs_Multiple(t *testing.T) {
	ips := NormalizeFallbackIPs([]string{"149.154.167.220", "91.108.4.1", "192.168.1.1"})
	assert.Len(t, ips, 2)
}

func TestParseFallbackIPEnv_Empty(t *testing.T) {
	ips := ParseFallbackIPEnv("")
	assert.Nil(t, ips)
}

func TestParseFallbackIPEnv_SingleIP(t *testing.T) {
	ips := ParseFallbackIPEnv("149.154.167.220")
	require.Len(t, ips, 1)
	assert.Equal(t, "149.154.167.220", ips[0])
}

func TestParseFallbackIPEnv_CommaSeparated(t *testing.T) {
	ips := ParseFallbackIPEnv("149.154.167.220,91.108.4.1")
	assert.Len(t, ips, 2)
}

func TestParseFallbackIPEnv_WithSpaces(t *testing.T) {
	ips := ParseFallbackIPEnv("149.154.167.220 , 91.108.4.1")
	assert.Len(t, ips, 2)
}

func TestNewTelegramFallbackTransport_NilBase(t *testing.T) {
	transport := NewTelegramFallbackTransport([]string{"149.154.167.220"}, nil)
	require.NotNil(t, transport)
	assert.Equal(t, []string{"149.154.167.220"}, transport.fallbackIPs)
}

func TestNewTelegramFallbackTransport_EmptyIPs(t *testing.T) {
	transport := NewTelegramFallbackTransport(nil, nil)
	require.NotNil(t, transport)
	assert.Empty(t, transport.fallbackIPs)
}

func TestIsRetryableConnectError_Nil(t *testing.T) {
	assert.False(t, isRetryableConnectError(nil))
}
