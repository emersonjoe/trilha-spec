# Spec 012 — Atestação humana e quórum de revisores

- **Issue**: [#9](https://github.com/emersonjoe/trilha-spec/issues/9) — a issue é a fonte do escopo.
- **Task**: `TASK-015` em `.trilha/tasks/`.
- **Versão**: 0.3 do protocolo (§3, §4, §5, §6 e §8); tipo novo de evidência e campo aditivo.
- **Depende de**: spec 008 (evidência assinada) e spec 009 (bloco com lista no front matter).

## Por quê

`review → done` era uma decisão de uma pessoa só, e evidência era ou de máquina ou uma `note`
livre. Entrega no setor público tem decisões que o protocolo não sabia expressar: um aceite de
UAT por um usuário nomeado, uma homologação jurídica ou de compras, uma tradução validada por
falante nativo, um comitê que exige N aprovações. Nada distinguia "alguém digitou uma nota" de
"esta pessoa, neste papel, atesta isto — e assinou".

Contexto: um programa de setor público em que todo PoC de produto termina com aceite humano da
equipe do cliente (detalhe na issue).

## O que muda

- `kind: "attestation"`: `by`, `role`, `statement`, `refs[]` (sequências de evidência ou
  caminhos de artefato que a declaração cobre) e `signature` opcional, reusando o esquema
  Ed25519 e o `keys/` — **a chave de uma pessoa é uma chave como a de um runner**.
- `role` são palavras minúsculas unidas por `-`, a mesma forma de um nome de agente.
- Front matter da task: `review: {quorum: N, roles: [uat, legal]}`.
- `task move <id> done` é recusado até existirem N atestações **assinadas**, válidas sob uma
  chave de `keys/`, em um dos `roles` e por **chaves distintas**. A recusa diz o que falta e
  por que cada registro recusado não contou: sem assinatura, chave desconhecida, papel fora da
  política, ou a mesma pessoa de novo.
- Atestação sem assinatura continua sendo protocolo válido — é uma alegação, e alegação não faz
  quórum.
- MCP `trilha_attest` (modo escrita); grava sem assinatura, e o próprio texto da ferramenta diz
  que por isso não conta para o quórum (este servidor não tem chave privada).
- Pacote de contexto: quantas atestações ainda faltam, quais já entraram e quais não contaram.
- `doctor`: `quorum-without-roles` (quórum sem papéis — qualquer papel o satisfaria) e
  `attestation-unknown-key` (assinada com chave que o projeto não tem).
- CLI: `evidence <id> add --attestation --by NOME --role R --statement S [--ref X]...`, com
  `--sign-key` como qualquer registro.
- O protocolo carrega; runner e control plane fazem valer — a mesma divisão de limites e
  manifestos.

```bash
trilha-spec keygen ana
trilha-spec evidence TASK-001 add --attestation --by "Ana Souza" --role uat \
  --statement "Homologado em 2027-04-28 com a equipe da prefeitura." --ref '#4' \
  --sign-key ~/.trilha/keys/ana.key
trilha-spec task move TASK-001 done
```

## Fora de escopo

- Verificar a identidade de quem atesta além da chave (isso é o login do control plane).
- O fluxo de quem pede a quem, prazos de aprovação, lembretes.

## Constitution Check

| Princípio | Como respeita |
|---|---|
| I — o protocolo é o produto | §3, §4 e §5 descrevem o registro e a regra da transição antes do código |
| II — só biblioteca padrão | reusa `crypto/ed25519` já presente |
| III — evidência verificável | o que fecha a task são registros assinados, não uma chamada de API |
| V — teste primeiro | `TestQuorumClosesTheTask` (um, o mesmo duas vezes, dois distintos), e2e |

## Tarefas

- [x] T001 `kind: attestation` com `role`, `statement`, `refs[]` e validação
- [x] T002 `Task.Review`, round trip do bloco `review:`, `task.QuorumOf` e a recusa em `Store.Move`
- [x] T003 `task.CheckAttestations`, códigos de problema e i18n
- [x] T004 CLI `evidence add --attestation`, MCP `trilha_attest`, pacote de contexto
- [x] T005 Protocolo §3/§4/§5/§6/§8 (en, pt-BR)

## Aceitação

- **SC-001** Com `quorum: 2`, uma atestação assinada não fecha a task e a recusa diz quantas faltam.
- **SC-002** Duas atestações da mesma chave não fecham a task.
- **SC-003** Duas atestações de chaves distintas, nos papéis pedidos, fecham a task.
- **SC-004** Atestação sem assinatura não conta, e a CLI avisa disso ao gravá-la.
- **SC-005** `doctor` acusa quórum sem papéis e atestação com chave desconhecida.
