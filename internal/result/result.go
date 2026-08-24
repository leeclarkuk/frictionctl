package result

import "time"

// Result is the JSON-serialisable outcome of a journey run.
type Result struct {
	Journey    string    `json:"journey"`
	Objective  string    `json:"objective"`
	StartedAt  time.Time `json:"started_at"`
	Duration   string    `json:"duration"`
	DurationMS int64     `json:"duration_ms"`
	Workdir    string    `json:"workdir,omitempty"`
	Steps      []Step    `json:"steps"`
	Signals    Signals   `json:"signals"`
	Score      Score     `json:"score"`
	SLO        SLO       `json:"slo"`
	Trace      Trace     `json:"trace"`
}

// Step is one executed journey step.
type Step struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	OK         bool    `json:"ok"`
	Duration   string  `json:"duration"`
	DurationMS int64   `json:"duration_ms"`
	Retries    int     `json:"retries"`
	Tool       string  `json:"tool,omitempty"`
	Waiting    bool    `json:"waiting"`
	Signals    Signals `json:"signals"`
	Error      string  `json:"error,omitempty"`
}

// Signals are the observable proxies collected from a journey.
type Signals struct {
	HumanActions         int   `json:"human_actions"`
	RequiredParameters   int   `json:"required_parameters"`
	ToolTransitions      int   `json:"tool_transitions"`
	Retries              int   `json:"retries"`
	Approvals            int   `json:"approvals"`
	PrivilegeEscalations int   `json:"privilege_escalations"`
	DocumentationLookups int   `json:"documentation_lookups"`
	EscapeHatchUsage     int   `json:"escape_hatch_usage"`
	WaitingMS            int64 `json:"waiting_ms"`
	TotalTimeMS          int64 `json:"total_time_ms"`
}

// Score is a transparent weighted sum, capped at 100. Lower is better.
type Score struct {
	Value         float64        `json:"value"`
	Max           float64        `json:"max"`
	Contributions []Contribution `json:"contributions"`
}

// Contribution explains one term in the score.
type Contribution struct {
	Signal string  `json:"signal"`
	Value  float64 `json:"value"`
	Weight float64 `json:"weight"`
	Points float64 `json:"points"`
}

// SLO is the budget evaluation. Pass/fail is decided here, not by the score.
type SLO struct {
	Pass     bool     `json:"pass"`
	Breaches []Breach `json:"breaches,omitempty"`
}

// Breach is one budget that was exceeded.
type Breach struct {
	Signal string `json:"signal"`
	Actual string `json:"actual"`
	Budget string `json:"budget"`
}

// Trace is a nested span tree for the journey. OTLP export is out of scope.
type Trace struct {
	Name       string `json:"name"`
	DurationMS int64  `json:"duration_ms"`
	Spans      []Span `json:"spans"`
}

// Span is one step in the journey trace.
type Span struct {
	Name       string `json:"name"`
	DurationMS int64  `json:"duration_ms"`
	OK         bool   `json:"ok"`
}
