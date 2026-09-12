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
