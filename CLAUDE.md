# trilha-spec — the open protocol for work agents can execute

Specs, tasks and evidence live in `.trilha/` and nowhere else; `specs/NNN/` folders are frozen history.
Why, and what `.specify/` still holds: `docs/adr/003-fonte-da-verdade.md`.

## Commands

- `make test` — gofmt + `go vet ./...` + `go test ./...` (includes the CLI e2e).
- `make conformance` — compares the execution contract with the sibling checkouts (`../trilha-runner`, `../trilha-cloud`); absent siblings are skipped.
- `make build` — `bin/trilha-spec`.
- `trilha-spec` on this repository itself: `.trilha/` holds the tasks of the current phase
  (`go run ./cmd/trilha-spec task next`).

## Structure

- `spec/` front matter, layout, specifications, project. `task/` model, store, graph, evidence.
  `agent/` manifests. `ai/` context pack. `mcp/` stdio server + tools. `cmd/trilha-spec/` CLI.
- `docs/protocol.md` is the normative description; code follows it, not the other way round.

## Rules (constitution in `.specify/memory/constitution.md`)

- Zero dependencies outside the standard library. The protocol must be readable without this module.
- Every file under `.trilha/` is Markdown + front matter or JSON; no other formats.
- Status transitions are the ones in `task.Transitions`; a new one is a spec, not a patch.
- Public text (README, docs, CLI) in English with pt-BR in the same commit; specs and ADRs in pt-BR.
- One spec per session, written with `trilha-spec spec new` into `.trilha/specs/` (adr/003); the
  template in `.specify/templates/spec-curta-template.md` is the outline for a short one.
- **The hands-on chapter follows the code.** The Trilha site teaches this tool in
  `https://emersonjoe.github.io/trilha/learn/agentic-protocol` (pt: `/pt/aprender/agentico-protocolo`). A change that alters what a user types or
  sees — a command, a flag, a file format, an output — updates that chapter in the same session:
  `site/internal/docs/content/en/learn/agentic-protocol.md` and `site/internal/docs/content/pt/aprender/agentico-protocolo.md` in the `emersonjoe/trilha` repository, both locales, before the spec closes.

## Agent skills

### Issue tracker

Work is tracked in this repository's GitHub Issues. See `docs/agents/issue-tracker.md`.

### Triage labels

Use the canonical five-label triage vocabulary. See `docs/agents/triage-labels.md`.

### Domain docs

Use the single-context domain documentation layout. See `docs/agents/domain.md`.
