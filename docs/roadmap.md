# Roadmap

> 🇺🇸 English · [🇧🇷 Português](pt-BR/roadmap.md)

What the protocol does not have yet, found by using it. Each item is an `idea` task in this
repository's own `.trilha/tasks/` (`trilha-spec task list --status idea`); it becomes a spec
under `specs/` when it is picked up, and the protocol version bumps when a field or a
transition changes (§9 of the protocol).

## Open

Nothing at the moment: `trilha-spec task list --status idea` is empty. The next gaps will
come from the runner and the control plane using protocol 0.2.

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

## Known conflict

The `trilha` framework's dev server writes `*` in `.trilha/.gitignore`, hiding the protocol
from git; `trilha-spec doctor` reports it and `init` rewrites the file. The lasting fix is the
framework change recorded in [adr/001](adr/001-tres-repositorios.md).
