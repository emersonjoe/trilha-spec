---
name: trilha-spec
description: The open protocol for work software agents can execute
default_agent: coder
verify:
  - go vet ./...
  - go test ./...
---

# trilha-spec

The public, dependency-free half of Trilha's agentic development strategy: the format of
`.trilha/`, the task model and life cycle, the dependency graph, the context pack, evidence
and the MCP surface. Go, standard library only.

## Commands

- build: `go build ./cmd/trilha-spec`
- test: `make test`

## Where things are

- `spec/` documents and layout · `task/` model, store, graph, evidence · `agent/` manifests
- `ai/` context pack · `mcp/` stdio server · `cmd/trilha-spec/` CLI
- `docs/protocol.md` is normative; `docs/adr/` records decisions.
