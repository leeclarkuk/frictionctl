# frictionctl

Developer experience is a property of the platform. A platform change can fail CI for making engineers do more work.

That is the whole product. Surveys, DORA dashboards and portal adoption numbers are useful, but they are lagging and they measure people. frictionctl measures the golden path. It runs a declared developer journey, records observable proxies, scores the friction in a formula you can read, and fails when a budget is breached or a change makes the path worse.

It does not watch engineers. It does not count commits. It does not invent a cognitive-load model. If a signal cannot be observed or honestly declared, it is not in the score.

## What it measures

| Signal | Meaning |
| --- | --- |
| `human_actions` | Work the platform failed to automate |
| `required_parameters` | Decisions pushed onto the developer |
| `tool_transitions` | Context switches between tools |
| `retries` | Failed attempts before the step succeeded |
| `approvals` | Manual gates |
| `privilege_escalations` | Broken self-service |
| `documentation_lookups` | The path was not legible |
| `escape_hatch_usage` | The golden path was not enough |
| `waiting_seconds` | Wall time the engineer is blocked on the platform |

Pass or fail is the budget, not the score. The score exists so a regression has magnitude, not just a boolean.

```
score = min(100,
    2.0 * human_actions
  + 1.0 * required_parameters
  + 1.5 * tool_transitions
  + 2.0 * retries
  + 3.0 * approvals
  + 4.0 * privilege_escalations
  + 5.0 * documentation_lookups
  + 3.0 * escape_hatch_usage
  + 0.01 * waiting_seconds
)
```

Lower is better. The weights live in code and in `frictionctl explain`. They are not tuned to flatter the demo.

## Install

```bash
go install github.com/leeclarkuk/frictionctl/cmd/frictionctl@latest
```

From a clone:

```bash
go build -o frictionctl ./cmd/frictionctl
```

## Commands

```
frictionctl run JOURNEY
frictionctl compare BASELINE.json CURRENT.json
frictionctl list
frictionctl explain [JOURNEY]
frictionctl version
```

`--dir` / `-C` points at a directory with `journeys/` and `budgets/`. `--json` writes machine output. `run` accepts `--output` for a result file, `--budget` to override the budget path, and `--workdir` if you want to keep the artefacts.

Exit codes: `0` pass, `1` SLO or regression fail, `2` operational error.

## Demo

The repository ships a local `create-service` golden path. It scaffolds a tiny Go service, builds it, writes deploy config, and records a fake deploy. There is no cluster.

You need the two demo backends on `PATH`:

```bash
go build -o demo-platform ./cmd/demo-platform
go build -o demo-kube ./cmd/demo-kube
export PATH="$PWD:$PATH"
```

Paved path, which should pass:

```bash
./frictionctl run create-service --dir examples/create-service --output paved.json
```

Frictionful path, which should fail the same budget. Extra flags, an extra tool, a retry:

```bash
./frictionctl run create-service-frictionful --dir examples/create-service --output frictionful.json
```

Compare them. This should fail:

```bash
./frictionctl compare paved.json frictionful.json
```

That loop is the point of the repository. If it does not work, nothing else here matters.

## Writing a journey

Signals that a shell cannot see (approvals, docs lookups, privilege, escape hatches) are declared on the step. Timing and retries are measured.

```yaml
name: create-service
objective: running-service
steps:
  - id: scaffold
    name: Scaffold repository
    command: ["demo-platform", "scaffold", "--name", "demo", "--out", "${WORKDIR}"]
    signals:
      human_actions: 1
      required_parameters: 1
      tool: demo-platform
```

`${WORKDIR}` is set by the runner. Commands are argv lists. There is no shell.

## Writing a budget

Missing fields are unconstrained. Zero is a real limit.

```yaml
journey: create-service
objective: running-service
budgets:
  total_time: 1m
  human_actions: 3
  tool_transitions: 2
  approvals: 0
  retries: 0
  privilege_escalations: 0
```

JSON schemas live in [`schemas/`](schemas/).

## What this is not

No autotuning. No generated platform PRs. No live Kubernetes, Argo CD, Terraform or GitHub adapters. No OTLP export (the result JSON carries a nested span tree; that is enough for now). No onboarding journey that needs an identity provider. No surveillance.

Those can wait until `run` and `compare` are honest.

## Licence

Apache License 2.0. Copyright 2026 Lee Clark.
