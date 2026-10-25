---
id: 016-protocolo-0-4-conversa-com-o-agente-memoria-da-task-e-achados
title: "Protocolo 0.4: conversa com o agente, memória da task e achados"
status: approved
assets:
  - contrato de fio runner↔cloud
  - texto do agente que atravessa o fio
trust_boundaries:
  - runner → cloud
  - cloud → prompt do agente
controls:
  - ASVS V5 validação de entrada com limites
  - defesa contra injeção de prompt por citação
evidence:
  - make test
  - make conformance
---

# Protocolo 0.4: conversa com o agente, memória da task e achados

Item DV-00 de `trilha-cloud/docs/PLANO-DEV3.md`, que traz para a stack as melhores ideias do
`h0x91b/dev-3.0` adaptadas a um control plane remoto que nunca vê código.

## Por quê

O runner e o cloud precisam de um vocabulário comum para: o que o agente está fazendo, quando
ele precisa de alguém, a pergunta que ele faz e a resposta que o continua, a memória da task
entre runs, as contagens do que mudou, os runs de revisão e de caça a bugs com achados
ancorados, os comentários de revisão humanos e os artefatos de relatório. Sem o contrato
primeiro, cada lado inventaria campos diferentes.

## O que muda

- `execution/agent.go`: tipos `AgentState`, `AgentEvent`, `NextAgentState`, `Attention`,
  `Progress`, `Question`, `Session`, `Continuation`, `Note`, `TaskContext`, `DiffStats`,
  `Finding`, `ReviewVerdict`, `ReviewMode`, `HuntSeed`, `ReviewComment`, `ReviewReply`,
  `Artifact`, os limites exportados e `QuoteContext`.
- `Run` ganha `kind`, `review_of`, `review_mode`, `hunt_seed`, `variant_group`, `variant_index`,
  `agent_state`, `attention`, `continuation`, `context`, `review_comments`; `Outcome` ganha
  `question`, `session`, `overview`, `notes`, `diff_stats`, `review`, `findings`,
  `review_replies`, `artifacts`.
- Status `needs_input` e fase `awaiting_input`: a única mudança que quebra.
- Schema JSON, fixture nova `execution/testdata/run-v1-needs-input.json`, script de
  conformidade comparando também a fixture nova, protocolo §9 e §10 nos dois idiomas, CLI 0.4.0.

## Fora de escopo

Implementar os campos no runner e no cloud (DV-01 a DV-14); o enum fechado de `failure_class`
(RM-050).

## Impacto de segurança

- Ativos: o contrato de fio runner ↔ cloud; o texto do agente que passa a atravessar o fio
  (pergunta, overview, notas, achados) e os bytes de artefatos.
- Fronteiras: runner → cloud (texto do agente, limitado e validado); cloud → agente (notas e
  respostas voltam a um prompt).
- Controles: ASVS V5 (validação de entrada com limites por campo, caminho relativo sem `..` nos
  achados, tipos de artefato fechados), defesa contra injeção de prompt (`QuoteContext` cita
  cada linha e rotula o bloco como informação). Achados e comentários não carregam código.

## Evidência

- `make test` (unidade da máquina de estados, limites, validação, citação, corte UTF-8, fixture).
- `make conformance` contra os irmãos depois que copiarem a fixture e o schema.
