---
id: TASK-005
title: F0.5 Evidence and verify
status: done
spec: 001-fundacao-fase-0
depends_on:
  - TASK-003
acceptance:
  - one JSON record per check with exit code and sha256
checks:
  - go test ./task/...
created: "2026-09-12T23:12:48Z"
updated: "2026-09-12T23:12:51Z"
---
