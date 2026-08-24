package compare

import (
	"fmt"
	"strings"

	"github.com/leeclarkuk/frictionctl/internal/result"
)

// Report is a baseline-versus-current comparison.
type Report struct {
	Journey string `json:"journey"`
	Pass    bool   `json:"pass"`
	Reason  string `json:"reason,omitempty"`
	Rows    []Row  `json:"rows"`
}

// Row is one signal compared across two runs.
type Row struct {
	Signal   string `json:"signal"`
	Baseline string `json:"baseline"`
	Current  string `json:"current"`
	Delta    string `json:"delta"`
	Worse    bool   `json:"worse"`
}

type metric struct {
	name    string
	base    float64
	curr    float64
	format  func(float64) string
	integer bool
	gated   bool
}

// Diff fails when gated friction signals or the score get worse, or when the
// current run misses its SLO. Wall time is reported but does not fail the
// comparison on its own; it is too noisy for CI.
func Diff(baseline, current result.Result) Report {
	rep := Report{
		Journey: current.Journey,
		Pass:    true,
	}
	if baseline.Journey != "" && current.Journey != "" && baseline.Journey != current.Journey {
		rep.Journey = current.Journey
	}

	metrics := []metric{
		{"time_to_service", float64(baseline.Signals.TotalTimeMS), float64(current.Signals.TotalTimeMS), formatMS, false, false},
		{"human_actions", float64(baseline.Signals.HumanActions), float64(current.Signals.HumanActions), formatInt, true, true},
		{"required_parameters", float64(baseline.Signals.RequiredParameters), float64(current.Signals.RequiredParameters), formatInt, true, true},
		{"tool_transitions", float64(baseline.Signals.ToolTransitions), float64(current.Signals.ToolTransitions), formatInt, true, true},
		{"retries", float64(baseline.Signals.Retries), float64(current.Signals.Retries), formatInt, true, true},
		{"approvals", float64(baseline.Signals.Approvals), float64(current.Signals.Approvals), formatInt, true, true},
		{"privilege_escalations", float64(baseline.Signals.PrivilegeEscalations), float64(current.Signals.PrivilegeEscalations), formatInt, true, true},
		{"documentation_lookups", float64(baseline.Signals.DocumentationLookups), float64(current.Signals.DocumentationLookups), formatInt, true, true},
		{"escape_hatch_usage", float64(baseline.Signals.EscapeHatchUsage), float64(current.Signals.EscapeHatchUsage), formatInt, true, true},
		{"friction_score", baseline.Score.Value, current.Score.Value, formatScore, false, true},
	}

	var worse []string
	for _, m := range metrics {
		row := Row{
			Signal:   m.name,
			Baseline: m.format(m.base),
			Current:  m.format(m.curr),
			Delta:    delta(m.base, m.curr, m.integer),
			Worse:    m.gated && m.curr > m.base,
		}
		rep.Rows = append(rep.Rows, row)
		if row.Worse {
			worse = append(worse, fmt.Sprintf("%s: %s → %s", m.name, row.Baseline, row.Current))
		}
	}

	if !current.SLO.Pass {
		rep.Pass = false
		if current.SLO.Pass != baseline.SLO.Pass {
			worse = append(worse, "developer experience SLO failed")
		} else {
			worse = append(worse, "current run is still outside its friction budget")
		}
	}
	if len(worse) > 0 {
		rep.Pass = false
		rep.Reason = reason(current, worse)
	}
	return rep
}

func reason(current result.Result, worse []string) string {
	var b strings.Builder
	b.WriteString("Developer journey regression.\n\n")
	if len(current.SLO.Breaches) > 0 {
		b.WriteString("The current run exceeds its friction budget:\n")
		for _, br := range current.SLO.Breaches {
			fmt.Fprintf(&b, "  %s: %s (budget %s)\n", br.Signal, br.Actual, br.Budget)
		}
		b.WriteString("\n")
	}
	b.WriteString("Worse than baseline:\n")
	for _, w := range worse {
		fmt.Fprintf(&b, "  %s\n", w)
	}
	return strings.TrimSpace(b.String())
}

func delta(base, curr float64, integer bool) string {
	if base == curr {
		return "0"
	}
	sign := "+"
	diff := curr - base
	if diff < 0 {
		sign = ""
	}
	if integer {
		if base == 0 {
			return fmt.Sprintf("%s%.0f", sign, diff)
		}
		pct := (diff / base) * 100
		return fmt.Sprintf("%s%.0f (%+.0f%%)", sign, diff, pct)
	}
	if base == 0 {
		return fmt.Sprintf("%s%.1f", sign, diff)
	}
	pct := (diff / base) * 100
	return fmt.Sprintf("%+.1f (%+.0f%%)", diff, pct)
}

func formatInt(v float64) string { return fmt.Sprintf("%.0f", v) }

func formatScore(v float64) string { return fmt.Sprintf("%.1f", v) }

func formatMS(v float64) string {
	ms := int64(v)
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}
