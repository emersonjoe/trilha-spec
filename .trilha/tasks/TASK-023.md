---
id: TASK-023
title: doctor avisa spec approved com todas as tasks done
status: done
spec: 015-fase-0-do-roadmap
agent: coder
depends_on: []
acceptance:
  - "spec approved cujas tasks estão todas done gera o aviso 'spec NNN: all tasks done, status should be done' e doctor continua saindo 0"
checks:
  - go test ./task -run TestCheckSpecStatus
  - make test
created: "2026-10-21T02:10:09Z"
updated: "2026-10-21T02:19:28Z"
---

Teste de unidade em spec/ e verificação pela CLI (EN e pt-BR). Aviso, não falha: a decisão de fechar a spec é humana.
