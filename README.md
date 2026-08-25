# frictionctl

frictionctl is synthetic monitoring for developer experience.

Not a developer portal. Not a productivity dashboard. Not surveillance. Not an assistant.

It runs a golden-path journey the way synthetic probes run a customer request, records observable friction, and fails CI when a platform change makes engineers do more work.

```
Platform change
      │
      ▼
Golden-path journey
      │
      ├── correctness
      ├── security
      ├── reliability
      └── developer friction
              │
              ▼
        friction budget
              │
       PASS ───┴─── FAIL
                      │
                      ▼
                 PR blocked
```

## The contract

Developer experience is a property of the platform. Surveys, DORA numbers and portal adoption are useful, but they lag and they measure people. frictionctl measures the path.

Two implementations can share one contract. `create-service` and `create-service-frictionful` are different journeys. They are the same SLO: contract `create-service`, objective `running-service`. Compare will not silently score two unrelated journeys against each other. That is an error, not a regression.

## What fails a change

| Condition | Compare |
| --- | --- |
| Friction budget breached | FAIL |
| Declared discrete signal up | FAIL |
| Total or waiting time up | REPORT |
| Composite score up | REPORT unless `--gate-score` or `--gate-time` |

Budgets are policy. The score is explanation. Waiting time still contributes to the score, because it is a real cost. It does not fail a comparison by stealth when a CI runner is slow.

Discrete signals: `human_actions`, `required_parameters`, `tool_transitions`, `retries`, `approvals`, `privilege_escalations`, `documentation_lookups`, `escape_hatch_usage`.

`tool_transitions` counts context switches. One tool throughout is 0. `demo-platform → demo-kube → demo-platform` is 2.

`total_time: 0s` is a real zero. Omitting `total_time` leaves it unconstrained.

It does not watch engineers. It does not count commits. If a signal cannot be observed or honestly declared, it is not in the score.

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

Lower is better. The weights live in code and in `frictionctl explain`.

## Install

```bash
go install github.com/leeclarkuk/frictionctl/cmd/frictionctl@v0.1.0
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

`compare` accepts `--gate-score` and `--gate-time` if you really want wall time or the composite score as a merge gate. Default is not to.

Exit codes: `0` pass, `1` SLO or regression fail, `2` operational error, including a contract or objective mismatch.

## Proof in this repository

The local `create-service` golden path scaffolds a tiny Go service, builds it, writes deploy config, and records a fake deploy. There is no cluster. The frictionful variant is the same contract with extra flags, a second tool, and a retry.

```bash
go build -o demo-platform ./cmd/demo-platform
go build -o demo-kube ./cmd/demo-kube
export PATH="$PWD:$PATH"

./frictionctl run create-service --dir examples/create-service --output paved.json
./frictionctl run create-service-frictionful --dir examples/create-service --output frictionful.json
./frictionctl compare paved.json frictionful.json
```

The paved path must pass. The frictionful path must fail the shared budget. Compare must fail. A slower copy of the paved result must not fail compare. Tests cover that loop by building all three binaries.

## Writing a journey

Signals a shell cannot see (approvals, docs lookups, privilege, escape hatches) are declared on the step. Timing and retries are measured. Set `budget:` when the implementation name is not the contract name.

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
  tool_transitions: 1
  approvals: 0
  retries: 0
  privilege_escalations: 0
```

JSON schemas live in [`schemas/`](schemas/).

## What this is not, and what comes next

No autotuning. No generated platform PRs. No Kubernetes adapters for their own sake. No OTLP export yet. No surveillance.

The next honest step is to point frictionctl at a real golden path, keep a baseline, and let CI reject a fixture that adds work. That is when the category stops being a thesis.

## Licence

Apache License 2.0. Copyright 2026 Lee Clark.
