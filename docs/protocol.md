# The Trilha protocol, version 0.2

> 🇺🇸 English · [🇧🇷 Português](pt-BR/protocol.md)

This is the normative description of what lives in `.trilha/`. The Go module in this
repository is the reference implementation; a tool in any language that follows this page
speaks the protocol.

## 1. Layout

```
.trilha/
├── .gitignore        runs/, keys/*.key and the framework's build cache; everything else is committed
├── project.md        Document: see §12
├── constitution.md   Markdown, free form
├── specs/NNN-name.md Document: see §11
├── tasks/TASK-NNN.md Document: see §3
├── agents/name.md    Document: name, role, driver, command, model, tools[], constraints[]
├── context/*.md      Markdown, free form; every file is handed to the agent
├── evidence/TASK-NNN/NNN-kind.json   see §5
├── keys/<key_id>.pub public Ed25519 keys, PEM; see §5 — a private key is never here
└── runs/             opaque to the protocol
```

The directory is found by walking up from the working directory, like `.git`.

## 2. Documents

A *Document* is UTF-8 Markdown that starts with a front matter block:

```
---
key: scalar
key: "quoted: scalar"
key: [a, b]
key:
  - item
  - item
key:
  sub: scalar
  sub: scalar
---

Body, free Markdown.
```

The grammar is deliberately a subset of YAML: scalars, lists of scalars, maps of scalars one
level deep (`key: {}` is an empty map), `#` comments and blank lines. Inside double quotes, `\"` and `\\` are the only escapes; single quotes carry text
as is. A key a reader does not know is preserved on write. Field order is stable: protocol
fields first, in the order below, then unknown fields in the order read.

## 3. Task

| Field | Required | Meaning |
|---|---|---|
| `id` | yes | `TASK-NNN`, NNN ≥ 3 digits, equal to the file name |
| `title` | yes | one line |
| `status` | yes | §4; missing means `idea` |
| `spec` | no | the specification ID it implements |
| `agent` | no | an agent name; `project.default_agent` otherwise |
| `depends_on` | no | task IDs; every one must exist; no cycles |
| `acceptance` | for `ready` on | what must be true to close, in words |
| `checks` | no | commands `verify` runs; program + arguments, no shell |
| `attempt`, `max_attempts`, `token_budget` | no | execution limits and current attempt metadata |
| `retry_of` | no | the preceding Run ID (`run-NNNNNN`) when this task is a repair |
| `failure_class`, `repair_reason` | no | stable failure category and human repair intent |
| `created`, `updated` | no | RFC 3339 UTC |

The body is the specification detail the agent reads.

## 4. Life cycle

```
idea → spec → ready → running → verify → review → done
```

| From | To |
|---|---|
| idea | spec, ready |
| spec | ready, idea |
| ready | running, blocked, spec |
| running | verify, failed, blocked, ready |
| verify | review, failed |
| review | done, ready, failed |
| done | ready (reopen) |
| blocked | ready |
| failed | ready |

Rules a writer enforces:

- `ready` and `running` require at least one acceptance criterion.
- `running` requires every dependency to be `done`.
- A task is **executable** when it is `ready` and every dependency is `done`. `next` lists
  executable tasks in dependency order, ties broken by ID.

## 5. Evidence

One JSON file per record, `evidence/TASK-NNN/NNN-kind.json`, NNN the sequence within the task:

```json
{
  "task": "TASK-001",
  "seq": 1,
  "kind": "check",
  "at": "2026-09-12T14:03:11Z",
  "by": "trilha-spec verify",
  "command": "go test ./...",
  "dir": "/work/app",
  "exit_code": 0,
  "output": "ok  \tapp\t0.4s\n",
  "output_sha256": "…",
  "passed": true
}
```

`kind` is `check` (a command ran), `note` (a person or agent wrote something), `artifact`
(files produced; `files[]`) or `run` (a runner's execution record; `meta{}` is free). `output`
may be truncated at 64 KiB; `output_sha256` hashes the whole of it. A record is never edited;
a correction is a new record.

A `run` record may carry its **cost** in standard fields, so ledgers from different runners
reconcile: `provider` (`anthropic`), `model` (`claude-sonnet-5`), `tokens_in`, `tokens_out`,
`cost` and `currency` (ISO 4217, required when `cost` is set). `cost` is what the runner
observed; the protocol does not claim it is verified — reconciling it against a provider's
invoice is a control plane's concern, and prices per model are not in the protocol. An
execution Attempt (§10) uses the same names.

A record may be **signed**, so a reviewer can tell what a runner attested from what anyone
could have typed. The signature is one field, left out when the record is unsigned:

```json
"signature": { "alg": "ed25519", "key_id": "runner-01", "sig": "<base64>" }
```

The signed bytes are the record's canonical form: the JSON object without `signature`, keys
sorted, no insignificant whitespace, no HTML escaping — the form any JSON library reproduces
from the file. `alg` is `ed25519` only; `key_id` is lowercase words joined by `-` or `.`; the
public key lives in `keys/<key_id>.pub` as PEM (PKIX) and is committed, while the private key
stays outside `.trilha` (`doctor` reports one inside). A reader checks every record and answers
one of three verdicts: `unsigned` (no signature — the record is a claim), `valid` (the
signature matches the record under the named key) or `invalid` (it does not, the key is
unknown, or the algorithm is not `ed25519`). Signing is opt-in: an unsigned record is still a
record. An edited record is `invalid`, which is the point.

`verify` runs the task's `checks` then `project.verify`, records one `check` per command, stops
for nothing, and answers *passed* only when every exit code is 0. A task with no checks at all
fails verification with a `note` saying so.

## 6. Context pack

What an agent receives for a task, in this order: project, constitution, its own manifest,
the specification, the task (body, acceptance, checks, dependencies with their status,
evidence so far), every file in `context/`. Markdown for a prompt, JSON for a tool. The pack
ends by telling the agent not to mark the task done: that is the reviewer's decision.

## 7. Agent manifest

`name` (lowercase words joined by `-`, equal to the file name), `role`, `driver` (`exec`, `ai`;
a runner may add others), `command`, `model`, `tools` ⊂ {read, write, run, git, network},
`constraints[]`. The protocol carries the manifest; enforcing it is the runner's job.

## 8. MCP

`trilha-spec mcp` serves JSON-RPC 2.0 over stdio, newline-delimited, MCP revision 2025-03-26,
capability `tools` only.

| Tool | Write? | Arguments |
|---|---|---|
| `trilha_list_tasks` | | `status?` |
| `trilha_get_task` | | `id` |
| `trilha_next` | | — ; a paused project (§12) answers an error with the reason |
| `trilha_context` | | `id`, `format?` (markdown \| json) |
| `trilha_list_specs` | | `status?` |
| `trilha_list_evidence` | | `id`; every record with its `verdict` (and `reason` when invalid) |
| `trilha_graph` | | |
| `trilha_move` | yes | `id`, `status` |
| `trilha_evidence` | yes | `id`, `kind` (note \| artifact), `note?`, `files?`, `by?` |
| `trilha_verify` | yes | `id`, `by?` |
| `trilha_spec_move` | yes | `id`, `status` (§11) |

Write tools are only listed with `--write`; a tool not listed cannot be called.

## 9. Versioning

This page is version 0.2. A change to a field, a transition or a file name bumps it and is
recorded in a spec under `specs/`. 0.2 (specs 003–008) added spec `rejected`/`superseded` and
relations, security impact on a spec, project limits and pause, cost fields on `run`
evidence, and signed evidence; every addition is a new optional field, so a 0.1 reader still
reads a 0.2 directory. Readers should tolerate unknown fields and unknown evidence
kinds.

## 10. Execution contract

Run transport between control planes and runners uses `trilha.execution/v1`. Its normative
JSON Schema is `contracts/execution/v1/schema.json`; `execution/testdata/run-v1.json` is the
canonical compatibility fixture. A Run is one logical execution of a Task and may contain
multiple Attempts. Repair lineage uses `retry_of` with a Run ID, never a Task ID. An Attempt
carries its cost under the names a `run` evidence record uses (§5): `provider`, `model`,
`tokens_in`, `tokens_out`, `cost`, `currency`.

## 11. Specification

| Field | Required | Meaning |
|---|---|---|
| `id` | yes | `NNN-name`, equal to the file name; the spec-kit branch numbering |
| `title` | yes | one line |
| `status` | yes | below; missing means `draft` |
| `issue` | no | the issue that is the source of the scope |
| `supersedes` | no | spec IDs this one replaces; every one must exist |
| `depends_on` | no | spec IDs this one builds on; every one must exist |
| `assets` | no | what the change touches, for the security reviewer: free identifiers |
| `trust_boundaries` | no | the boundaries it crosses (`browser → api`) |
| `controls` | no | the controls it affects (`ASVS V4.1`); free identifiers |
| `evidence` | no | commands a reviewer must see run: program and arguments, no shell, as task `checks` (§3) |

The four security fields are the **security impact** of the spec. The protocol carries them and
judges nothing about them: whether the controls are sufficient is the reviewer's call. The
context pack (§6) hands them to the agent next to the task's acceptance and checks; an
`approved` spec that declares none of them is a `doctor` warning, not a fault.

The body is the specification: why, what changes, out of scope, acceptance. A `draft` is being
written; `approved` is agreed and tasks may be cut from it; `done` has every task delivered;
`rejected` was judged and refused; `superseded` was replaced by a spec that names it in
`supersedes`.

| From | To |
|---|---|
| draft | approved, rejected, superseded |
| approved | done, rejected, superseded, draft |
| done | superseded |
| rejected | draft |
| superseded | — |

Rules a writer enforces: a spec never references itself; every reference exists (`doctor`
reports one that does not); a `superseded` spec is named in `supersedes` of at least one other
spec, or `doctor` reports it as an orphan. Task states never move a spec: that is a decision.

## 12. Project

`project.md` is the first thing an agent reads. Its body is prose; its front matter is what
tools consume.

| Field | Required | Meaning |
|---|---|---|
| `name` | yes | the project |
| `description` | no | one line |
| `default_agent` | no | the agent a task without `agent` gets |
| `verify` | no | commands every task runs on top of its own checks |
| `limits` | no | a map of numeric thresholds, below |
| `paused` | no | `true` stops the queue: `next` answers nothing and says why |
| `pause_reason` | no | free text; `breaker:<limit>` when a control plane tripped on a limit |
| `paused_at` | no | RFC 3339 UTC; required when `paused` |

`limits` is the project's envelope. The protocol names three keys and carries any other:

| Key | Meaning |
|---|---|
| `max_cost_per_hour` | in the currency of the evidence (§5) |
| `max_failure_rate` | a share, 0..1, over the last runs |
| `max_repeated_failure_class` | the same `failure_class` this many times in a row |

The protocol *carries* limits and pause; enforcing them — refusing to start a task, stopping a
running attempt, tripping the breaker — is the runner's and the control plane's job, as with
agent manifests (§7). The context pack (§6) includes `limits` and the pause, so an agent knows
its envelope. Every reader may ignore all of these fields.
