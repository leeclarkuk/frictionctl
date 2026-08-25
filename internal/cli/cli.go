package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/leeclarkuk/frictionctl/internal/budget"
	"github.com/leeclarkuk/frictionctl/internal/compare"
	"github.com/leeclarkuk/frictionctl/internal/journey"
	"github.com/leeclarkuk/frictionctl/internal/report"
	"github.com/leeclarkuk/frictionctl/internal/result"
	"github.com/leeclarkuk/frictionctl/internal/runner"
	"github.com/leeclarkuk/frictionctl/internal/score"
	"github.com/leeclarkuk/frictionctl/internal/version"
	"github.com/spf13/cobra"
)

const (
	ExitOK  = 0
	ExitSLO = 1
	ExitErr = 2
)

type options struct {
	dir    string
	json   bool
	stdout io.Writer
	stderr io.Writer
}

type exitError struct {
	code int
	msg  string
}

func (e exitError) Error() string { return e.msg }

// Execute runs the CLI and returns a process exit code.
func Execute() int {
	return ExecuteWith(os.Args[1:], os.Stdout, os.Stderr)
}

// ExecuteWith is used by tests.
func ExecuteWith(args []string, stdout, stderr io.Writer) int {
	opt := &options{stdout: stdout, stderr: stderr}
	root := newRoot(opt)
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)
	if err := root.Execute(); err != nil {
		var ex exitError
		if errors.As(err, &ex) {
			if ex.msg != "" {
				fmt.Fprintln(stderr, ex.msg)
			}
			return ex.code
		}
		fmt.Fprintln(stderr, err.Error())
		return ExitErr
	}
	return ExitOK
}

func newRoot(opt *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "frictionctl",
		Short:         "Synthetic monitoring for developer experience",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.PersistentFlags().StringVarP(&opt.dir, "dir", "C", ".", "directory containing journeys/ and budgets/")
	cmd.PersistentFlags().BoolVar(&opt.json, "json", false, "write JSON to stdout")
	cmd.AddCommand(runCmd(opt), compareCmd(opt), listCmd(opt), explainCmd(opt), versionCmd(opt))
	return cmd
}

func runCmd(opt *options) *cobra.Command {
	var (
		budgetPath string
		output     string
		workdir    string
		keep       bool
	)
	cmd := &cobra.Command{
		Use:   "run JOURNEY",
		Short: "Run a developer journey and evaluate its friction budget",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			j, err := journey.Find(opt.dir, args[0])
			if err != nil {
				return err
			}
			var b budget.File
			if budgetPath != "" {
				b, err = budget.LoadFile(budgetPath)
			} else {
				b, err = budget.Find(opt.dir, j.BudgetName())
			}
			if err != nil {
				return err
			}
			res, runErr := runner.Runner{}.Run(cmd.Context(), j, b, runner.Options{
				Workdir: workdir,
				Keep:    keep,
			})
			if output != "" {
				if err := result.WriteFile(output, res); err != nil {
					return err
				}
			}
			if opt.json {
				if err := report.JSON(opt.stdout, res); err != nil {
					return err
				}
			} else if err := report.Journey(opt.stdout, res); err != nil {
				return err
			}
			if runErr != nil {
				return exitError{code: ExitErr, msg: runErr.Error()}
			}
			if !res.SLO.Pass {
				return exitError{code: ExitSLO}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&budgetPath, "budget", "", "budget YAML path (defaults to matching file in --dir)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "write the JSON result to this path")
	cmd.Flags().StringVar(&workdir, "workdir", "", "working directory for the journey (default: temp dir)")
	cmd.Flags().BoolVar(&keep, "keep", false, "keep a generated temp workdir")
	return cmd
}

func compareCmd(opt *options) *cobra.Command {
	var (
		gateScore bool
		gateTime  bool
	)
	cmd := &cobra.Command{
		Use:   "compare BASELINE.json CURRENT.json",
		Short: "Compare two journey results and fail on friction regressions",
		Long: `Compare two results for the same contract and objective.

Budgets and increases in declared discrete signals fail the comparison
(exit 1). Total time, waiting time and the composite score are reported.
They fail the comparison only with --gate-time or --gate-score.

A contract or objective mismatch is an error (exit 2), not a regression.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			baseline, err := result.LoadFile(args[0])
			if err != nil {
				return err
			}
			current, err := result.LoadFile(args[1])
			if err != nil {
				return err
			}
			if err := compare.CheckIdentity(baseline, current); err != nil {
				return exitError{code: ExitErr, msg: err.Error()}
			}
			rep := compare.Diff(baseline, current, compare.Options{
				GateScore: gateScore,
				GateTime:  gateTime,
			})
			if opt.json {
				if err := report.JSON(opt.stdout, rep); err != nil {
					return err
				}
			} else if err := report.Compare(opt.stdout, rep); err != nil {
				return err
			}
			if !rep.Pass {
				return exitError{code: ExitSLO}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&gateScore, "gate-score", false, "fail when the composite friction score increases")
	cmd.Flags().BoolVar(&gateTime, "gate-time", false, "fail when total or waiting time increases")
	return cmd
}

func listCmd(opt *options) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List journeys in --dir",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			journeys, err := journey.LoadDir(opt.dir)
			if err != nil {
				return err
			}
			sort.Slice(journeys, func(i, j int) bool { return journeys[i].Name < journeys[j].Name })
			found := make(map[string]string, len(journeys))
			for _, j := range journeys {
				if b, err := budget.Find(opt.dir, j.BudgetName()); err == nil {
					found[j.BudgetName()] = b.Source
				}
			}
			if opt.json {
				type row struct {
					Name      string `json:"name"`
					Objective string `json:"objective"`
					Steps     int    `json:"steps"`
					Budget    string `json:"budget"`
				}
				rows := make([]row, 0, len(journeys))
				for _, j := range journeys {
					b := "missing"
					if src, ok := found[j.BudgetName()]; ok {
						b = src
					}
					rows = append(rows, row{Name: j.Name, Objective: j.Objective, Steps: len(j.Steps), Budget: b})
				}
				return report.JSON(opt.stdout, rows)
			}
			return report.List(opt.stdout, journeys, found)
		},
	}
}

func explainCmd(opt *options) *cobra.Command {
	return &cobra.Command{
		Use:   "explain [JOURNEY]",
		Short: "Show how the friction score is computed",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprint(opt.stdout, score.FormulaHelp())
			if len(args) == 0 {
				return nil
			}
			j, err := journey.Find(opt.dir, args[0])
			if err != nil {
				return err
			}
			var s result.Signals
			var tools []string
			for _, step := range j.Steps {
				s.HumanActions += step.Signals.HumanActions
				s.RequiredParameters += step.Signals.RequiredParameters
				s.Approvals += step.Signals.Approvals
				s.PrivilegeEscalations += step.Signals.PrivilegeEscalations
				s.DocumentationLookups += step.Signals.DocumentationLookups
				s.EscapeHatchUsage += step.Signals.EscapeHatchUsage
				tool := step.Signals.Tool
				if tool == "" && len(step.Command) > 0 {
					tool = step.Command[0]
				}
				tools = append(tools, tool)
			}
			s.ToolTransitions = result.CountToolTransitions(tools)
			fmt.Fprintf(opt.stdout, "\nDeclared signals for %s (before a run; waiting time and retries are measured):\n", j.Name)
			fmt.Fprintf(opt.stdout, "  human_actions:          %d\n", s.HumanActions)
			fmt.Fprintf(opt.stdout, "  required_parameters:    %d\n", s.RequiredParameters)
			fmt.Fprintf(opt.stdout, "  tool_transitions:       %d\n", s.ToolTransitions)
			fmt.Fprintf(opt.stdout, "  approvals:              %d\n", s.Approvals)
			fmt.Fprintf(opt.stdout, "  privilege_escalations:  %d\n", s.PrivilegeEscalations)
			fmt.Fprintf(opt.stdout, "  documentation_lookups:  %d\n", s.DocumentationLookups)
			fmt.Fprintf(opt.stdout, "  escape_hatch_usage:     %d\n", s.EscapeHatchUsage)
			fmt.Fprintf(opt.stdout, "\nStatic partial score (waiting=0, retries=0): %.1f / 100\n", score.Compute(s).Value)
			return nil
		},
	}
}

func versionCmd(opt *options) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the frictionctl version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(opt.stdout, version.Version)
			return nil
		},
	}
}
