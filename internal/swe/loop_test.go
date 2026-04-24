package swe

import "testing"

func TestIterationController_Next(t *testing.T) {
	c := NewIterationController(3)
	if c.Current() != 0 {
		t.Errorf("initial current = %d, want 0", c.Current())
	}

	for i := 1; i <= 3; i++ {
		if !c.Next() {
			t.Errorf("Next() = false at iteration %d, want true", i)
		}
		if c.Current() != i {
			t.Errorf("Current() = %d, want %d", c.Current(), i)
		}
	}

	// 4th call: should be false
	if c.Next() {
		t.Errorf("Next() = true after max, want false")
	}
}

func TestIterationController_Exhausted(t *testing.T) {
	c := NewIterationController(1)
	if c.Exhausted() {
		t.Error("Exhausted() = true before any Next(), want false")
	}
	c.Next()
	if !c.Exhausted() {
		t.Error("Exhausted() = false after max iterations, want true")
	}
}

func TestIterationController_DefaultMax(t *testing.T) {
	c := NewIterationController(0)
	count := 0
	for c.Next() {
		count++
	}
	if count != 10 {
		t.Errorf("default max iterations = %d, want 10", count)
	}
}
