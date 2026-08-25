package compare

import (
	"strings"
	"testing"

	"github.com/leeclarkuk/frictionctl/internal/result"
)

func baseResult() result.Result {
	return result.Result{
		Journey:   "create-service",
		Contract:  "create-service",
		Objective: "running-service",
		Signals:   result.Signals{HumanActions: 2, ToolTransitions: 0, TotalTimeMS: 800, WaitingMS: 800},
		Score:     result.Score{Value: 6.5, Max: 100},
		SLO:       result.SLO{Pass: true},
	}
}

func TestDiffTimeAndScoreAreReportedNotGated(t *testing.T) {
	base := baseResult()
	slower := base
	slower.Signals.TotalTimeMS = 5000
	slower.Signals.WaitingMS = 5000
	slower.Score.Value = 12
	rep := Diff(base, slower, Options{})
	if !rep.Pass {
		t.Fatalf("time and score increases must not fail compare: %+v", rep)
	}
	if rep.Notes == "" {
		t.Fatal("expected notes for ungated increases")
	}
	if !strings.Contains(rep.Notes, "friction_score") {
		t.Fatalf("notes = %s", rep.Notes)
	}
}

func TestDiffDiscreteSignalIncreaseFails(t *testing.T) {
	base := baseResult()
	worse := base
	worse.Signals.HumanActions = 5
	worse.Score.Value = 12
	worse.SLO.Pass = false
	worse.SLO.Breaches = []result.Breach{{Signal: "human_actions", Actual: "5", Budget: "3"}}
	rep := Diff(base, worse, Options{})
	if rep.Pass {
		t.Fatal("expected regression")
	}
	if rep.Reason == "" {
		t.Fatal("expected reason")
	}
}

func TestDiffGateScore(t *testing.T) {
	base := baseResult()
	higher := base
	higher.Score.Value = 12
	rep := Diff(base, higher, Options{})
	if !rep.Pass {
		t.Fatal("score increase must pass without --gate-score")
	}
	rep = Diff(base, higher, Options{GateScore: true})
	if rep.Pass {
		t.Fatal("score increase must fail with GateScore")
	}
}

func TestDiffGateTime(t *testing.T) {
	base := baseResult()
	slower := base
	slower.Signals.TotalTimeMS = 5000
	rep := Diff(base, slower, Options{GateTime: true})
	if rep.Pass {
		t.Fatal("time increase must fail with GateTime")
	}
}

func TestCheckIdentity(t *testing.T) {
	a := baseResult()
	b := baseResult()
	if err := CheckIdentity(a, b); err != nil {
		t.Fatal(err)
	}
	b.Contract = "onboard-engineer"
	if err := CheckIdentity(a, b); err == nil {
		t.Fatal("expected contract mismatch")
	}
	b = baseResult()
	b.Objective = "first-build"
	if err := CheckIdentity(a, b); err == nil {
		t.Fatal("expected objective mismatch")
	}
	b = baseResult()
	b.Contract = ""
	if err := CheckIdentity(a, b); err == nil {
		t.Fatal("expected empty contract error")
	}
}
