---
id: 007-custo-na-evidencia-run
title: Custo na evidência run
status: done
issue: 4
assets:
  - evidence/TASK-NNN/NNN-run.json
  - contracts/execution/v1/schema.json
trust_boundaries:
  - runner → control plane (ledger)
controls:
  - custo auto-declarado, não verificado pelo protocolo
evidence:
  - go test ./task/... ./execution/... ./cmd/...
---

# Custo padronizado na evidência `run`

Spec curta em `specs/007-custo-na-evidencia-run/spec.md`; escopo na issue #4.

## Acceptance

- **SC-001** `evidence add --run --cost 1` sem `--currency` falha.
- **SC-002** `evidence <task> --json` expõe os seis campos de custo.
- **SC-003** O fixture `run-v1.json` valida com os campos na Attempt.
