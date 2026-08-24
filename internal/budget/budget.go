package budget

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/leeclarkuk/frictionctl/internal/result"
	"gopkg.in/yaml.v3"
)

// File is a friction budget for one journey. Missing fields are unconstrained.
type File struct {
	Journey   string `yaml:"journey"`
	Objective string `yaml:"objective"`
	Budgets   Limits `yaml:"budgets"`
	Source    string `yaml:"-"`
}

// Limits are hard SLO gates. The composite score is not the pass/fail rule
// unless FrictionScore is set.
type Limits struct {
	TotalTime            Duration `yaml:"total_time"`
	HumanActions         *int     `yaml:"human_actions"`
	RequiredParameters   *int     `yaml:"required_parameters"`
	ToolTransitions      *int     `yaml:"tool_transitions"`
	Approvals            *int     `yaml:"approvals"`
	Retries              *int     `yaml:"retries"`
	PrivilegeEscalations *int     `yaml:"privilege_escalations"`
	DocumentationLookups *int     `yaml:"documentation_lookups"`
	EscapeHatchUsage     *int     `yaml:"escape_hatch_usage"`
	FrictionScore        *float64 `yaml:"friction_score"`
}

// Duration wraps time.Duration for YAML values such as 1m, 90s, 5m.
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.ScalarNode {
		return fmt.Errorf("duration must be a string")
	}
	if strings.TrimSpace(value.Value) == "" {
		d.Duration = 0
		return nil
	}
	parsed, err := time.ParseDuration(value.Value)
	if err != nil {
		return fmt.Errorf("parse duration %q: %w", value.Value, err)
	}
	d.Duration = parsed
	return nil
}

// LoadFile reads a budget YAML file.
func LoadFile(path string) (File, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return File{}, fmt.Errorf("read budget %s: %w", path, err)
	}
	var f File
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return File{}, fmt.Errorf("parse budget %s: %w", path, err)
	}
	f.Source = path
	if err := f.Validate(); err != nil {
		return File{}, fmt.Errorf("validate budget %s: %w", path, err)
	}
	return f, nil
}

// Validate checks required fields.
func (f File) Validate() error {
	if strings.TrimSpace(f.Journey) == "" {
		return fmt.Errorf("journey is required")
	}
	if strings.TrimSpace(f.Objective) == "" {
		return fmt.Errorf("objective is required")
	}
	return nil
}

// Find loads the budget for a journey name from dir.
func Find(dir, name string) (File, error) {
	candidates := []string{
		filepath.Join(dir, "budgets", name+".yaml"),
		filepath.Join(dir, "budgets", name+".yml"),
		filepath.Join(dir, "budget.yaml"),
		filepath.Join(dir, "budget.yml"),
		filepath.Join(dir, name+".budget.yaml"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		return LoadFile(p)
	}
	return File{}, fmt.Errorf("budget %q not found in %s", name, dir)
}

// Evaluate compares measured signals against the budget.
func Evaluate(limits Limits, signals result.Signals, score float64) result.SLO {
	var breaches []result.Breach
	if limits.TotalTime.Duration > 0 && time.Duration(signals.TotalTimeMS)*time.Millisecond > limits.TotalTime.Duration {
		breaches = append(breaches, result.Breach{
			Signal: "total_time",
			Actual: formatMS(signals.TotalTimeMS),
			Budget: limits.TotalTime.Duration.String(),
		})
	}
	breaches = appendIntBreach(breaches, "human_actions", signals.HumanActions, limits.HumanActions)
	breaches = appendIntBreach(breaches, "required_parameters", signals.RequiredParameters, limits.RequiredParameters)
	breaches = appendIntBreach(breaches, "tool_transitions", signals.ToolTransitions, limits.ToolTransitions)
	breaches = appendIntBreach(breaches, "approvals", signals.Approvals, limits.Approvals)
	breaches = appendIntBreach(breaches, "retries", signals.Retries, limits.Retries)
	breaches = appendIntBreach(breaches, "privilege_escalations", signals.PrivilegeEscalations, limits.PrivilegeEscalations)
	breaches = appendIntBreach(breaches, "documentation_lookups", signals.DocumentationLookups, limits.DocumentationLookups)
	breaches = appendIntBreach(breaches, "escape_hatch_usage", signals.EscapeHatchUsage, limits.EscapeHatchUsage)
	if limits.FrictionScore != nil && score > *limits.FrictionScore {
		breaches = append(breaches, result.Breach{
			Signal: "friction_score",
			Actual: strconv.FormatFloat(score, 'f', 1, 64),
			Budget: strconv.FormatFloat(*limits.FrictionScore, 'f', 1, 64),
		})
	}
	return result.SLO{Pass: len(breaches) == 0, Breaches: breaches}
}

func appendIntBreach(breaches []result.Breach, name string, actual int, limit *int) []result.Breach {
	if limit == nil {
		return breaches
	}
	if actual > *limit {
		return append(breaches, result.Breach{
			Signal: name,
			Actual: strconv.Itoa(actual),
			Budget: strconv.Itoa(*limit),
		})
	}
	return breaches
}

func formatMS(ms int64) string {
	return (time.Duration(ms) * time.Millisecond).Round(time.Millisecond).String()
}
