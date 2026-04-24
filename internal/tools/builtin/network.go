package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/sadewadee/hera/internal/tools"
)

// NetworkTool provides network diagnostic capabilities (ping, DNS lookup, port check).
type NetworkTool struct{}

type networkArgs struct {
	Action  string `json:"action"`
	Host    string `json:"host"`
	Port    int    `json:"port,omitempty"`
	Count   int    `json:"count,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
}

func (t *NetworkTool) Name() string { return "network" }

func (t *NetworkTool) Description() string {
	return "Network diagnostics: ping, DNS lookup, port check, and traceroute."
}

func (t *NetworkTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["ping", "dns", "port_check", "traceroute"],
				"description": "Network diagnostic action."
			},
			"host": {
				"type": "string",
				"description": "Target hostname or IP address."
			},
			"port": {
				"type": "integer",
				"description": "Port number (for port_check action)."
			},
			"count": {
				"type": "integer",
				"description": "Number of pings. Defaults to 4."
			},
			"timeout": {
				"type": "integer",
				"description": "Timeout in seconds. Defaults to 5."
			}
		},
		"required": ["action", "host"]
	}`)
}

func (t *NetworkTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var a networkArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	if a.Timeout <= 0 {
		a.Timeout = 5
	}

	switch a.Action {
	case "ping":
		return networkPing(ctx, a)
	case "dns":
		return networkDNS(ctx, a)
	case "port_check":
		return networkPortCheck(a)
	case "traceroute":
		return networkTraceroute(ctx, a)
	default:
		return &tools.Result{Content: "unknown action: " + a.Action, IsError: true}, nil
	}
}

func networkPing(ctx context.Context, a networkArgs) (*tools.Result, error) {
	count := a.Count
	if count <= 0 {
		count = 4
	}
	cmd := exec.CommandContext(ctx, "ping", "-c", fmt.Sprintf("%d", count), "-W", fmt.Sprintf("%d", a.Timeout), a.Host)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("ping failed: %s\n%s", err, string(out)), IsError: true}, nil
	}
	return &tools.Result{Content: string(out)}, nil
}

func networkDNS(ctx context.Context, a networkArgs) (*tools.Result, error) {
	resolver := net.Resolver{}
	timeout := time.Duration(a.Timeout) * time.Second
	lookupCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var sb strings.Builder
	fmt.Fprintf(&sb, "DNS lookup for %s:\n", a.Host)

	addrs, err := resolver.LookupHost(lookupCtx, a.Host)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("DNS lookup failed: %v", err), IsError: true}, nil
	}
	fmt.Fprintf(&sb, "\nA/AAAA records:\n")
	for _, addr := range addrs {
		fmt.Fprintf(&sb, "  %s\n", addr)
	}

	mxs, err := resolver.LookupMX(lookupCtx, a.Host)
	if err == nil && len(mxs) > 0 {
		fmt.Fprintf(&sb, "\nMX records:\n")
		for _, mx := range mxs {
			fmt.Fprintf(&sb, "  %s (priority %d)\n", mx.Host, mx.Pref)
		}
	}

	txts, err := resolver.LookupTXT(lookupCtx, a.Host)
	if err == nil && len(txts) > 0 {
		fmt.Fprintf(&sb, "\nTXT records:\n")
		for _, txt := range txts {
			fmt.Fprintf(&sb, "  %s\n", txt)
		}
	}

	return &tools.Result{Content: sb.String()}, nil
}

func networkPortCheck(a networkArgs) (*tools.Result, error) {
	if a.Port <= 0 || a.Port > 65535 {
		return &tools.Result{Content: "invalid port number", IsError: true}, nil
	}

	addr := net.JoinHostPort(a.Host, fmt.Sprintf("%d", a.Port))
	timeout := time.Duration(a.Timeout) * time.Second

	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("Port %d on %s: CLOSED (%v)", a.Port, a.Host, err)}, nil
	}
	conn.Close()
	return &tools.Result{Content: fmt.Sprintf("Port %d on %s: OPEN", a.Port, a.Host)}, nil
}

func networkTraceroute(ctx context.Context, a networkArgs) (*tools.Result, error) {
	cmd := exec.CommandContext(ctx, "traceroute", "-m", "15", "-w", fmt.Sprintf("%d", a.Timeout), a.Host)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("traceroute: %s\n%s", err, string(out)), IsError: true}, nil
	}
	return &tools.Result{Content: string(out)}, nil
}

// RegisterNetwork registers the network tool with the given registry.
func RegisterNetwork(registry *tools.Registry) {
	registry.Register(&NetworkTool{})
}
