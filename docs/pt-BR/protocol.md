# O protocolo Trilha, versão 0.1

> [🇺🇸 English](../protocol.md) · 🇧🇷 Português

Esta é a descrição normativa do que mora em `.trilha/`. O módulo Go deste repositório é a
implementação de referência; uma ferramenta em qualquer linguagem que siga esta página fala o
protocolo.

## 1. Layout

```
.trilha/
├── .gitignore        runs/ e o cache de build do framework; o resto é commitado
├── project.md        Documento: name, description, default_agent, verify[]
├── constitution.md   Markdown livre
├── specs/NNN-nome.md Documento: id, title, status (draft|approved|done), issue
├── tasks/TASK-NNN.md Documento: ver §3
├── agents/nome.md    Documento: name, role, driver, command, model, tools[], constraints[]
├── context/*.md      Markdown livre; todo arquivo vai para o agente
├── evidence/TASK-NNN/NNN-kind.json   ver §5
└── runs/             opaco ao protocolo
```

O diretório é encontrado subindo a partir do diretório de trabalho, como o `.git`.

## 2. Documentos

Um *Documento* é Markdown UTF-8 que começa com um bloco de front matter:

```
---
chave: escalar
chave: "escalar: com aspas"
chave: [a, b]
chave:
  - item
  - item
---

Corpo, Markdown livre.
```

A gramática é de propósito um subconjunto de YAML: escalares, listas de escalares, comentários
`#` e linhas em branco. Entre aspas duplas, `\"` e `\\` são os únicos escapes; aspas simples
carregam o texto como está. Chave desconhecida é preservada na escrita. A ordem dos campos é
estável: campos do protocolo primeiro, na ordem abaixo, depois os desconhecidos na ordem lida.

## 3. Task

| Campo | Obrigatório | Significado |
|---|---|---|
| `id` | sim | `TASK-NNN`, NNN ≥ 3 dígitos, igual ao nome do arquivo |
| `title` | sim | uma linha |
| `status` | sim | §4; ausente vale `idea` |
| `spec` | não | o ID da especificação que implementa |
| `agent` | não | nome de agente; senão `project.default_agent` |
| `depends_on` | não | IDs de task; todos precisam existir; sem ciclo |
| `acceptance` | de `ready` em diante | o que precisa ser verdade para fechar, em palavras |
| `checks` | não | comandos que o `verify` roda; programa + argumentos, sem shell |
| `attempt`, `max_attempts`, `token_budget` | não | limites de execução e metadados da tentativa atual |
| `retry_of` | não | ID da Run anterior (`run-NNNNNN`) quando esta task é uma correção |
| `failure_class`, `repair_reason` | não | categoria estável da falha e intenção humana de correção |
| `created`, `updated` | não | RFC 3339 UTC |

O corpo é o detalhe da especificação que o agente lê.

## 4. Ciclo de vida

```
idea → spec → ready → running → verify → review → done
```

| De | Para |
|---|---|
| idea | spec, ready |
| spec | ready, idea |
| ready | running, blocked, spec |
| running | verify, failed, blocked, ready |
| verify | review, failed |
| review | done, ready, failed |
| done | ready (reabrir) |
| blocked | ready |
| failed | ready |

Regras que quem escreve impõe:

- `ready` e `running` exigem pelo menos um critério de aceite.
- `running` exige toda dependência `done`.
- Uma task é **executável** quando está `ready` e toda dependência está `done`. `next` lista
  as executáveis em ordem de dependência, empate por ID.

## 5. Evidência

Um JSON por registro, `evidence/TASK-NNN/NNN-kind.json`, NNN a sequência dentro da task:

```json
{
  "task": "TASK-001",
  "seq": 1,
  "kind": "check",
  "at": "2026-09-12T14:03:11Z",
  "by": "trilha-spec verify",
  "command": "go test ./...",
  "dir": "/work/app",
  "exit_code": 0,
  "output": "ok  \tapp\t0.4s\n",
  "output_sha256": "…",
  "passed": true
}
```

`kind` é `check` (um comando rodou), `note` (pessoa ou agente escreveu algo), `artifact`
(arquivos produzidos; `files[]`) ou `run` (registro de execução de um runner; `meta{}` é
livre). `output` pode ser truncado em 64 KiB; `output_sha256` é o hash do todo. Um registro
nunca é editado; correção é registro novo.

`verify` roda os `checks` da task e depois `project.verify`, grava um `check` por comando, não
para em falha, e responde *passou* só quando todo código de saída é 0. Task sem check nenhum
falha a verificação com uma `note` dizendo isso.

## 6. Pacote de contexto

O que um agente recebe para uma task, nesta ordem: projeto, constituição, o próprio manifesto,
a especificação, a task (corpo, aceite, checks, dependências com status, evidência até aqui),
todo arquivo de `context/`. Markdown para prompt, JSON para ferramenta. O pacote termina
dizendo ao agente para não marcar a task como done: isso é decisão do revisor.

## 7. Manifesto de agente

`name` (palavras minúsculas unidas por `-`, igual ao nome do arquivo), `role`, `driver` (`exec`,
`ai`; um runner pode acrescentar), `command`, `model`, `tools` ⊂ {read, write, run, git,
network}, `constraints[]`. O protocolo carrega o manifesto; fazê-lo valer é papel do runner.

## 8. MCP

`trilha-spec mcp` serve JSON-RPC 2.0 por stdio, uma mensagem por linha, revisão MCP
2025-03-26, só a capacidade `tools`.

| Ferramenta | Escreve? | Argumentos |
|---|---|---|
| `trilha_list_tasks` | | `status?` |
| `trilha_get_task` | | `id` |
| `trilha_next` | | |
| `trilha_context` | | `id`, `format?` (markdown \| json) |
| `trilha_graph` | | |
| `trilha_move` | sim | `id`, `status` |
| `trilha_evidence` | sim | `id`, `kind` (note \| artifact), `note?`, `files?`, `by?` |
| `trilha_verify` | sim | `id`, `by?` |

Ferramentas de escrita só aparecem com `--write`; ferramenta não listada não pode ser chamada.

## 9. Versionamento

Esta página é a versão 0.1. Mudança de campo, transição ou nome de arquivo sobe a versão e é
registrada em uma spec em `specs/`. Leitores devem tolerar campos e tipos de evidência
desconhecidos.

## 10. Contrato de execução

O transporte de Runs entre control planes e runners usa `trilha.execution/v1`. O JSON Schema
normativo está em `contracts/execution/v1/schema.json`; `execution/testdata/run-v1.json` é o
fixture canônico de compatibilidade. Uma Run é uma execução lógica de uma Task e pode conter
várias Attempts. A linhagem de correção usa `retry_of` com ID de Run, nunca ID de Task.
