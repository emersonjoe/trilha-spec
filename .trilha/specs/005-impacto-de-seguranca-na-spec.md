---
id: 005-impacto-de-seguranca-na-spec
title: Impacto de segurança na spec
status: done
issue: 1
assets:
  - specs/NNN-nome.md (front matter)
controls:
  - "constituição do trilha-cloud: impacto de segurança em toda spec"
evidence:
  - go test ./spec/... ./ai/... ./cmd/...
---

# Impacto de segurança e comandos de evidência na spec

Spec curta em `specs/005-impacto-de-seguranca-na-spec/spec.md`; escopo na issue #1.

## Acceptance

- **SC-001** `spec set X --evidence CMD` grava `evidence:` e `spec show X --json` expõe.
- **SC-002** `doctor` avisa spec `approved` sem impacto e sai com 0.
- **SC-003** `context TASK` mostra os quatro campos ao lado do aceite.
