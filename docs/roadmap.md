# Roadmap

> 🇺🇸 English · [🇧🇷 Português](pt-BR/roadmap.md)

What the protocol does not have yet, found by using it. Each item is an `idea` task in this
repository's own `.trilha/tasks/` (`trilha-spec task list --status idea`); it becomes a spec
under `.trilha/specs/` when it is picked up (adr/003), and the protocol version bumps when a field or a
transition changes (§9 of the protocol).

## Open

Each one is an `idea` task here (`trilha-spec task list --status idea`) and an item of the
stack roadmap (`trilha-cloud/docs/ROADMAP-STACK.md`, phase 4). None is started.

| Task | Roadmap | Gap |
|---|---|---|
| TASK-024 | RM-050 | `failure_class` is a recommended vocabulary (spec 014), not an enum; make it one in `execution/contract.go` and the schema, validate it in `doctor`, bump to 0.4. |
| TASK-025 | RM-052 | The MCP write surface stops at `trilha_evidence` (`note`/`artifact`) and an unsigned `trilha_attest`; add `run`/`eval`, `--sign-key`, `trilha_task_add`. |
| TASK-026 | RM-053 | The execution contract has `cancelled` and `preview`; the task life cycle (§4) has no `cancelled` and Preview lives only in `CONTEXT.md`. |
| TASK-027 | RM-054 | `task.Resolver` has no remote implementation; the runner resolves on its own. One resolution path, tested against the interface. |
| TASK-028 | RM-055 | The path to 1.0: JSON Schema for every front matter, `doctor --schema`, a compatibility section and a public conformance command. |

Phase 0 of the same roadmap (spec 015: CI formatting the whole module, a `conformance` job
against the runner, releases with binaries on `v*` tags, adr/003 on the single source of truth
and the `spec-tasks-done` doctor warning) is `.trilha/specs/015-fase-0-do-roadmap.md`.

## Delivered in 0.4

### 0.4.0 — protocol 0.4 (spec 016)

What the trilha-cloud plan drawn from dev-3.0 needed on the wire (`trilha-cloud/docs/PLANO-DEV3.md`,
item DV-00): agent state from hooks, an attention badge, a run that stops to ask (`needs_input`)
and the answer that resumes its session, task memory between runs (overview and quoted notes),
diff counts, review and hunt runs with anchored findings and no code, human review comments and
the agent's replies, variant groups and report artifacts. One breaking change: the status
`needs_input`.

## Delivered in 0.3

### 0.3.1 — 2026-10-25

The first release with binaries (linux and macOS, amd64 and arm64, with `sha256sums.txt`); the
protocol is still 0.3, no field or transition changed.

- **Failure taxonomy** (spec 014, #15): §10 documents a recommended vocabulary for
  `failure_class` and `error_code` (`environment_missing_tool`, `test_failure`, `lint_failure`,
  `timeout`, `provider_transient`, `agent_error`), not yet a closed enum.
- **`doctor` warns `spec-tasks-done`** when an `approved` spec has every task `done` (spec 015).
- **One source of truth** (ADR 003): `.trilha/` is authoritative in trilha-spec, trilha-runner
  and trilha-cloud; Spec Kit folders are frozen history.
- **CI** formats the whole module and checks the execution contract against trilha-runner.

### 0.3.0

These came from planning a public-sector delivery programme that spans four repositories — the
product app, the framework, the runner and the control plane — and has to be auditable by the
buyer at every milestone. Each one is a spec under `specs/` and
a `done` task with evidence.

| Task | Gap | Why it mattered |
|---|---|---|
| TASK-016 · spec 009 · [#10](https://github.com/emersonjoe/trilha-spec/issues/10) | When the source of scope is an **external document** — a tender's requirement list, a law, a KPI in a contract — nothing links tasks to it. | "Which requirements are covered, by which task, with what evidence" was a spreadsheet kept by hand, and it is the question a public buyer asks at every milestone. |
| TASK-018 · spec 010 · [#12](https://github.com/emersonjoe/trilha-spec/issues/12) | The protocol had statuses and dependencies but **no calendar**: no way to say "this task belongs to M2, due 2027-04-30". | A contract with paid milestones reported its physical-financial schedule outside the protocol. |
| TASK-014 · spec 011 · [#8](https://github.com/emersonjoe/trilha-spec/issues/8) | Quality is measured with **numbers**, and the only way to record one was a `check` whose exit code hid it. | The reviewer could not see the value, nothing could trend it, and a control plane could not gate on it. |
| TASK-015 · spec 012 · [#9](https://github.com/emersonjoe/trilha-spec/issues/9) | `review → done` was one person's decision, and nothing distinguished "someone typed a note" from "this person, in this role, **attests** this — signed". | Public-sector delivery ends with named human acceptance, sometimes by a committee. |
| TASK-017 · spec 013 · [#11](https://github.com/emersonjoe/trilha-spec/issues/11) | `depends_on` was local only, so a programme spanning repositories wrote its real dependencies as **prose**. | `next` offered work that could not start, and nobody could draw the graph. |

## Delivered in 0.2

The items below came from writing trilha-cloud's spec 011 (project circuit breaker and fleet
pause) with `trilha-spec` instead of spec-kit. Each one is a spec under `specs/` and a `done`
task with evidence.

| Task | Gap | Why it mattered |
|---|---|---|
| TASK-009 · spec 005 · [#1](https://github.com/emersonjoe/trilha-spec/issues/1) | A spec has no place for **security impact**: assets, trust boundaries, affected controls (ASVS), evidence commands. trilha-cloud's constitution requires them, so they live as free sections in the body that `doctor` and the context pack do not see. | A reviewer decides from evidence; the security evidence should be as visible to the tools as `acceptance` and `checks` are. |
| TASK-010 · spec 004 · [#2](https://github.com/emersonjoe/trilha-spec/issues/2) | Spec **status** is `draft \| approved \| done`; nothing for `rejected` or `superseded`, no relation between specs (`supersedes`, `depends_on`), and the CLI cannot change a spec's status or `issue` (`spec new \| list \| show` only). | Specs are edited by hand once created; the tools cannot answer "which spec replaced this one". |
| TASK-011 · spec 006 · [#3](https://github.com/emersonjoe/trilha-spec/issues/3) | `project.md` has no **limits** (`max_cost_per_hour`, `max_failure_rate`, `max_repeated_failure_class`) and no **paused** state. A task carries `token_budget`; a project carries nothing. | Every control plane invents its own circuit breaker, and a runner cannot read the same thresholds before it starts. |
| TASK-012 · spec 007 · [#4](https://github.com/emersonjoe/trilha-spec/issues/4) | `run` evidence has a free `meta{}`; **cost** is not standard (`provider`, `model`, `tokens_in`, `tokens_out`, `cost`, `currency`). | Two runners report cost differently; a control plane cannot reconcile a ledger against the provider's invoice. |
| TASK-013 · spec 003 · [#5](https://github.com/emersonjoe/trilha-spec/issues/5) | `task add` has no `--body`; `spec new` and `task add` write an English template only; `--status` works but is not in `--help`. | A pt-BR project writes the spec by hand right after creating it; the CLI could do the whole job. |

### Earlier, also delivered

| Task | Gap |
|---|---|
| TASK-007 · spec 002 · [#6](https://github.com/emersonjoe/trilha-spec/issues/6) | CLI messages in pt-BR via `TRILHA_LANG`. |
| TASK-008 · spec 008 · [#7](https://github.com/emersonjoe/trilha-spec/issues/7) | Signed evidence records, for a remote runner whose evidence crosses a trust boundary. |

## Resolved

The `trilha` framework's dev server used to write `*` in `.trilha/.gitignore`, hiding the
protocol from git. `trilha` 0.124.0 (2026-09-12) moved its build cache to `.trilha/cache/`,
and the scaffold's `.gitignore` now ignores only `.trilha/cache/` and `.trilha/runs/`, as
[adr/001](adr/001-tres-repositorios.md) asked. `trilha-spec doctor` still reports a
`.gitignore` with `*` and `init` still rewrites it, for projects created before that release.
