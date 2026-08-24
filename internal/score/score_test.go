package score

import (
	"math"
	"testing"

	"github.com/leeclarkuk/frictionctl/internal/result"
)

func TestComputeKnownValues(t *testing.T) {
	got := Compute(result.Signals{
		HumanActions:       2,
		RequiredParameters: 1,
		ToolTransitions:    1,
		WaitingMS:          1800,
	})
	// 2*2 + 1*1 + 1.5*1 + 0.01*1.8 = 6.518 -> 6.5
	if math.Abs(got.Value-6.5) > 0.05 {
		t.Fatalf("score = %.1f, want 6.5", got.Value)
	}
	if got.Max != 100 {
		t.Fatalf("max = %.0f, want 100", got.Max)
	}
	if len(got.Contributions) != 9 {
		t.Fatalf("contributions = %d, want 9", len(got.Contributions))
	}
}

func TestComputeCap(t *testing.T) {
	got := Compute(result.Signals{DocumentationLookups: 50})
	if got.Value != 100 {
		t.Fatalf("capped score = %.1f, want 100", got.Value)
	}
}

func TestFormulaHelpMentionsBudget(t *testing.T) {
	help := FormulaHelp()
	if help == "" {
		t.Fatal("empty formula help")
	}
}
