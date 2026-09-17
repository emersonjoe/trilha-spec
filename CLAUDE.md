# trilha-spec — the open protocol for work agents can execute

<!-- SPECKIT START -->
For additional context about technologies to be used, project structure,
shell commands, and other important information, read the current plan
<!-- SPECKIT END -->

## Commands

- `make test` — gofmt + `go vet ./...` + `go test ./...` (includes the CLI e2e).
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
- One spec per session; short spec (`.specify/templates/spec-curta-template.md`) for small changes.
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
