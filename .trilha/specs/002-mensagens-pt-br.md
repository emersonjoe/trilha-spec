---
id: 002-mensagens-pt-br
title: Mensagens pt-BR
status: done
issue: "6"
---

# Mensagens da CLI em pt-BR via `TRILHA_LANG`

Spec curta em `specs/002-mensagens-pt-br/spec.md`; escopo na issue #6.

## Acceptance

- **SC-001** `TRILHA_LANG=pt-BR trilha-spec --help` sai em português; língua desconhecida cai para inglês.
- **SC-002** `--json` e os arquivos em `.trilha/` não mudam com a variável.
