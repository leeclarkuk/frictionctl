package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/leeclarkuk/frictionctl/internal/compare"
	"github.com/leeclarkuk/frictionctl/internal/journey"
	"github.com/leeclarkuk/frictionctl/internal/result"
)

// Journey writes the human report for a completed run.
func Journey(w io.Writer, r result.Result) error {
	fmt.Fprintf(w, "Developer Journey: %s\n", r.Journey)
	if r.Contract != "" {
		fmt.Fprintf(w, "Contract:          %s\n", r.Contract)
	}
	if r.Objective != "" {
		fmt.Fprintf(w, "Objective:         %s\n", r.Objective)
	}
	fmt.Fprintln(w)
	for _, s := range r.Steps {
		mark := "ok"
		if !s.OK {
			mark = "x "
		}
		fmt.Fprintf(w, "%s %-36s %8s\n", mark, s.Name, s.Duration)
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Developer work:        %10d actions\n", r.Signals.HumanActions)
	fmt.Fprintf(w, "Required parameters:   %10d\n", r.Signals.RequiredParameters)
	fmt.Fprintf(w, "Platform work:         %10s\n", formatMS(r.Signals.TotalTimeMS))
	fmt.Fprintf(w, "Waiting time:          %10s\n", formatMS(r.Signals.WaitingMS))
	fmt.Fprintf(w, "Tool transitions:      %10d\n", r.Signals.ToolTransitions)
	fmt.Fprintf(w, "Manual approvals:      %10d\n", r.Signals.Approvals)
	fmt.Fprintf(w, "Retries:               %10d\n", r.Signals.Retries)
	fmt.Fprintf(w, "Privilege escalations: %10d\n", r.Signals.PrivilegeEscalations)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Friction score:        %7.1f / %.0f\n\n", r.Score.Value, r.Score.Max)
	if r.SLO.Pass {
		fmt.Fprintln(w, "Developer Experience SLO: PASS")
		return nil
	}
	fmt.Fprintln(w, "Developer Experience SLO: FAIL")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Reason:")
	for _, b := range r.SLO.Breaches {
		fmt.Fprintf(w, "  %s: %s (budget %s)\n", b.Signal, b.Actual, b.Budget)
	}
	return nil
}

// Compare writes the human comparison report.
func Compare(w io.Writer, r compare.Report) error {
	fmt.Fprintln(w, "Developer Journey Regression")
	if r.Contract != "" || r.Objective != "" {
		fmt.Fprintf(w, "Contract:  %s\n", r.Contract)
		fmt.Fprintf(w, "Objective: %s\n", r.Objective)
	}
	fmt.Fprintln(w)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "\tbaseline\tcurrent\tdelta\n")
	for _, row := range r.Rows {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", row.Signal, row.Baseline, row.Current, row.Delta)
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	fmt.Fprintln(w)
	if r.Pass {
		fmt.Fprintln(w, "PASS")
		if r.Notes != "" {
			fmt.Fprintln(w)
			fmt.Fprintln(w, r.Notes)
		}
		return nil
	}
	fmt.Fprintln(w, "FAILED")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Reason:")
	fmt.Fprintln(w, indent(r.Reason, "  "))
	if r.Notes != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, r.Notes)
	}
	return nil
}

// List writes the journey catalogue.
func List(w io.Writer, journeys []journey.Journey, budgets map[string]string) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "JOURNEY\tOBJECTIVE\tSTEPS\tBUDGET\n")
	for _, j := range journeys {
		budget := budgets[j.BudgetName()]
		if budget == "" {
			budget = "missing"
		}
		fmt.Fprintf(tw, "%s\t%s\t%d\t%s\n", j.Name, j.Objective, len(j.Steps), budget)
	}
	return tw.Flush()
}

// JSON writes v as indented JSON.
func JSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func formatMS(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000.0)
}

func indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}
