package runner

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/leeclarkuk/frictionctl/internal/adapter"
	"github.com/leeclarkuk/frictionctl/internal/budget"
	"github.com/leeclarkuk/frictionctl/internal/journey"
	"github.com/leeclarkuk/frictionctl/internal/result"
	"github.com/leeclarkuk/frictionctl/internal/score"
	"github.com/leeclarkuk/frictionctl/internal/subst"
)

// Runner executes a journey through an adapter.
type Runner struct {
	Adapter adapter.Adapter
}

// Options control a single run.
type Options struct {
	Workdir string
	Keep    bool
}

// Run executes j against b. A failed step after retries is an error, not an SLO fail.
func (r Runner) Run(ctx context.Context, j journey.Journey, b budget.File, opt Options) (result.Result, error) {
	if r.Adapter == nil {
		r.Adapter = adapter.Shell{}
	}
	workdir := opt.Workdir
	cleanup := false
	if workdir == "" {
		dir, err := os.MkdirTemp("", "frictionctl-")
		if err != nil {
			return result.Result{}, fmt.Errorf("create workdir: %w", err)
		}
		workdir = dir
		cleanup = !opt.Keep
	} else if err := os.MkdirAll(workdir, 0o755); err != nil {
		return result.Result{}, fmt.Errorf("create workdir: %w", err)
	}
	if cleanup {
		defer os.RemoveAll(workdir)
	}

	started := time.Now()
	vars := subst.Merge(j.Env, map[string]string{
		"WORKDIR":          workdir,
		"FRICTION_WORKDIR": workdir,
		"JOURNEY":          j.Name,
	})

	out := result.Result{
		Journey:   j.Name,
		Objective: j.Objective,
		StartedAt: started.UTC(),
		Workdir:   workdir,
	}

	for _, step := range j.Steps {
		sr, err := r.runStep(ctx, step, vars)
		out.Steps = append(out.Steps, sr)
		if err != nil {
			finish(&out, started, b)
			return out, err
		}
	}
	finish(&out, started, b)
	return out, nil
}

func finish(out *result.Result, started time.Time, b budget.File) {
	out.DurationMS = time.Since(started).Milliseconds()
	out.Duration = formatDuration(out.DurationMS)
	out.Signals = aggregate(out.Steps, out.DurationMS)
	out.Score = score.Compute(out.Signals)
	out.SLO = budget.Evaluate(b.Budgets, out.Signals, out.Score.Value)
	out.Trace = traceOf(out)
}

func (r Runner) runStep(ctx context.Context, step journey.Step, vars map[string]string) (result.Step, error) {
	command := subst.ExpandAll(step.Command, vars)
	if err := subst.RequireExpanded("command", command); err != nil {
		return result.Step{}, fmt.Errorf("step %s: %w", step.ID, err)
	}
	dir := subst.Expand(step.Workdir, vars)
	env := subst.Merge(vars, step.Env)
	for k, v := range env {
		env[k] = subst.Expand(v, vars)
	}

	tool := step.Signals.Tool
	if tool == "" && len(command) > 0 {
		tool = command[0]
	}

	sr := result.Step{
		ID:      step.ID,
		Name:    step.Name,
		Tool:    tool,
		Waiting: step.CountsAsWaiting(),
		Signals: result.Signals{
			HumanActions:         step.Signals.HumanActions,
			RequiredParameters:   step.Signals.RequiredParameters,
			Approvals:            step.Signals.Approvals,
			PrivilegeEscalations: step.Signals.PrivilegeEscalations,
			DocumentationLookups: step.Signals.DocumentationLookups,
			EscapeHatchUsage:     step.Signals.EscapeHatchUsage,
		},
	}

	attempts := step.MaxRetries + 1
	if attempts < 1 {
		attempts = 1
	}
	start := time.Now()
	var lastErr string
	for i := 0; i < attempts; i++ {
		if i > 0 {
			sr.Retries++
		}
		outcome, err := r.Adapter.Run(ctx, adapter.Spec{
			Command: command,
			Dir:     dir,
			Env:     env,
		})
		if err != nil {
			lastErr = err.Error()
			continue
		}
		if outcome.ExitCode == 0 {
			completeStep(&sr, start, true, "")
			return sr, nil
		}
		lastErr = outcome.Stderr
		if lastErr == "" {
			lastErr = fmt.Sprintf("exit code %d", outcome.ExitCode)
		}
	}
	completeStep(&sr, start, false, lastErr)
	return sr, fmt.Errorf("step %s: %s", step.ID, lastErr)
}

func completeStep(sr *result.Step, start time.Time, ok bool, errMsg string) {
	sr.OK = ok
	sr.Error = errMsg
	sr.DurationMS = time.Since(start).Milliseconds()
	sr.Duration = formatDuration(sr.DurationMS)
	sr.Signals.Retries = sr.Retries
	if sr.Waiting {
		sr.Signals.WaitingMS = sr.DurationMS
	}
	sr.Signals.TotalTimeMS = sr.DurationMS
}

func aggregate(steps []result.Step, totalMS int64) result.Signals {
	var s result.Signals
	s.TotalTimeMS = totalMS
	s.ToolTransitions = toolTransitions(steps)
	for _, step := range steps {
		s.HumanActions += step.Signals.HumanActions
		s.RequiredParameters += step.Signals.RequiredParameters
		s.Retries += step.Retries
		s.Approvals += step.Signals.Approvals
		s.PrivilegeEscalations += step.Signals.PrivilegeEscalations
		s.DocumentationLookups += step.Signals.DocumentationLookups
		s.EscapeHatchUsage += step.Signals.EscapeHatchUsage
		if step.Waiting {
			s.WaitingMS += step.DurationMS
		}
	}
	return s
}

func toolTransitions(steps []result.Step) int {
	n := 0
	prev := ""
	for _, s := range steps {
		if s.Tool == "" {
			continue
		}
		if prev == "" {
			n = 1
			prev = s.Tool
			continue
		}
		if s.Tool != prev {
			n++
			prev = s.Tool
		}
	}
	return n
}

func traceOf(r *result.Result) result.Trace {
	t := result.Trace{
		Name:       "developer.journey." + r.Journey,
		DurationMS: r.DurationMS,
	}
	for _, s := range r.Steps {
		t.Spans = append(t.Spans, result.Span{
			Name:       s.ID,
			DurationMS: s.DurationMS,
			OK:         s.OK,
		})
	}
	return t
}

func formatDuration(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	if d < time.Second {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
