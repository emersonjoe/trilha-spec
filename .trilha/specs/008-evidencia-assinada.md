---
id: 008-evidencia-assinada
title: Evidência assinada
status: done
issue: 7
assets:
  - registros de evidência (.trilha/evidence)
  - chaves públicas (.trilha/keys)
trust_boundaries:
  - runner → repositório
  - repositório → revisor/control plane
controls:
  - "ASVS V6.2 (algoritmos aprovados: Ed25519)"
  - ASVS V6.4 (chave privada fora do repositório)
evidence:
  - go test ./task/... -run TestSignAndVerifyEvidence
  - go test ./cmd/... -run TestSignedEvidenceCLI
---

Ver `specs/008-evidencia-assinada/spec.md` (issue #7). Evidência ganha `signature{alg, key_id, sig}` em Ed25519 sobre a forma canônica do registro; chaves públicas em `.trilha/keys/`; vereditos `unsigned | valid | invalid` na CLI (`evidence --verify`), no MCP (`trilha_list_evidence`) e no pacote de contexto.
