package score

import (
	"fmt"
	"strings"

	"github.com/leeclarkuk/frictionctl/internal/result"
)

const Max = 100.0

// Weights are the published formula. Do not tune them to flatter a demo.
const (
	WeightHumanActions         = 2.0
	WeightRequiredParameters   = 1.0
	WeightToolTransitions      = 1.5
	WeightRetries              = 2.0
	WeightApprovals            = 3.0
	WeightPrivilegeEscalations = 4.0
	WeightDocumentationLookups = 5.0
	WeightEscapeHatchUsage     = 3.0
	WeightWaitingPerSecond     = 0.01
)

// Compute returns a transparent friction score. Lower is better.
func Compute(signals result.Signals) result.Score {
	waitingSeconds := float64(signals.WaitingMS) / 1000.0
	contribs := []result.Contribution{
		term("human_actions", float64(signals.HumanActions), WeightHumanActions),
		term("required_parameters", float64(signals.RequiredParameters), WeightRequiredParameters),
		term("tool_transitions", float64(signals.ToolTransitions), WeightToolTransitions),
		term("retries", float64(signals.Retries), WeightRetries),
		term("approvals", float64(signals.Approvals), WeightApprovals),
		term("privilege_escalations", float64(signals.PrivilegeEscalations), WeightPrivilegeEscalations),
		term("documentation_lookups", float64(signals.DocumentationLookups), WeightDocumentationLookups),
		term("escape_hatch_usage", float64(signals.EscapeHatchUsage), WeightEscapeHatchUsage),
		term("waiting_seconds", waitingSeconds, WeightWaitingPerSecond),
	}
	var raw float64
	for _, c := range contribs {
		raw += c.Points
	}
	value := raw
	if value > Max {
		value = Max
	}
	if value < 0 {
		value = 0
	}
	return result.Score{Value: round1(value), Max: Max, Contributions: contribs}
}

func term(signal string, value, weight float64) result.Contribution {
	return result.Contribution{
		Signal: signal,
		Value:  round1(value),
		Weight: weight,
		Points: round1(value * weight),
	}
}

func round1(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}

// FormulaHelp is the human explanation printed by `frictionctl explain`.
func FormulaHelp() string {
	var b strings.Builder
	b.WriteString("Friction score\n\n")
	b.WriteString("Lower is better. The score is a weighted sum of observable proxies, capped at 100.\n")
	b.WriteString("It is not a measure of engineer skill, utilisation or cognitive load.\n")
	b.WriteString("Budgets are policy. This number is explanation.\n")
	b.WriteString("Pass or fail is decided by the budget, not by this number.\n\n")
	b.WriteString("score = min(100,\n")
	fmt.Fprintf(&b, "    %.1f * human_actions\n", WeightHumanActions)
	fmt.Fprintf(&b, "  + %.1f * required_parameters\n", WeightRequiredParameters)
	fmt.Fprintf(&b, "  + %.1f * tool_transitions\n", WeightToolTransitions)
	fmt.Fprintf(&b, "  + %.1f * retries\n", WeightRetries)
	fmt.Fprintf(&b, "  + %.1f * approvals\n", WeightApprovals)
	fmt.Fprintf(&b, "  + %.1f * privilege_escalations\n", WeightPrivilegeEscalations)
	fmt.Fprintf(&b, "  + %.1f * documentation_lookups\n", WeightDocumentationLookups)
	fmt.Fprintf(&b, "  + %.1f * escape_hatch_usage\n", WeightEscapeHatchUsage)
	fmt.Fprintf(&b, "  + %.2f * waiting_seconds\n", WeightWaitingPerSecond)
	b.WriteString(")\n")
	return b.String()
}
