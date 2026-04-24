package batch

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestTTYProgress_Render(t *testing.T) {
	var buf bytes.Buffer
	p := NewTTYProgress(&buf, 20)

	p.Start(10, 0)
	for i := 0; i < 5; i++ {
		p.Increment()
	}
	p.Finish()

	out := buf.String()
	// Should contain bar characters.
	if !strings.Contains(out, "[") || !strings.Contains(out, "]") {
		t.Errorf("expected bar brackets in output: %q", out)
	}
	// Finish should include "Done:".
	if !strings.Contains(out, "Done:") {
		t.Errorf("expected 'Done:' in finish output: %q", out)
	}
}

func TestTTYProgress_Resume(t *testing.T) {
	var buf bytes.Buffer
	p := NewTTYProgress(&buf, 10)
	// Start with 3 already done out of 10.
	p.Start(10, 3)
	p.Increment()
	p.Finish()

	// Should have rendered at least once.
	if buf.Len() == 0 {
		t.Error("expected non-empty output")
	}
}

func TestTTYProgress_Empty(t *testing.T) {
	var buf bytes.Buffer
	p := NewTTYProgress(&buf, 10)
	// Zero total should not panic.
	p.Start(0, 0)
	p.Increment()
	p.Finish()
}

func TestNoopProgress(t *testing.T) {
	var p NoopProgress
	p.Start(100, 0)
	p.Increment()
	p.Finish()
	// Just verify no panic.
}

func TestFmtDuration(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{5 * time.Second, "5s"},
		{65 * time.Second, "1m5s"},
		{120 * time.Second, "2m0s"},
	}
	for _, tc := range cases {
		got := fmtDuration(tc.d)
		if got != tc.want {
			t.Errorf("fmtDuration(%v) = %q, want %q", tc.d, got, tc.want)
		}
	}
}
