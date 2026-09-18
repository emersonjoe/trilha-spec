---
id: 012-atestacao-humana
title: Atestação humana e quórum de revisores
status: done
issue: 9
depends_on:
  - 008-evidencia-assinada
  - 009-rastreabilidade-de-requisitos
assets:
  - registros de evidência (.trilha/evidence)
  - chaves públicas de pessoas (.trilha/keys)
  - a transição review → done
trust_boundaries:
  - pessoa → repositório
  - repositório → comprador/auditor
controls:
  - "ASVS V6.2 (Ed25519 reusado para a chave de uma pessoa)"
  - só atestação assinada e de chave distinta conta para o quórum
  - "identidade além da chave é do control plane, não do protocolo"
evidence:
  - go test ./task/... -run TestQuorum
  - go test ./task/... -run TestAttestation
  - go test ./cmd/... -run TestAttestationQuorumCLI
---

Ver `specs/012-atestacao-humana/spec.md` (issue #9). Novo `kind: "attestation"` com `role`,
`statement` e `refs[]`, assinável com o esquema Ed25519 já existente; `review: {quorum, roles}`
na task, e `review → done` recusado até haver N atestações assinadas, de chaves distintas, nos
papéis pedidos. MCP `trilha_attest`, quórum no pacote de contexto e duas checagens novas no
`doctor`.
