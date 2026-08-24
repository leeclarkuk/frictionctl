package journey

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Journey is a named developer outcome and the steps required to reach it.
type Journey struct {
	Name      string            `yaml:"name"`
	Objective string            `yaml:"objective"`
	Budget    string            `yaml:"budget"`
	Env       map[string]string `yaml:"env"`
	Steps     []Step            `yaml:"steps"`
	Source    string            `yaml:"-"`
}

// Step is one action in a journey.
type Step struct {
	ID         string            `yaml:"id"`
	Name       string            `yaml:"name"`
	Adapter    string            `yaml:"adapter"`
	Command    []string          `yaml:"command"`
	Workdir    string            `yaml:"workdir"`
	Env        map[string]string `yaml:"env"`
	MaxRetries int               `yaml:"max_retries"`
	IsWaiting  *bool             `yaml:"waiting"`
	Signals    Signals           `yaml:"signals"`
}

// Signals declared on a step. Timing and retries are measured at run time.
type Signals struct {
	HumanActions         int    `yaml:"human_actions"`
	RequiredParameters   int    `yaml:"required_parameters"`
	Tool                 string `yaml:"tool"`
	Approvals            int    `yaml:"approvals"`
	PrivilegeEscalations int    `yaml:"privilege_escalations"`
	DocumentationLookups int    `yaml:"documentation_lookups"`
	EscapeHatchUsage     int    `yaml:"escape_hatch_usage"`
}

// CountsAsWaiting reports whether the step blocks the engineer on platform work.
// Default is true: if the platform is running, the engineer is waiting.
func (s Step) CountsAsWaiting() bool {
	if s.IsWaiting == nil {
		return true
	}
	return *s.IsWaiting
}

// LoadFile reads a single journey YAML file.
func LoadFile(path string) (Journey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Journey{}, fmt.Errorf("read journey %s: %w", path, err)
	}
	var j Journey
	if err := yaml.Unmarshal(raw, &j); err != nil {
		return Journey{}, fmt.Errorf("parse journey %s: %w", path, err)
	}
	j.Source = path
	if err := j.Validate(); err != nil {
		return Journey{}, fmt.Errorf("validate journey %s: %w", path, err)
	}
	return j, nil
}

// Validate checks required fields and unique step IDs.
func (j Journey) Validate() error {
	if strings.TrimSpace(j.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(j.Objective) == "" {
		return fmt.Errorf("objective is required")
	}
	if len(j.Steps) == 0 {
		return fmt.Errorf("at least one step is required")
	}
	seen := make(map[string]struct{}, len(j.Steps))
	for i, s := range j.Steps {
		if strings.TrimSpace(s.ID) == "" {
			return fmt.Errorf("step %d: id is required", i)
		}
		if strings.TrimSpace(s.Name) == "" {
			return fmt.Errorf("step %q: name is required", s.ID)
		}
		if _, ok := seen[s.ID]; ok {
			return fmt.Errorf("duplicate step id %q", s.ID)
		}
		seen[s.ID] = struct{}{}
		adapter := s.Adapter
		if adapter == "" {
			adapter = "shell"
		}
		if adapter != "shell" {
			return fmt.Errorf("step %q: unsupported adapter %q", s.ID, s.Adapter)
		}
		if len(s.Command) == 0 {
			return fmt.Errorf("step %q: command is required", s.ID)
		}
		if s.MaxRetries < 0 {
			return fmt.Errorf("step %q: max_retries cannot be negative", s.ID)
		}
	}
	return nil
}

// BudgetName is the budget file stem to load. Defaults to the journey name.
func (j Journey) BudgetName() string {
	if strings.TrimSpace(j.Budget) != "" {
		return j.Budget
	}
	return j.Name
}

// LoadDir loads every journey YAML under dir/journeys and dir itself.
func LoadDir(dir string) ([]Journey, error) {
	var paths []string
	for _, pattern := range []string{
		filepath.Join(dir, "journeys", "*.yaml"),
		filepath.Join(dir, "journeys", "*.yml"),
		filepath.Join(dir, "journey.yaml"),
		filepath.Join(dir, "journey.yml"),
		filepath.Join(dir, "*.journey.yaml"),
	} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		paths = append(paths, matches...)
	}
	seen := make(map[string]struct{})
	var journeys []Journey
	for _, p := range paths {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		j, err := LoadFile(p)
		if err != nil {
			return nil, err
		}
		journeys = append(journeys, j)
	}
	return journeys, nil
}

// Find loads the named journey from dir.
func Find(dir, name string) (Journey, error) {
	if name == "" {
		return Journey{}, fmt.Errorf("journey name is required")
	}
	candidates := []string{
		filepath.Join(dir, "journeys", name+".yaml"),
		filepath.Join(dir, "journeys", name+".yml"),
		filepath.Join(dir, name+".yaml"),
		filepath.Join(dir, "journey.yaml"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		j, err := LoadFile(p)
		if err != nil {
			return Journey{}, err
		}
		if j.Name == name || filepath.Base(strings.TrimSuffix(p, filepath.Ext(p))) == name {
			return j, nil
		}
		if filepath.Base(p) == "journey.yaml" && j.Name == name {
			return j, nil
		}
	}
	all, err := LoadDir(dir)
	if err != nil {
		return Journey{}, err
	}
	for _, j := range all {
		if j.Name == name {
			return j, nil
		}
	}
	return Journey{}, fmt.Errorf("journey %q not found in %s", name, dir)
}
