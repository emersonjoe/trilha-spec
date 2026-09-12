# Spec 001 — Fundação (Fase 0)

- **Issue**: — (primeira spec; a issue nasce com o repositório)
- **Branch**: `001-fundacao`
- **Versão**: 0.1.0

## Por quê

O Trilha tem um framework Go com recursos de IA (`ai`, `ai/mcp`, `trilha agents`, `trilha mcp`,
`bench/agent`), mas o *modo de trabalho* que o próprio repositório usa — spec-kit, uma spec por
sessão, evidência antes de fechar — só existe como convenção de quem mantém. Nenhuma outra
ferramenta consegue ler "o que está pronto para um agente fazer agora" nem "que prova esta
task deixou". A Fase 0 transforma isso em protocolo: um diretório `.trilha/` que qualquer
agente entende, e uma CLI que o manipula.

## O que muda

- **F0.1 `.trilha/`**: `project.md`, `constitution.md`, `specs/`, `tasks/`, `agents/`,
  `context/`, `evidence/`, `runs/` (`spec.Layout`, `spec.Init`).
- **F0.2 Task model**: `task.Task` com id, título, status, spec, agent, `depends_on`,
  `acceptance`, `checks`; arquivo `tasks/TASK-NNN.md` com front matter.
- **F0.3 Ciclo de vida**: `idea → spec → ready → running → verify → review → done`, mais
  `blocked` e `failed`; `task.Transitions` é a única tabela.
- **F0.4 CLI**: `trilha-spec init | spec | task | agent | context | verify | evidence | mcp | doctor`.
- **F0.5 Evidência**: `task.Evidence` em `evidence/TASK-NNN/NNN-kind.json`; `verify` roda os
  checks sem shell e grava um registro por comando.
- **F0.6 MCP**: `trilha-spec mcp [--write]` publica o protocolo a hosts MCP.

## Fora de escopo

- Execução (worktree, agente, sandbox): trilha-runner.
- i18n das mensagens da CLI (`TRILHA_LANG`): próxima spec; documentação já sai nas duas línguas.
- Assinatura de evidência e cadeia de custódia: fase seguinte, quando houver runner remoto.
- Despacho `trilha spec …` a partir do binário do framework: patch no repositório `trilha`
  (ADR 001).

## Constitution Check

| Princípio | Como respeita |
|---|---|
| I — protocolo é o produto | `docs/protocol.md` escrito junto com o código; formato lido sem o módulo |
| II — só stdlib | `go.mod` sem `require` |
| III — evidência verificável | `task.RunChecks` grava exit code + sha256 por comando |
| IV — determinismo | ordem fixa de campos em `Task.Bytes`; grafo por Kahn com menor ID primeiro |
| V — teste primeiro | `spec`, `task`, `agent`, `ai`, `mcp` e e2e da CLI |
| VI — segurança | MCP só leitura sem `--write`; sem shell nos checks; timeout |

## Tarefas

- [x] T001 Parser/escritor de front matter com round-trip estável
- [x] T002 Layout `.trilha/` + `init` idempotente + `doctor`
- [x] T003 Task model, transições, store, grafo (ciclo e dependência inexistente recusados)
- [x] T004 Evidência + `verify` sem shell
- [x] T005 Manifesto de agente + pacote de contexto
- [x] T006 Servidor MCP stdio com ferramentas de leitura e de escrita
- [x] T007 CLI + e2e
- [x] T008 Documentação nas duas línguas + ADR 001

## Aceitação

- **SC-001** `trilha-spec init` em diretório vazio cria `.trilha/` e `doctor` responde saudável.
- **SC-002** Uma task `ready` com dependência aberta não entra em `running`; a mensagem nomeia a dependência.
- **SC-003** `verify` de uma task em `verify` grava uma evidência por check e move para `review` ou `failed`.
- **SC-004** `trilha-spec mcp` responde `initialize`, `tools/list` e `tools/call` por stdio.
- **SC-005** `make test` verde em Go 1.22 e na última estável.
