---
id: TASK-020
title: "RM-007: gofmt completo no CI e job conformance contra o runner"
status: ready
spec: 015-fase-0-do-roadmap
agent: coder
depends_on: []
acceptance:
  - o CI roda gofmt -l . e um job conformance faz checkout do trilha-runner e roda scripts/check-execution-contract.sh
  - "o script diz 'skipped: <repo> not present' para irmão ausente e falha no primeiro fixture divergente"
checks:
  - make test
created: "2026-10-21T02:10:09Z"
---

Roadmap RM-007 e a parte do trilha-spec de RM-003 (trilha-cloud/docs/ROADMAP-STACK.md). O trilha-cloud é privado e fica fora do job de CI; roda em make conformance local.
