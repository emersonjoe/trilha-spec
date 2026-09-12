---
id: TASK-006
title: F0.6 MCP server
status: done
spec: 001-fundacao-fase-0
depends_on:
  - TASK-005
acceptance:
  - initialize, tools/list, tools/call over stdio; write tools only with --write
checks:
  - go test ./mcp/...
created: "2026-09-12T23:12:48Z"
updated: "2026-09-12T23:12:51Z"
---
