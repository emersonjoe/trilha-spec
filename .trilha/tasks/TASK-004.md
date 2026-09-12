---
id: TASK-004
title: F0.4 CLI
status: done
spec: 001-fundacao-fase-0
depends_on:
  - TASK-003
acceptance:
  - init, spec, task, agent, context, verify, evidence, mcp, doctor work end to end
checks:
  - go test ./cmd/...
created: "2026-09-12T23:12:48Z"
updated: "2026-09-12T23:12:50Z"
---
