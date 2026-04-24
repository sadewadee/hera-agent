// Package environments provides execution environment implementations.
//
// modal_utils.go contains shared execution flow utilities for Modal
// transports. This module handles command preparation, cwd/timeout
// normalisation, stdin/sudo shell wrapping, and the common
// start-poll-cancel execution loop used by both direct and managed
// Modal backends.
package environments

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// PreparedModalExec holds normalised command data passed to a
// transport-specific exec runner.
type PreparedModalExec struct {
	Command   string
	CWD       string
	Timeout   int
	StdinData string
}

// ModalExecStart is the response from a transport after starting an exec.
type ModalExecStart struct {
	Handle          any
	ImmediateResult *ExecResult
}

// WrapModalStdinHeredoc appends stdin as a shell heredoc for transports
// without stdin piping.
func WrapModalStdinHeredoc(command, stdinData string) string {
	marker := fmt.Sprintf("HERA_EOF_%s", uuid.New().String()[:8])
	for strings.Contains(stdinData, marker) {
		marker = fmt.Sprintf("HERA_EOF_%s", uuid.New().String()[:8])
	}
	return fmt.Sprintf("%s << '%s'\n%s\n%s", command, marker, stdinData, marker)
}

// WrapModalSudoPipe feeds sudo via a shell pipe for transports without
// direct stdin piping.
func WrapModalSudoPipe(command, sudoStdin string) string {
	sudoStdin = strings.TrimRight(sudoStdin, "\n\r")
	return fmt.Sprintf("printf '%%s\\n' %s | %s", shellQuoteSimple(sudoStdin), command)
}

// PrepareModalExec normalises command, cwd, timeout, and stdin for a
// Modal execution.
func PrepareModalExec(command, cwd string, timeout int, stdinData string, defaultCWD string, defaultTimeout int, stdinMode string) PreparedModalExec {
	effectiveCWD := cwd
	if effectiveCWD == "" {
		effectiveCWD = defaultCWD
	}
	effectiveTimeout := timeout
	if effectiveTimeout <= 0 {
		effectiveTimeout = defaultTimeout
	}

	execCommand := command
	execStdin := ""
	if stdinMode == "payload" && stdinData != "" {
		execStdin = stdinData
	} else if stdinMode == "heredoc" && stdinData != "" {
		execCommand = WrapModalStdinHeredoc(execCommand, stdinData)
	}

	return PreparedModalExec{
		Command:   execCommand,
		CWD:       effectiveCWD,
		Timeout:   effectiveTimeout,
		StdinData: execStdin,
	}
}

// ModalPollConfig holds polling configuration.
type ModalPollConfig struct {
	PollInterval    time.Duration
	GraceSeconds    float64
	InterruptOutput string
	ErrorPrefix     string
}

// DefaultModalPollConfig returns sensible defaults for Modal polling.
func DefaultModalPollConfig() ModalPollConfig {
	return ModalPollConfig{
		PollInterval:    250 * time.Millisecond,
		GraceSeconds:    10.0,
		InterruptOutput: "[Command interrupted]",
		ErrorPrefix:     "Modal execution error",
	}
}

// shellQuoteSimple wraps a string in single quotes for shell safety.
func shellQuoteSimple(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
