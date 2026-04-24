package gateway

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatusReport_String(t *testing.T) {
	report := StatusReport{
		ActiveAdapters: 1,
		TotalAdapters:  2,
		AdapterStatuses: []AdapterStatus{
			{Name: "telegram", Connected: true},
			{Name: "discord", Connected: false},
		},
	}

	str := report.String()
	assert.Contains(t, str, "Gateway Status")
	assert.Contains(t, str, "1/2 connected")
	assert.Contains(t, str, "telegram")
	assert.Contains(t, str, "discord")
}

func TestStatusReport_String_Empty(t *testing.T) {
	report := StatusReport{}
	str := report.String()
	assert.Contains(t, str, "Gateway Status")
	assert.Contains(t, str, "0/0 connected")
}

func TestAdapterStatus_Fields(t *testing.T) {
	status := AdapterStatus{Name: "test", Connected: true}
	assert.Equal(t, "test", status.Name)
	assert.True(t, status.Connected)
}
