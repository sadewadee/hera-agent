package gateway

import (
	"fmt"
	"strings"
	"time"
)

// StatusReport describes the current state of the gateway.
type StatusReport struct {
	Uptime          time.Duration
	ActiveAdapters  int
	TotalAdapters   int
	ConnectedNames  []string
	TotalMessages   int64
	AdapterStatuses []AdapterStatus
}

// AdapterStatus describes a single adapter's state.
type AdapterStatus struct {
	Name      string
	Connected bool
}

// StatusReporter generates status reports for the gateway.
type StatusReporter struct {
	gateway   *Gateway
	startTime time.Time
}

// NewStatusReporter creates a status reporter for the given gateway.
func NewStatusReporter(gw *Gateway) *StatusReporter {
	return &StatusReporter{gateway: gw, startTime: time.Now()}
}

// Report generates a current status report.
func (sr *StatusReporter) Report() StatusReport {
	adapters := sr.gateway.Adapters()
	report := StatusReport{
		Uptime:        time.Since(sr.startTime),
		TotalAdapters: len(adapters),
	}
	for _, a := range adapters {
		status := AdapterStatus{Name: a.Name(), Connected: a.IsConnected()}
		report.AdapterStatuses = append(report.AdapterStatuses, status)
		if a.IsConnected() {
			report.ActiveAdapters++
			report.ConnectedNames = append(report.ConnectedNames, a.Name())
		}
	}
	return report
}

// String returns a human-readable status string.
func (r StatusReport) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Gateway Status\n")
	fmt.Fprintf(&b, "  Uptime: %s\n", r.Uptime.Truncate(time.Second))
	fmt.Fprintf(&b, "  Adapters: %d/%d connected\n", r.ActiveAdapters, r.TotalAdapters)
	for _, a := range r.AdapterStatuses {
		mark := "x"; if a.Connected { mark = "v" }
		fmt.Fprintf(&b, "    [%s] %s\n", mark, a.Name)
	}
	return b.String()
}
