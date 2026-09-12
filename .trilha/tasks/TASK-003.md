---
id: TASK-003
title: F0.3 Life cycle and dependency graph
status: done
spec: 001-fundacao-fase-0
depends_on:
  - TASK-002
acceptance:
  - illegal transitions refused; running needs deps done; cycles named
checks:
  - go test ./task/...
created: "2026-09-12T23:12:48Z"
updated: "2026-09-12T23:12:50Z"
---
