---
id: 004-ciclo-de-vida-de-spec
title: Ciclo de vida de spec
status: done
issue: 2
---

# Ciclo de vida de spec: `rejected`, `superseded`, relações e `spec move`

Spec curta em `specs/004-ciclo-de-vida-de-spec/spec.md`; escopo na issue #2.

## Acceptance

- **SC-001** `spec move X done` a partir de `draft` falha.
- **SC-002** `spec set X --supersedes Y` com `Y` inexistente falha e não grava.
- **SC-003** `doctor` aponta uma spec `superseded` que ninguém cita.
