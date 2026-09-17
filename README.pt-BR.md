# trilha-spec

> [🇺🇸 English](README.md) · 🇧🇷 Português
>
> **Capítulo hands-on no site do Trilha:** <https://emersonjoe.github.io/trilha/pt/aprender/agentico-protocolo>

**Um padrão aberto para descrever trabalho executável por agentes de software.**

`trilha-spec` é a metade pública e sem dependências da estratégia de desenvolvimento agentic
do [Trilha](https://github.com/emersonjoe/trilha). Diz *o que* fazer — especificações, tasks,
grafo de tasks, o contexto que um agente recebe, a evidência que uma task precisa deixar — e
nada sobre *como* executar. Execução (worktrees, sandboxes, workers, filas) é o
[trilha-runner](https://github.com/emersonjoe/trilha-runner); control plane, fleet e
governança são o trilha-cloud.

```
.trilha/
├── project.md        o que é este projeto, para um agente que acabou de chegar
├── constitution.md   as regras que toda task obedece
├── specs/            001-oauth.md — o que construir e por quê
├── tasks/            TASK-001.md — unidades executáveis com critérios de aceite
├── agents/           coder.md, reviewer.md — quem pode executar, com o quê
├── context/          documentos extras que um agente recebe
├── evidence/         TASK-001/001-check.json — a prova que uma task produziu
└── runs/             rascunho do runner; nunca commitado
```

Tudo é Markdown com um front matter pequeno — legível por uma pessoa, por um diff e por um
modelo — e o módulo depende só da biblioteca padrão do Go, para que qualquer ferramenta leia.

## Uma task

```markdown
---
id: TASK-002
title: Implementar login OAuth
status: ready
spec: 001-oauth
agent: coder
depends_on:
  - TASK-001
acceptance:
  - Login OAuth funciona com o provedor descrito em context/
  - Testes passam
checks:
  - go test ./...
---
```

Uma task percorre uma vida estrita:

```
idea → spec → ready → running → verify → review → done
                 ↑         │        │        │
                 └── blocked / failed ←──────┘
```

`ready` exige critérios de aceite; `running` exige toda dependência `done`; `verify` roda os
checks e grava cada um como **evidência** — código de saída, hash da saída, quem rodou,
quando — e o revisor decide pela evidência, não pelo log do chat.

## A CLI

```bash
go install github.com/emersonjoe/trilha-spec/cmd/trilha-spec@latest

trilha-spec init --name meu-app
trilha-spec spec new "Login OAuth"
trilha-spec spec set 001-login-oauth --asset "cookie de sessão" --control "ASVS V3.4" \
    --evidence "go test ./internal/auth/..."   # impacto de segurança, entregue ao agente e ao revisor
trilha-spec spec move 001-login-oauth approved  # draft → approved → done; rejected, superseded
trilha-spec task add "Config do provedor" --spec 001-login-oauth --status ready \
    --accept "config carrega" --check "go test ./internal/oauth/..." \
    --body "Lê config.yaml; falha alto quando falta chave."   # ou --body-file CAMINHO (- lê stdin)
trilha-spec task next                          # o que pode rodar agora
trilha-spec context TASK-001                   # o pacote que um agente recebe
trilha-spec task move TASK-001 running
trilha-spec task move TASK-001 verify
trilha-spec verify TASK-001                    # roda checks, grava evidência, → review ou failed
trilha-spec evidence TASK-001
trilha-spec task graph                         # Mermaid; --dot para Graphviz
trilha-spec mcp --write                        # o mesmo por MCP, para Claude Code / Cursor
```

Toda listagem aceita `--json`. `trilha-spec doctor` diz onde um leitor tropeçaria.
`TRILHA_LANG=pt` (ou `pt-BR`) põe toda mensagem em português, `--help` incluído, e faz `init` e
`spec new` escreverem seus templates em português; os formatos de arquivo e o `--json` não
mudam, porque nomes de status, de campo e IDs são o protocolo.

### MCP

`trilha-spec mcp` serve o protocolo por stdio a qualquer host MCP. Só leitura por padrão
(`trilha_list_tasks`, `trilha_get_task`, `trilha_next`, `trilha_context`, `trilha_list_specs`,
`trilha_graph`); `--write` acrescenta `trilha_move`, `trilha_spec_move`, `trilha_evidence` e
`trilha_verify`. Ferramenta não
oferecida não pode ser chamada.

## Pacotes

| Pacote | O que é |
|---|---|
| `spec` | parser/escritor de front matter, o layout `.trilha/`, especificações, `project.md` |
| `task` | o modelo de task, status e transições, o store em arquivos, o grafo, evidência e checks |
| `agent` | manifestos de agente: papel, driver, ferramentas permitidas, restrições |
| `ai` | o pacote de contexto: tudo que um agente recebe para uma task, em Markdown ou JSON |
| `mcp` | o protocolo como ferramentas MCP por stdio |
| `cmd/trilha-spec` | a CLI |

## Relação com o framework `trilha`

A CLI do framework é `trilha`; esta é `trilha-spec`. Não dividem binário porque o `trilha` já
é dono de `mcp`, `agents`, `ctx`, `check` e do diretório `.trilha/` como cache de build. O
[docs/adr/001](docs/adr/001-tres-repositorios.md) registra os conflitos e as duas mudanças que
o framework precisa para que `trilha spec …` despache para cá, no estilo do git.

O protocolo completo está em [docs/pt-BR/protocol.md](docs/pt-BR/protocol.md); o que ainda
falta está em [docs/pt-BR/roadmap.md](docs/pt-BR/roadmap.md).

## Licença

MIT.
