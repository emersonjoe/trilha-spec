---
id: 009-rastreabilidade-de-requisitos
title: Rastreabilidade de requisitos
status: done
issue: 10
assets:
  - front matter de spec e de task (.trilha/specs, .trilha/tasks)
  - matriz de cobertura exposta por CLI e MCP
trust_boundaries:
  - repositório → comprador/auditor
controls:
  - texto de requisito é dado do edital, não segredo
  - "ASVS V5.1 (ids validados: sem espaço nem vírgula)"
evidence:
  - go test ./spec/... -run TestBlockLists
  - go test ./task/... -run TestCoverageMatrix
  - go test ./cmd/... -run TestRequirementCoverageCLI
---

Ver `specs/009-rastreabilidade-de-requisitos/spec.md` (issue #10). A spec ganha
`requirements: [{id, source, text}]` e a task ganha `covers: [...]`; a gramática do front
matter ganha bloco com lista e lista de blocos. `spec show --coverage`, `trilha_coverage` e o
pacote de contexto respondem a matriz requisito → tasks → status → evidência; `doctor` acusa
requisito sem task (aviso), `covers` para id inexistente e id declarado por duas specs.
