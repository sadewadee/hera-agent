package builtin

import (
	"context"
	"encoding/json"
	"testing"
)

func TestMathTool_Arithmetic(t *testing.T) {
	tool := &MathTool{}
	ctx := context.Background()

	tests := []struct {
		name string
		args mathToolArgs
		want string
	}{
		{"add", mathToolArgs{Operation: "add", A: 3, B: 4}, "7"},
		{"subtract", mathToolArgs{Operation: "subtract", A: 10, B: 3}, "7"},
		{"multiply", mathToolArgs{Operation: "multiply", A: 6, B: 7}, "42"},
		{"divide", mathToolArgs{Operation: "divide", A: 15, B: 3}, "5"},
		{"power", mathToolArgs{Operation: "power", A: 2, B: 10}, "1024"},
		{"sqrt", mathToolArgs{Operation: "sqrt", A: 144}, "12"},
		{"abs", mathToolArgs{Operation: "abs", A: -42}, "42"},
		{"mod", mathToolArgs{Operation: "mod", A: 17, B: 5}, "2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, _ := json.Marshal(tt.args)
			result, err := tool.Execute(ctx, args)
			if err != nil {
				t.Fatalf("Execute error: %v", err)
			}
			if result.IsError {
				t.Fatalf("unexpected error: %s", result.Content)
			}
			if result.Content != tt.want {
				t.Errorf("got %q, want %q", result.Content, tt.want)
			}
		})
	}
}

func TestMathTool_DivisionByZero(t *testing.T) {
	tool := &MathTool{}
	ctx := context.Background()

	args, _ := json.Marshal(mathToolArgs{Operation: "divide", A: 10, B: 0})
	result, _ := tool.Execute(ctx, args)
	if !result.IsError {
		t.Error("division by zero should return error")
	}
}

func TestMathTool_Aggregates(t *testing.T) {
	tool := &MathTool{}
	ctx := context.Background()

	values := []float64{1, 2, 3, 4, 5}

	tests := []struct {
		name string
		op   string
		want string
	}{
		{"sum", "sum", "15"},
		{"mean", "mean", "3"},
		{"median", "median", "3"},
		{"min", "min", "1"},
		{"max", "max", "5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, _ := json.Marshal(mathToolArgs{Operation: tt.op, Values: values})
			result, _ := tool.Execute(ctx, args)
			if result.IsError {
				t.Fatalf("unexpected error: %s", result.Content)
			}
			if result.Content != tt.want {
				t.Errorf("got %q, want %q", result.Content, tt.want)
			}
		})
	}
}

func TestMathTool_Factorial(t *testing.T) {
	tool := &MathTool{}
	ctx := context.Background()

	args, _ := json.Marshal(mathToolArgs{Operation: "factorial", A: 5})
	result, _ := tool.Execute(ctx, args)
	if result.Content != "120" {
		t.Errorf("5! = %s, want 120", result.Content)
	}
}
