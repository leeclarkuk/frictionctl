package budget

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/leeclarkuk/frictionctl/internal/result"
	"gopkg.in/yaml.v3"
)

func TestEvaluatePassAndFail(t *testing.T) {
	one := 1
	zero := 0
	limits := Limits{
		TotalTime:    &Duration{2 * time.Minute},
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

func TestZeroTotalTimeIsALimit(t *testing.T) {
	var f File
	raw := []byte("journey: x\nobjective: y\nbudgets:\n  total_time: 0s\n")
	if err := yaml.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	if f.Budgets.TotalTime == nil {
		t.Fatal("total_time: 0s must not be treated as omitted")
	}
	if f.Budgets.TotalTime.Duration != 0 {
		t.Fatalf("duration = %s", f.Budgets.TotalTime.Duration)
	}
	if Evaluate(f.Budgets, result.Signals{TotalTimeMS: 1}, 0).Pass {
		t.Fatal("1ms must exceed a 0s budget")
	}
	if !Evaluate(f.Budgets, result.Signals{TotalTimeMS: 0}, 0).Pass {
		t.Fatal("0ms must pass a 0s budget")
	}
}

func TestOmittedTotalTimeIsUnconstrained(t *testing.T) {
	var f File
	raw := []byte("journey: x\nobjective: y\nbudgets:\n  human_actions: 1\n")
	if err := yaml.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	if f.Budgets.TotalTime != nil {
		t.Fatal("omitted total_time must be nil")
	}
	slo := Evaluate(f.Budgets, result.Signals{HumanActions: 1, TotalTimeMS: 999999}, 0)
	if !slo.Pass {
		t.Fatalf("omitted total_time must not constrain: %#v", slo)
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
	if f.Budgets.ToolTransitions == nil || *f.Budgets.ToolTransitions != 1 {
		t.Fatalf("tool_transitions budget = %v", f.Budgets.ToolTransitions)
	}
}
