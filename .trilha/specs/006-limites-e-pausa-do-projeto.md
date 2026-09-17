---
id: 006-limites-e-pausa-do-projeto
title: Limites e pausa do projeto
status: done
issue: 3
assets:
  - project.md (front matter)
trust_boundaries:
  - control plane → runner
controls:
  - disjuntor de projeto (trilha-cloud spec 011)
evidence:
  - go test ./spec/... ./mcp/... ./cmd/...
---

# Limites e pausa do projeto

Spec curta em `specs/006-limites-e-pausa-do-projeto/spec.md`; escopo na issue #3.

## Acceptance

- **SC-001** `project limit max_cost_per_hour 5` grava o mapa `limits:` no project.md.
- **SC-002** `project pause --reason R` faz `task next` responder `project is paused: R`.
- **SC-003** `context TASK` mostra `### Limits`.
