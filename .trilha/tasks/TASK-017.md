---
id: TASK-017
title: Cross-repository dependencies and a program manifest
status: done
spec: 013-dependencias-entre-repositorios
depends_on:
  - TASK-018
acceptance:
  - a task depends on a task in another repository and waits until somebody answers for it
  - a program manifest above the checkouts resolves the aliases and draws the whole graph
checks: []
created: "2026-09-18T16:51:19Z"
updated: "2026-09-18T18:50:48Z"
---

Issue: https://github.com/emersonjoe/trilha-spec/issues/11 — the issue is the source of the scope.

Origin: gap found while planning a delivery program across four repositories — the
product app, the framework, the runner and the control plane — where the protocol has to
track the work with quality gates the buyer can audit.
