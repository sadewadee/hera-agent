// Package gateway provides the multi-platform messaging gateway.
//
// restart.go defines shared gateway restart constants and parsing helpers.
package gateway

import (
	"strconv"
	"strings"
)

// GatewayServiceRestartExitCode is the EX_TEMPFAIL from sysexits.h,
// used to ask the service manager to restart the gateway after a
// graceful drain/reload path completes.
const GatewayServiceRestartExitCode = 75

// DefaultGatewayRestartDrainTimeout is the default drain timeout in seconds.
const DefaultGatewayRestartDrainTimeout = 30.0

// ParseRestartDrainTimeout parses a configured drain timeout,
// falling back to the shared default.
func ParseRestartDrainTimeout(raw string) float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return DefaultGatewayRestartDrainTimeout
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return DefaultGatewayRestartDrainTimeout
	}
	if value < 0 {
		return 0
	}
	return value
}
