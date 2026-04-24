package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/sadewadee/hera/internal/tools"
)

// MathTool provides mathematical computation capabilities.
type MathTool struct{}

type mathToolArgs struct {
	Operation string    `json:"operation"`
	Values    []float64 `json:"values,omitempty"`
	A         float64   `json:"a,omitempty"`
	B         float64   `json:"b,omitempty"`
	Base      int       `json:"base,omitempty"`
	Number    string    `json:"number,omitempty"`
}

func (t *MathTool) Name() string { return "math" }

func (t *MathTool) Description() string {
	return "Mathematical operations: arithmetic, statistics, trigonometry, base conversion."
}

func (t *MathTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"operation": {
				"type": "string",
				"enum": ["add", "subtract", "multiply", "divide", "power", "sqrt", "abs", "mod", "sum", "mean", "median", "min", "max", "sin", "cos", "tan", "log", "log2", "log10", "ceil", "floor", "round", "base_convert", "factorial"],
				"description": "Math operation."
			},
			"values": {
				"type": "array",
				"items": {"type": "number"},
				"description": "Array of numbers for aggregate operations (sum, mean, etc.)."
			},
			"a": {"type": "number", "description": "First operand."},
			"b": {"type": "number", "description": "Second operand."},
			"base": {"type": "integer", "description": "Target base for base_convert (2-36)."},
			"number": {"type": "string", "description": "Number string for base_convert."}
		},
		"required": ["operation"]
	}`)
}

func (t *MathTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var a mathToolArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	fmtNum := func(v float64) string {
		if v == float64(int64(v)) {
			return fmt.Sprintf("%.0f", v)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	}

	switch a.Operation {
	case "add":
		return &tools.Result{Content: fmtNum(a.A + a.B)}, nil
	case "subtract":
		return &tools.Result{Content: fmtNum(a.A - a.B)}, nil
	case "multiply":
		return &tools.Result{Content: fmtNum(a.A * a.B)}, nil
	case "divide":
		if a.B == 0 {
			return &tools.Result{Content: "division by zero", IsError: true}, nil
		}
		return &tools.Result{Content: fmtNum(a.A / a.B)}, nil
	case "power":
		return &tools.Result{Content: fmtNum(math.Pow(a.A, a.B))}, nil
	case "sqrt":
		if a.A < 0 {
			return &tools.Result{Content: "cannot take square root of negative number", IsError: true}, nil
		}
		return &tools.Result{Content: fmtNum(math.Sqrt(a.A))}, nil
	case "abs":
		return &tools.Result{Content: fmtNum(math.Abs(a.A))}, nil
	case "mod":
		return &tools.Result{Content: fmtNum(math.Mod(a.A, a.B))}, nil
	case "sin":
		return &tools.Result{Content: fmtNum(math.Sin(a.A))}, nil
	case "cos":
		return &tools.Result{Content: fmtNum(math.Cos(a.A))}, nil
	case "tan":
		return &tools.Result{Content: fmtNum(math.Tan(a.A))}, nil
	case "log":
		return &tools.Result{Content: fmtNum(math.Log(a.A))}, nil
	case "log2":
		return &tools.Result{Content: fmtNum(math.Log2(a.A))}, nil
	case "log10":
		return &tools.Result{Content: fmtNum(math.Log10(a.A))}, nil
	case "ceil":
		return &tools.Result{Content: fmtNum(math.Ceil(a.A))}, nil
	case "floor":
		return &tools.Result{Content: fmtNum(math.Floor(a.A))}, nil
	case "round":
		return &tools.Result{Content: fmtNum(math.Round(a.A))}, nil

	case "factorial":
		n := int(a.A)
		if n < 0 {
			return &tools.Result{Content: "factorial of negative number", IsError: true}, nil
		}
		if n > 170 {
			return &tools.Result{Content: "factorial too large (max 170)", IsError: true}, nil
		}
		result := 1.0
		for i := 2; i <= n; i++ {
			result *= float64(i)
		}
		return &tools.Result{Content: fmtNum(result)}, nil

	case "sum":
		s := 0.0
		for _, v := range a.Values {
			s += v
		}
		return &tools.Result{Content: fmtNum(s)}, nil

	case "mean":
		if len(a.Values) == 0 {
			return &tools.Result{Content: "values array is empty", IsError: true}, nil
		}
		s := 0.0
		for _, v := range a.Values {
			s += v
		}
		return &tools.Result{Content: fmtNum(s / float64(len(a.Values)))}, nil

	case "median":
		if len(a.Values) == 0 {
			return &tools.Result{Content: "values array is empty", IsError: true}, nil
		}
		sorted := make([]float64, len(a.Values))
		copy(sorted, a.Values)
		// Simple insertion sort for median
		for i := 1; i < len(sorted); i++ {
			key := sorted[i]
			j := i - 1
			for j >= 0 && sorted[j] > key {
				sorted[j+1] = sorted[j]
				j--
			}
			sorted[j+1] = key
		}
		n := len(sorted)
		if n%2 == 0 {
			return &tools.Result{Content: fmtNum((sorted[n/2-1] + sorted[n/2]) / 2)}, nil
		}
		return &tools.Result{Content: fmtNum(sorted[n/2])}, nil

	case "min":
		if len(a.Values) == 0 {
			return &tools.Result{Content: "values array is empty", IsError: true}, nil
		}
		m := a.Values[0]
		for _, v := range a.Values[1:] {
			if v < m {
				m = v
			}
		}
		return &tools.Result{Content: fmtNum(m)}, nil

	case "max":
		if len(a.Values) == 0 {
			return &tools.Result{Content: "values array is empty", IsError: true}, nil
		}
		m := a.Values[0]
		for _, v := range a.Values[1:] {
			if v > m {
				m = v
			}
		}
		return &tools.Result{Content: fmtNum(m)}, nil

	case "base_convert":
		if a.Number == "" {
			return &tools.Result{Content: "number string is required", IsError: true}, nil
		}
		base := a.Base
		if base < 2 || base > 36 {
			return &tools.Result{Content: "base must be 2-36", IsError: true}, nil
		}
		n, err := strconv.ParseInt(strings.TrimPrefix(a.Number, "0x"), 0, 64)
		if err != nil {
			return &tools.Result{Content: fmt.Sprintf("parse number: %v", err), IsError: true}, nil
		}
		return &tools.Result{Content: strconv.FormatInt(n, base)}, nil

	default:
		return &tools.Result{Content: "unknown operation: " + a.Operation, IsError: true}, nil
	}
}

// RegisterMath registers the math tool with the given registry.
func RegisterMath(registry *tools.Registry) {
	registry.Register(&MathTool{})
}
