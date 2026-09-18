---
id: TASK-014
title: "Evidence kind eval: metric records with thresholds"
status: done
spec: 011-evidencia-eval
depends_on: []
acceptance:
  - a harness prints a number and the protocol records it with the threshold it had to beat
  - a metric below its threshold fails verification even when the command exits 0
checks: []
created: "2026-09-18T16:51:19Z"
updated: "2026-09-18T18:35:10Z"
---

Issue: https://github.com/emersonjoe/trilha-spec/issues/8 — the issue is the source of the scope.

Origin: gap found while planning a delivery program across four repositories — the
product app, the framework, the runner and the control plane — where the protocol has to
track the work with quality gates the buyer can audit.
