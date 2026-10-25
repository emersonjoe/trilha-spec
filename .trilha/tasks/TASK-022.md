---
id: TASK-022
title: "RM-004: ADR 003, fonte da verdade e estado honesto"
status: done
spec: 015-fase-0-do-roadmap
agent: coder
depends_on: []
acceptance:
  - docs/adr/003-fonte-da-verdade.md diz que .trilha/ é autoritativo nos três repositórios do protocolo e que specs/NNN/ é histórico congelado
  - CLAUDE.md sem o bloco SPECKIT; roadmap e ADR 001 sem texto obsoleto; cinco ideias do roadmap como tasks idea
checks:
  - make test
created: "2026-10-21T02:10:09Z"
updated: "2026-10-21T02:19:28Z"
---

Roadmap RM-004. As pastas specs/NNN/ existentes não são apagadas: são o registro histórico das specs 001-014.
