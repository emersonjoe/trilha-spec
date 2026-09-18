---
id: TASK-016
title: "Requirement traceability: external requirement references"
status: done
spec: 009-rastreabilidade-de-requisitos
depends_on: []
acceptance:
  - a spec carries the requirements of an external document and a task says which it covers
  - `spec show --coverage`, `trilha_coverage` and doctor answer what is covered and what is not
checks: []
created: "2026-09-18T16:51:19Z"
updated: "2026-09-18T18:16:15Z"
---

Issue: https://github.com/emersonjoe/trilha-spec/issues/10 — the issue is the source of the scope.

Origin: gap found while planning a delivery program across four repositories — the
product app, the framework, the runner and the control plane — where the protocol has to
track the work with quality gates the buyer can audit.
