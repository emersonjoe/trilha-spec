# Spec 004 — Ciclo de vida de spec: `rejected`, `superseded`, relações e `spec move`

- **Issue**: [#2](https://github.com/emersonjoe/trilha-spec/issues/2) — a issue é a fonte do escopo.
- **Task**: `TASK-010` em `.trilha/tasks/`.
- **Versão**: 0.2 do protocolo (§11 novo); arquivos existentes continuam válidos.

## Por quê

Uma spec só conhecia `draft`, `approved` e `done`, e nada movia uma spec pela CLI: a pessoa
editava o front matter na mão. Uma spec recusada ou substituída por outra ficava `draft` para
sempre, e nenhuma spec dizia de qual outra dependia — o leitor não sabia o que era
decisão e o que era abandono.

## O que muda

- `status` de spec ganha `rejected` e `superseded`; a tabela de transições é
  `spec.Transitions` (§11 do protocolo): `draft → approved|rejected|superseded`,
  `approved → done|rejected|superseded|draft`, `done → superseded`, `rejected → draft`,
  `superseded` é final.
- Campos `supersedes` e `depends_on` (listas de IDs de spec). Uma spec nunca se referencia;
  toda referência precisa existir; uma `superseded` que nenhuma spec cita em `supersedes` é
  órfã. `Layout.CheckSpecs` devolve `spec.Problem` com código, e `doctor` os mostra.
- CLI: `spec move <id> <status>`, `spec set <id> [--issue N|-] [--supersedes A,B]
  [--depends A,B]`, `spec list --status S`. Mensagens no catálogo pt-BR.
- MCP: `trilha_list_specs` (leitura, `status?`) e `trilha_spec_move` (escrita).
- `spec.Spec.Move`, `spec.Status.CanMoveTo`, `Layout.MoveSpec`.

```bash
trilha-spec spec move 001-login-oauth approved
trilha-spec spec set 002-login-passkey --supersedes 001-login-oauth
trilha-spec spec move 001-login-oauth superseded
trilha-spec spec list --status superseded
```

## Fora de escopo

- Mover a spec a partir do estado das tasks: `done` é uma decisão, não um cálculo.
- Bloquear tasks de uma spec `rejected`.

## Constitution Check

| Princípio | Como respeita |
|---|---|
| I — o protocolo é o produto | §11 descreve campos e transições; o código segue o documento |
| II — Markdown + front matter | `supersedes`/`depends_on` são listas de escalares |
| V — teste primeiro | `TestSpecLifecycle`, `TestSpecRelationsRoundTripAndCheck`, e2e `TestSpecLifecycleCLI` |

## Tarefas

- [x] T001 Testes: transições, validação de referências, round trip, doctor, CLI e MCP
- [x] T002 `spec.Status`, `Transitions`, `Move`, `CheckSpecs`; `spec move|set|list --status`
- [x] T003 Protocolo §1/§8/§11 (en, pt-BR), README (en, pt-BR), capítulo hands-on nas duas locales
- [x] T004 `make test` verde; TASK-010 com evidência

## Aceitação

- **SC-001** `spec move X done` a partir de `draft` falha com `cannot move from draft to done`.
- **SC-002** `spec set X --supersedes Y` com `Y` inexistente falha e não grava.
- **SC-003** `doctor` aponta uma spec `superseded` que ninguém cita em `supersedes`.
