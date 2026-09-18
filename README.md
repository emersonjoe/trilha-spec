# trilha-spec

> 🇺🇸 English · [🇧🇷 Português](README.pt-BR.md)
>
> **Hands-on chapter on the Trilha site:** <https://emersonjoe.github.io/trilha/learn/agentic-protocol>

**An open protocol for describing work that software agents can execute.**

`trilha-spec` is the public, dependency-free half of [Trilha](https://github.com/emersonjoe/trilha)'s
agentic development strategy. It says *what* to do — specifications, tasks, the task graph,
the context an agent receives, the evidence a task must leave behind — and nothing about
*how* it runs. Execution (worktrees, sandboxes, workers, queues) is
[trilha-runner](https://github.com/emersonjoe/trilha-runner); control plane, fleet and
governance are trilha-cloud.

```
.trilha/
├── project.md        what this project is, for an agent that just arrived
├── constitution.md   the rules every task obeys
├── specs/            001-oauth.md — what to build and why
├── tasks/            TASK-001.md — executable units with acceptance criteria
├── agents/           coder.md, reviewer.md — who may execute, with what
├── context/          extra documents an agent receives
├── evidence/         TASK-001/001-check.json — proof a task produced
├── keys/             runner-01.pub — who may sign evidence; private keys never here
└── runs/             runner scratch; never committed
```

A programme that spans repositories may put an optional `program.md` above the checkouts,
naming them and the milestones they share; a single-repository project never needs one.

Everything is Markdown with a small front matter — readable by a person, a diff and a model —
and the module depends only on the Go standard library, so any tool can read it.

## A task

```markdown
---
id: TASK-002
title: Implement OAuth login
status: ready
spec: 001-oauth
milestone: M2
agent: coder
covers:
  - D2-R8
depends_on:
  - TASK-001
  - "trilha:TASK-004"
acceptance:
  - OAuth login works against the provider in context/
  - "metric: login_p95 <= 2"
checks:
  - go test ./...
  - sh -c "curl -sf localhost:3000/login | grep -q 'Sign in'"
review:
  quorum: 2
  roles: [uat, legal]
---

Use the provider described in `context/oauth.md`. Do not add a dependency.
```

A task moves through a strict life:

```
idea → spec → ready → running → verify → review → done
                 ↑         │        │        │
                 └── blocked / failed ←──────┘
```

`ready` needs acceptance criteria; `running` needs every dependency `done` — including one in
another repository, `trilha:TASK-004`; `verify` runs the checks and records each as
**evidence** — exit code, output hash, who ran it, when, and the numbers a harness printed —
and the reviewer decides from the evidence, not from a chat log. A task that declares a
`review` quorum does not close until that many named people have signed what they attest.

## The CLI

```bash
go install github.com/emersonjoe/trilha-spec/cmd/trilha-spec@latest

trilha-spec init --name my-app                 # .trilha/ with project, constitution, agents
trilha-spec spec new "OAuth login"             # specs/001-oauth-login.md
trilha-spec spec set 001-oauth-login --asset "session cookie" --control "ASVS V3.4" \
    --evidence "go test ./internal/auth/..."   # security impact, handed to the agent and the reviewer
trilha-spec spec move 001-oauth-login approved  # draft → approved → done; rejected, superseded
trilha-spec task add "Provider config" --spec 001-oauth-login --status ready \
    --accept "config loads" --check "go test ./internal/oauth/..."
trilha-spec task add "Login page" --depends TASK-001 --status ready --accept "login works" \
    --body "The page posts to /login; errors stay on the page."   # or --body-file PATH (- for stdin)
trilha-spec task next                          # what can run now
trilha-spec project limit max_cost_per_hour 5  # limits: in project.md; the runner enforces them
trilha-spec project pause --reason "breaker:max_cost_per_hour"   # next answers nothing and says why
trilha-spec context TASK-001                   # the pack an agent receives
trilha-spec task move TASK-001 running
trilha-spec task move TASK-001 verify
trilha-spec verify TASK-001                    # runs checks, records evidence, → review or failed
trilha-spec evidence TASK-001
trilha-spec keygen runner-01                   # private key in ~/.trilha/keys, public in .trilha/keys
trilha-spec evidence TASK-001 add --run --by runner-01 --model claude-sonnet-5 \
    --sign-key ~/.trilha/keys/runner-01.key    # Ed25519 signature on the record
trilha-spec evidence TASK-001 --verify         # unsigned | valid | invalid, per record
trilha-spec evidence TASK-001 add --run --provider anthropic --model claude-sonnet-5 \
    --tokens-in 12345 --tokens-out 678 --cost 0.0421 --currency USD   # what a runner declares
trilha-spec evidence TASK-001 add --eval --metric triage_top1 --value 0.87 \
    --threshold 0.85 --comparator '>=' --dataset golden-2026   # a number, not an exit code
trilha-spec evidence TASK-001 add --attestation --by "Ana Souza" --role uat \
    --statement "Homologado com a equipe da prefeitura." \
    --sign-key ~/.trilha/keys/ana.key          # `review: {quorum, roles}` on the task gates `done`
trilha-spec project milestone M2 --title "PoC" --due 2027-04-30 --gate "client sign-off"
trilha-spec task list --milestone M2           # next prefers the nearest deadline
trilha-spec spec show 001-oauth-login --coverage   # requirement → tasks → status → evidence
trilha-spec task next --repo trilha=../trilha  # answers `depends_on: [trilha:TASK-004]`
trilha-spec task graph                         # Mermaid; --dot for Graphviz, --program for the whole program
trilha-spec mcp --write                        # the same over MCP, for Claude Code / Cursor
```

Every listing takes `--json`. `trilha-spec doctor` says what a reader would trip on.
`TRILHA_LANG=pt` (or `pt-BR`) puts every message, `--help` included, in Portuguese, and makes
`init` and `spec new` write their templates in Portuguese; the file formats and `--json` do
not change, because status names, field names and IDs are the protocol.

### MCP

`trilha-spec mcp` serves the protocol over stdio to any MCP host. Read-only by default
(`trilha_list_tasks`, `trilha_get_task`, `trilha_next`, `trilha_context`, `trilha_list_specs`,
`trilha_list_evidence`, `trilha_coverage`, `trilha_graph`); `--write` adds `trilha_move`,
`trilha_spec_move`, `trilha_evidence`, `trilha_attest` and `trilha_verify`. A tool that is not
offered cannot be called.

```json
{ "mcpServers": { "trilha": { "command": "trilha-spec", "args": ["mcp", "--write"] } } }
```

## Packages

| Package | What it is |
|---|---|
| `spec` | front matter parser/writer, the `.trilha/` layout, specifications, `project.md` |
| `task` | the task model, statuses and transitions, the file store, the dependency graph, evidence and checks |
| `agent` | agent manifests: role, driver, tools allowed, constraints |
| `ai` | the context pack: everything an agent receives for one task, as Markdown or JSON |
| `mcp` | the protocol as MCP tools over stdio |
| `cmd/trilha-spec` | the CLI |

## Relationship with the `trilha` framework

The framework's CLI is `trilha`; this one is `trilha-spec`. They do not share a binary,
because `trilha` already owns `mcp`, `agents`, `ctx`, `check` and the `.trilha/` directory as
a build cache. [docs/adr/001](docs/adr/001-tres-repositorios.md) records the conflicts and the
two changes the framework needs so `trilha spec …` dispatches here, git-style.

Read the full protocol in [docs/protocol.md](docs/protocol.md); what it still lacks is in
[docs/roadmap.md](docs/roadmap.md).

## License

MIT.
