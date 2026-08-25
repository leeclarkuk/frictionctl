package runner

import (
	"context"
	"testing"

	"github.com/leeclarkuk/frictionctl/internal/adapter"
	"github.com/leeclarkuk/frictionctl/internal/budget"
	"github.com/leeclarkuk/frictionctl/internal/journey"
)

type fakeAdapter struct {
	failRemaining map[string]int
	calls         []string
}

func (f *fakeAdapter) Run(_ context.Context, spec adapter.Spec) (adapter.Outcome, error) {
	name := spec.Command[0]
	if len(spec.Command) > 1 {
		name = spec.Command[1]
	}
	f.calls = append(f.calls, name)
	if f.failRemaining[name] > 0 {
		f.failRemaining[name]--
		return adapter.Outcome{ExitCode: 1, Stderr: "transient"}, nil
	}
	return adapter.Outcome{}, nil
}

func TestRunAggregatesSignalsAndRetries(t *testing.T) {
	one := 10
	zero := 0
	j := journey.Journey{
		Name:      "create-service",
		Objective: "running-service",
		Steps: []journey.Step{
			{
				ID:      "scaffold",
				Name:    "Scaffold",
				Command: []string{"demo-platform", "scaffold"},
				Signals: journey.Signals{HumanActions: 1, RequiredParameters: 1, Tool: "demo-platform"},
			},
			{
				ID:         "validate",
				Name:       "Validate",
				Command:    []string{"demo-platform", "validate"},
				MaxRetries: 1,
				Signals:    journey.Signals{Tool: "demo-platform"},
			},
			{
				ID:      "kube",
				Name:    "Set namespace",
				Command: []string{"demo-kube", "set-namespace"},
				Signals: journey.Signals{HumanActions: 1, Tool: "demo-kube"},
			},
		},
	}
	b := budget.File{
		Journey:   "create-service",
		Objective: "running-service",
		Budgets:   budget.Limits{HumanActions: &one, Retries: &zero},
	}
	fake := &fakeAdapter{failRemaining: map[string]int{"validate": 1}}
	res, err := Runner{Adapter: fake}.Run(context.Background(), j, b, Options{Workdir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if res.Contract != "create-service" {
		t.Fatalf("contract = %s", res.Contract)
	}
	if res.Signals.HumanActions != 2 {
		t.Fatalf("human_actions = %d", res.Signals.HumanActions)
	}
	if res.Signals.ToolTransitions != 1 {
		t.Fatalf("tool_transitions = %d", res.Signals.ToolTransitions)
	}
	if res.Signals.Retries != 1 {
		t.Fatalf("retries = %d, calls=%v", res.Signals.Retries, fake.calls)
	}
	if res.SLO.Pass {
		t.Fatal("expected SLO fail on retries")
	}
	if res.Trace.Name == "" || len(res.Trace.Spans) != 3 {
		t.Fatalf("trace = %#v", res.Trace)
	}
}
