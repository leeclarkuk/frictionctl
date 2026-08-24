package compare

import (
	"testing"

	"github.com/leeclarkuk/frictionctl/internal/result"
)

func TestDiffRegressionAndTimeNoise(t *testing.T) {
	base := result.Result{
		Journey: "create-service",
		Signals: result.Signals{HumanActions: 2, ToolTransitions: 1, TotalTimeMS: 800},
		Score:   result.Score{Value: 6.5, Max: 100},
		SLO:     result.SLO{Pass: true},
	}
	same := base
	same.Signals.TotalTimeMS = 1200
	rep := Diff(base, same)
	if !rep.Pass {
		t.Fatalf("time-only change should pass: %+v", rep)
	}

	worse := base
	worse.Signals.HumanActions = 5
	worse.Score.Value = 12
	worse.SLO.Pass = false
	worse.SLO.Breaches = []result.Breach{{Signal: "human_actions", Actual: "5", Budget: "3"}}
	rep = Diff(base, worse)
	if rep.Pass {
		t.Fatal("expected regression")
	}
	if rep.Reason == "" {
		t.Fatal("expected reason")
	}
}
