// Package swe implements the software-engineering agent engine for hera-swe.
// It orchestrates multi-step autonomous code changes via a bounded iteration
// loop, using a curated tool subset (file_read, file_write, run_command,
// patch, code_exec, git) and a TDD feedback cycle.
package swe

// IterationController tracks iterations and decides when to stop.
type IterationController struct {
	max     int
	current int
}

// NewIterationController creates a controller with a maximum iteration count.
// If max is less than 1 it defaults to 10.
func NewIterationController(max int) *IterationController {
	if max < 1 {
		max = 10
	}
	return &IterationController{max: max}
}

// Next increments the counter and returns true if another iteration should run.
func (c *IterationController) Next() bool {
	if c.current >= c.max {
		return false
	}
	c.current++
	return true
}

// Current returns the current iteration number (1-indexed after first Next()).
func (c *IterationController) Current() int {
	return c.current
}

// Exhausted returns true if the max has been reached.
func (c *IterationController) Exhausted() bool {
	return c.current >= c.max
}
