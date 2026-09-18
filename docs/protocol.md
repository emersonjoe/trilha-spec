# The Trilha protocol, version 0.3

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
  sub: [a, b]
key:
  - sub: scalar
    sub: scalar
  - sub: scalar
---

Body, free Markdown.
```

The grammar is deliberately a subset of YAML: scalars, lists of scalars, a **block** of
scalars and lists under one key (`key: {}` is an empty block), a **list of blocks**, `#`
comments and blank lines. Nesting stops there: a block holds scalars and lists, never another
block. A `- ` item that reads as an unquoted `sub: value` opens a block; a scalar item holding
a colon is quoted, which is how the writer emits it, and one list never mixes the two. Inside
double quotes, `\"` and `\\` are the only escapes; single quotes carry text
as is. A key a reader does not know is preserved on write. Field order is stable: protocol
fields first, in the order below, then unknown fields in the order read.

## 3. Task

| Field | Required | Meaning |
|---|---|---|
| `id` | yes | `TASK-NNN`, NNN ≥ 3 digits, equal to the file name |
| `title` | yes | one line |
| `status` | yes | §4; missing means `idea` |
| `spec` | no | the specification ID it implements |
| `milestone` | no | the milestone (§12) it belongs to; empty means its spec's |
| `agent` | no | an agent name; `project.default_agent` otherwise |
| `depends_on` | no | task IDs, or `<alias>:TASK-NNN` for another repository (§13); a local one must exist; no cycles |
| `covers` | no | requirement IDs the task delivers (§11); every one must be declared by a spec |
| `acceptance` | for `ready` on | what must be true to close, in words; an item may be a metric gate, `metric: <name> <comparator> <number>` (§5) |
| `checks` | no | commands `verify` runs; program + arguments, no shell |
| `review` | no | the human quorum needed to close: `{quorum: N, roles: [uat, legal]}` (§5) |
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
- `done` requires the task's `review` quorum (§3), when it declares one: N distinct signed
  attestations (§5) whose `role` is in `roles`. The refusal names what is missing.
- A task is **executable** when it is `ready` and every dependency is `done`. `next` lists
  executable tasks in dependency order, ties broken by ID, and among those the nearest
  milestone due date (§12) first; a task with no due date comes last.

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
(files produced; `files[]`), `run` (a runner's execution record; `meta{}` is free), `eval`
(a number a harness measured) or `attestation` (a named person's decision) — the last two
below. `output` may be truncated at 64 KiB; `output_sha256`
hashes the whole of it. A record is never edited; a correction is a new record. A record is
written without HTML escaping, so a comparator reads as `>=`.

An **`eval`** record carries a measurement and the bar it had to clear, so the value is
visible, can be trended across runs and can gate a release — a `check` hides all three in an
exit code:

```json
{
  "kind": "eval",
  "metric": "triage_top1",
  "value": 0.87,
  "unit": "",
  "threshold": 0.85,
  "comparator": ">=",
  "dataset": { "id": "golden-2026", "sha256": "…" },
  "passed": true
}
```

`metric` is lowercase words joined by `_`, `-` or `.`, so two harnesses that measure the same
thing agree on the name. `comparator` is `>=`, `<=` or `==` and is **required** when
`threshold` is set: a guessed comparator is a wrong gate. A record without a threshold is a
measurement, not a gate, and passes. `value` and `threshold` are written even when they are
zero, because zero is a number a gate cares about. `dataset` names the set the metric was
measured on and hashes its **manifest**, never its content: a golden set holds real cases, and
real cases hold personal data. An `eval` is signed like any other record.

`verify` turns every line a check prints on stdout that is a JSON object with a `metric` and a
`value` into one `eval` record, in that small shape — so a harness in any language emits them
with one `echo` — and a check that prints plain text is unaffected. A failing `eval` fails the
verification even when the command exited 0: that is what a gate is for. A line whose
`threshold` comes without a comparator is recorded as a measurement, with the reason in `note`
and `passed: false`, rather than gated on a guess.

An **`attestation`** record is a decision by a named person, not a machine: a UAT sign-off, a
legal or procurement homologation, a translation validated by a native speaker.

```json
{
  "kind": "attestation",
  "by": "Ana Souza",
  "role": "uat",
  "statement": "Homologado em 2027-04-28 com a equipe da prefeitura.",
  "refs": ["#4", "docs/ata.pdf"],
  "signature": { "alg": "ed25519", "key_id": "ana", "sig": "…" }
}
```

`role` is lowercase words joined by `-`; `refs` names the evidence sequences or artifact paths
the statement covers. A person's key is a key like a runner's: it lives in `keys/<key_id>.pub`
and signs the record the same way. An **unsigned** attestation is a claim — valid protocol,
but it does not make quorum, because anyone could have typed it.

A task declares what it needs in `review`:

```
review:
  quorum: 2
  roles: [uat, legal]
```

`review → done` is refused until there are `quorum` attestations that are signed, valid under
a key in `keys/`, in one of `roles`, and by **distinct keys** — the same person twice is still
one person. The refusal says which of those each rejected record failed. `doctor` reports a
quorum declared with no roles (any role would satisfy it) and an attestation signed with a key
the project does not hold. The context pack (§6) shows what is still owed. A task with no
`review` closes exactly as before.

A task may declare a gate as an acceptance criterion, `metric: triage_top1 >= 0.85`; it is
still a criterion in words, and `doctor` reports one that no `eval` answers once the task
reaches `review`. The context pack (§6) carries the last value of every metric.

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
the specification, the task (body, its milestone with the due date, acceptance, the
requirements it covers with their text, checks, dependencies with their status, evidence so
far, the last value of each metric and the attestations still owed), every file in `context/`. Markdown for a prompt, JSON for a tool. The pack
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
| `trilha_coverage` | | `spec?`; the requirement matrix of §11 |
| `trilha_graph` | | |
| `trilha_move` | yes | `id`, `status` |
| `trilha_evidence` | yes | `id`, `kind` (note \| artifact), `note?`, `files?`, `by?` |
| `trilha_attest` | yes | `id`, `by`, `role`, `statement`, `refs?`; written unsigned |
| `trilha_verify` | yes | `id`, `by?` |
| `trilha_spec_move` | yes | `id`, `status` (§11) |

Write tools are only listed with `--write`; a tool not listed cannot be called.

## 9. Versioning

This page is version 0.3. A change to a field, a transition or a file name bumps it and is
recorded in a spec under `specs/`.

0.3 (specs 009–013) came from planning a public-sector programme across four repositories. It
added requirement traceability (`requirements` on a spec, `covers` on a task), milestones with
due dates, the `eval` evidence kind for metrics with thresholds, the `attestation` kind with a
review quorum on a task, and cross-repository dependencies with an optional `program.md`. The
front matter grammar (§2) grew two shapes for them: a block may hold lists, and a list may hold
blocks. Every addition is a new optional field, so a 0.2 reader still reads a 0.3 directory —
with one new rule a *writer* must know: `review → done` is refused while a declared quorum is
unmet (§4).

0.2 (specs 003–008) added spec `rejected`/`superseded` and
relations, security impact on a spec, project limits and pause, cost fields on `run`
evidence, and signed evidence. Readers should tolerate unknown fields and unknown evidence
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
| `milestone` | no | the milestone (§12) the whole spec belongs to |
| `supersedes` | no | spec IDs this one replaces; every one must exist |
| `depends_on` | no | spec IDs this one builds on; every one must exist |
| `assets` | no | what the change touches, for the security reviewer: free identifiers |
| `trust_boundaries` | no | the boundaries it crosses (`browser → api`) |
| `controls` | no | the controls it affects (`ASVS V4.1`); free identifiers |
| `evidence` | no | commands a reviewer must see run: program and arguments, no shell, as task `checks` (§3) |
| `requirements` | no | external requirements this spec answers, one block each: `id`, `source`, `text` |

The four security fields are the **security impact** of the spec. The protocol carries them and
judges nothing about them: whether the controls are sufficient is the reviewer's call. The
context pack (§6) hands them to the agent next to the task's acceptance and checks; an
`approved` spec that declares none of them is a `doctor` warning, not a fault.

### Requirements

When the source of scope is a document outside the repository — a public tender's requirement
list, an article of a law, a KPI in a contract — the spec carries the reference and the tasks
point back at it:

```
requirements:
  - id: D2-R8
    source: "cp-01-2026#anexo-I"
    text: informar o cidadao sobre o andamento
```

An `id` is a free identifier without a space or a comma, so `covers: [D2-R8, D2-R9]` never
splits one; it is unique across the project, because `covers` names it and nothing else. The
protocol judges nothing about `source` and `text`: it carries them, hands them to the agent in
the context pack (§6) and answers the matrix — requirement → tasks → status → evidence count —
to `spec show --coverage` and to `trilha_coverage` (§8). `doctor` reports a requirement no task
covers (a warning: the scope is declared, the work is not cut yet), a task covering an id no
spec declares, and the same id declared by two specs.

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
| `repos` | no | other repositories this one depends on, alias → URL; see §13 |
| `milestones` | no | the dated points of the programme, one block each: `id`, `title`, `due`, `gate` |
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

### Milestones

A contract with paid milestones, or a programme reported against a calendar, declares them
once in `project.md`:

```
milestones:
  - id: M2
    title: PoC entregue
    due: 2027-04-30
    gate: aceite do cliente
```

An `id` follows the rule of a requirement id (§11); `due` is a calendar day, `YYYY-MM-DD`, not
an instant. A spec and a task name one in `milestone`, and a task without one of its own takes
its spec's, the way a task without an `agent` takes `default_agent`. `task list --milestone M2`
narrows a listing; `next` prefers the nearest due date among what can run; the context pack
(§6) hands the agent the milestone and its date. `doctor` reports a milestone no task belongs
to and an unfinished task whose milestone is past due — both warnings, because the protocol
reports the calendar and the people keep it — and, as a fault, a task or spec naming a
milestone `project.md` does not declare. Estimates, capacity and what a milestone is worth are
a control plane's business.

The protocol *carries* limits, milestones and pause; enforcing them — refusing to start a task, stopping a
running attempt, tripping the breaker — is the runner's and the control plane's job, as with
agent manifests (§7). The context pack (§6) includes `limits` and the pause, so an agent knows
its envelope. Every reader may ignore all of these fields.

## 13. Programs across repositories

A programme spans repositories — the product, the framework it runs on, the control plane, a
gateway — and their work depends on each other. `project.md` names the ones this repository
points at:

```
repos:
  trilha: https://github.com/emersonjoe/trilha
```

A task then depends on a task over there, `depends_on: [trilha:TASK-004]`. An alias is
lowercase words joined by `-` or `.`; the URL says which repository it is, and `doctor` reports
a dependency on an alias `repos` does not declare.

Resolving it is somebody's job, not the format's. A reader answers a remote dependency from a
**sibling checkout** (`--repo alias=path`, repeatable) or from a control plane that answers for
it — the reference implementation states that as an interface and opens no socket. While
nobody answers, the dependency blocks the task with the reason `waiting:<alias>:TASK-NNN`:
`next` does not offer it, `running` refuses it and `graph` draws it. A remote dependency never
joins this repository's topological order, because it is not this repository's work.

An optional `program.md`, in a directory above the checkouts, names the repositories and the
milestones they share:

```
---
name: Platform programme
repos:
  app: app
  trilha: trilha
milestones:
  - id: M2
    due: 2027-04-30
---
```

Its `repos` are paths, relative to the file or absolute, so a reader that finds the manifest
resolves the aliases without being told where anything is; `--repo` still wins over it. It is
found by walking up from the project's parent, it is not part of `.trilha/`, and its
milestones are the shape of §12. `graph --program` draws one subgraph per repository, with the
dependencies that cross between them; a repository nobody checked out still appears, as the
thing the work is waiting for. A project with no manifest and no `repos` never notices any of
this.
