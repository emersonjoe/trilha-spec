---
id: TASK-029
title: Contrato 0.4 com status needs_input, estado do agente, memória, achados e artefatos
status: review
spec: 016-protocolo-0-4-conversa-com-o-agente-memoria-da-task-e-achados
agent: coder
depends_on: []
acceptance:
  - execution.Run e Outcome carregam os campos 0.4 e Validate recusa valores fora dos limites
  - a fixture run-v1-needs-input.json valida e o script de conformidade a compara
  - protocolo §9 e §10 documentam a 0.4 nos dois idiomas
checks:
  - go test ./...
  - gofmt -l .
created: "2026-10-25T23:50:04Z"
updated: "2026-10-25T23:50:15Z"
---
