package gateway

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGatewayServiceRestartExitCode(t *testing.T) {
	assert.Equal(t, 75, GatewayServiceRestartExitCode)
}

func TestDefaultGatewayRestartDrainTimeout(t *testing.T) {
	assert.Equal(t, 30.0, DefaultGatewayRestartDrainTimeout)
}

func TestParseRestartDrainTimeout_Empty(t *testing.T) {
	assert.Equal(t, DefaultGatewayRestartDrainTimeout, ParseRestartDrainTimeout(""))
}

func TestParseRestartDrainTimeout_ValidNumber(t *testing.T) {
	assert.Equal(t, 60.0, ParseRestartDrainTimeout("60"))
	assert.Equal(t, 15.5, ParseRestartDrainTimeout("15.5"))
}

func TestParseRestartDrainTimeout_InvalidString(t *testing.T) {
	assert.Equal(t, DefaultGatewayRestartDrainTimeout, ParseRestartDrainTimeout("not-a-number"))
}

func TestParseRestartDrainTimeout_Negative(t *testing.T) {
	assert.Equal(t, 0.0, ParseRestartDrainTimeout("-5"))
}

func TestParseRestartDrainTimeout_Zero(t *testing.T) {
	assert.Equal(t, 0.0, ParseRestartDrainTimeout("0"))
}

func TestParseRestartDrainTimeout_Whitespace(t *testing.T) {
	assert.Equal(t, 45.0, ParseRestartDrainTimeout("  45  "))
}
