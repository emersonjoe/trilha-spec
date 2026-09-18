---
id: 011-evidencia-eval
title: "Evidência eval: métricas com limiar"
status: done
issue: 8
assets:
  - registros de evidência (.trilha/evidence)
  - saída dos checks lida pelo verify
  - manifesto de golden set referenciado em dataset{id, sha256}
trust_boundaries:
  - harness → repositório
  - repositório → revisor/control plane
controls:
  - "dataset guarda hash do manifesto, nunca o conteúdo (golden set tem dado pessoal)"
  - comparador obrigatório com limiar; portão nunca é adivinhado
  - "ASVS V5.1 (nome de métrica e comparador validados)"
evidence:
  - go test ./task/... -run TestEval
  - go test ./task/... -run TestVerifyRecordsMetrics
  - go test ./cmd/... -run TestEvalEvidenceCLI
---

Ver `specs/011-evidencia-eval/spec.md` (issue #8). Novo `kind: "eval"` com
`metric`, `value`, `unit`, `threshold`, `comparator`, `dataset{id, sha256}` e `passed`; o
`verify` grava um `eval` por linha JSON que um check imprime, e um `eval` reprovado reprova a
verificação mesmo com saída 0. `acceptance` aceita `metric: <nome> <comparador> <número>`;
`doctor` aponta métrica sem evidência; registros e `--json` deixam de escapar HTML.
