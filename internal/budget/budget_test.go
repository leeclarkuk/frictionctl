package budget

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/leeclarkuk/frictionctl/internal/result"
)

func TestEvaluatePassAndFail(t *testing.T) {
	one := 1
	zero := 0
	limits := Limits{
		TotalTime:    Duration{2 * time.Minute},
		HumanActions: &one,
		Retries:      &zero,
	}
	pass := Evaluate(limits, result.Signals{HumanActions: 1, TotalTimeMS: 1000}, 4)
	if !pass.Pass {
		t.Fatalf("expected pass, got %#v", pass)
	}
	fail := Evaluate(limits, result.Signals{HumanActions: 4, Retries: 1, TotalTimeMS: 1000}, 10)
	if fail.Pass {
		t.Fatal("expected fail")
	}
	if len(fail.Breaches) != 2 {
		t.Fatalf("breaches = %d, want 2: %#v", len(fail.Breaches), fail.Breaches)
	}
}

func TestLoadExampleBudget(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	dir := filepath.Join(filepath.Dir(file), "..", "..", "examples", "create-service")
	f, err := Find(dir, "create-service")
	if err != nil {
		t.Fatal(err)
	}
	if f.Journey != "create-service" {
		t.Fatalf("journey = %s", f.Journey)
	}
	if f.Budgets.HumanActions == nil || *f.Budgets.HumanActions != 3 {
		t.Fatalf("human_actions budget = %v", f.Budgets.HumanActions)
	}
}
