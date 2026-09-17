---
id: 003-corpo-e-templates-da-cli
title: Corpo e templates da CLI
status: done
issue: 5
---

# Corpo da task, templates pt-BR e `--status` documentado

Spec curta em `specs/003-corpo-e-templates-da-cli/spec.md`; escopo na issue #5.

## Acceptance

- **SC-001** `task add X --body T` grava `T` como corpo; `task show X --json` expõe `body`.
- **SC-002** `TRILHA_LANG=pt-BR trilha-spec init` escreve `# Constituição`.
