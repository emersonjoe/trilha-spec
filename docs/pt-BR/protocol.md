# O protocolo Trilha, versão 0.2

> [🇺🇸 English](../protocol.md) · 🇧🇷 Português

Esta é a descrição normativa do que mora em `.trilha/`. O módulo Go deste repositório é a
implementação de referência; uma ferramenta em qualquer linguagem que siga esta página fala o
protocolo.

## 1. Layout

```
.trilha/
├── .gitignore        runs/, keys/*.key e o cache de build do framework; o resto é commitado
├── project.md        Documento: ver §12
├── constitution.md   Markdown livre
├── specs/NNN-nome.md Documento: ver §11
├── tasks/TASK-NNN.md Documento: ver §3
├── agents/nome.md    Documento: name, role, driver, command, model, tools[], constraints[]
├── context/*.md      Markdown livre; todo arquivo vai para o agente
├── evidence/TASK-NNN/NNN-kind.json   ver §5
├── keys/<key_id>.pub chaves públicas Ed25519, PEM; ver §5 — chave privada nunca fica aqui
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
chave:
  sub: escalar
  sub: [a, b]
chave:
  - sub: escalar
    sub: escalar
  - sub: escalar
---

Corpo, Markdown livre.
```

A gramática é de propósito um subconjunto de YAML: escalares, listas de escalares, um **bloco**
de escalares e listas sob uma chave (`chave: {}` é um bloco vazio), uma **lista de blocos**,
comentários `#` e linhas em branco. O aninhamento para aí: um bloco tem escalares e listas,
nunca outro bloco. Um item `- ` que se lê como `sub: valor` sem aspas abre um bloco; um item
escalar com dois-pontos vai entre aspas, que é como o escritor o emite, e uma lista nunca
mistura os dois. Entre aspas duplas, `\"` e `\\` são os únicos escapes; aspas simples
carregam o texto como está. Chave desconhecida é preservada na escrita. A ordem dos campos é
estável: campos do protocolo primeiro, na ordem abaixo, depois os desconhecidos na ordem lida.

## 3. Task

| Campo | Obrigatório | Significado |
|---|---|---|
| `id` | sim | `TASK-NNN`, NNN ≥ 3 dígitos, igual ao nome do arquivo |
| `title` | sim | uma linha |
| `status` | sim | §4; ausente vale `idea` |
| `spec` | não | o ID da especificação que implementa |
| `milestone` | não | o marco (§12) a que pertence; vazio vale o da sua spec |
| `agent` | não | nome de agente; senão `project.default_agent` |
| `depends_on` | não | IDs de task; todos precisam existir; sem ciclo |
| `covers` | não | IDs de requisito que a task entrega (§11); todos precisam ser declarados por uma spec |
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
  as executáveis em ordem de dependência, empate por ID, e entre elas o prazo de marco (§12)
  mais próximo primeiro; task sem prazo vai por último.

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

Um registro `run` pode carregar seu **custo** em campos padrão, para que ledgers de runners
diferentes conciliem: `provider` (`anthropic`), `model` (`claude-sonnet-5`), `tokens_in`,
`tokens_out`, `cost` e `currency` (ISO 4217, obrigatório quando há `cost`). `cost` é o que o
runner observou; o protocolo não afirma que foi verificado — conciliar com a fatura do
provedor é assunto do control plane, e preço por modelo não está no protocolo. Uma Attempt
do contrato de execução (§10) usa os mesmos nomes.

Um registro pode ser **assinado**, para que o revisor distinga o que um runner atestou do que
qualquer um poderia ter digitado. A assinatura é um campo, omitido quando o registro não é
assinado:

```json
"signature": { "alg": "ed25519", "key_id": "runner-01", "sig": "<base64>" }
```

Os bytes assinados são a forma canônica do registro: o objeto JSON sem `signature`, chaves
ordenadas, sem espaço insignificante, sem escape de HTML — a forma que qualquer biblioteca
JSON reproduz a partir do arquivo. `alg` é só `ed25519`; `key_id` são palavras minúsculas
unidas por `-` ou `.`; a chave pública fica em `keys/<key_id>.pub` em PEM (PKIX) e é
commitada, enquanto a privada fica fora de `.trilha` (`doctor` acusa uma que esteja dentro).
Um leitor confere cada registro e responde um de três vereditos: `unsigned` (sem assinatura —
o registro é uma alegação), `valid` (a assinatura confere com o registro sob a chave nomeada)
ou `invalid` (não confere, a chave é desconhecida ou o algoritmo não é `ed25519`). Assinar é
opcional: registro não assinado continua sendo registro. Registro editado fica `invalid`, e
esse é o ponto.

`verify` roda os `checks` da task e depois `project.verify`, grava um `check` por comando, não
para em falha, e responde *passou* só quando todo código de saída é 0. Task sem check nenhum
falha a verificação com uma `note` dizendo isso.

## 6. Pacote de contexto

O que um agente recebe para uma task, nesta ordem: projeto, constituição, o próprio manifesto,
a especificação, a task (corpo, o marco dela com o prazo, aceite, os requisitos que ela cobre
com o texto deles, checks, dependências com status, evidência até aqui),
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
| `trilha_list_tasks` | | `status?`, `milestone?` |
| `trilha_get_task` | | `id` |
| `trilha_next` | | — ; projeto pausado (§12) responde erro com o motivo |
| `trilha_context` | | `id`, `format?` (markdown \| json) |
| `trilha_graph` | | |
| `trilha_list_specs` | | `status?` |
| `trilha_list_evidence` | | `id`; cada registro com seu `verdict` (e `reason` quando inválido) |
| `trilha_coverage` | | `spec?`; a matriz de requisitos da §11 |
| `trilha_move` | sim | `id`, `status` |
| `trilha_evidence` | sim | `id`, `kind` (note \| artifact), `note?`, `files?`, `by?` |
| `trilha_verify` | sim | `id`, `by?` |
| `trilha_spec_move` | sim | `id`, `status` (§11) |

Ferramentas de escrita só aparecem com `--write`; ferramenta não listada não pode ser chamada.

## 9. Versionamento

Esta página é a versão 0.2. Mudança de campo, transição ou nome de arquivo sobe a versão e é
registrada em uma spec em `specs/`. A 0.2 (specs 003–008) acrescentou `rejected`/`superseded`
e relações entre specs, impacto de segurança na spec, limites e pausa do projeto, campos de
custo na evidência `run` e evidência assinada; toda adição é campo novo opcional, então um
leitor 0.1 ainda lê um diretório 0.2. Leitores devem tolerar campos e tipos de evidência
desconhecidos.

## 10. Contrato de execução

O transporte de Runs entre control planes e runners usa `trilha.execution/v1`. O JSON Schema
normativo está em `contracts/execution/v1/schema.json`; `execution/testdata/run-v1.json` é o
fixture canônico de compatibilidade. Uma Run é uma execução lógica de uma Task e pode conter
várias Attempts. A linhagem de correção usa `retry_of` com ID de Run, nunca ID de Task. Uma
Attempt carrega seu custo com os nomes do registro de evidência `run` (§5): `provider`,
`model`, `tokens_in`, `tokens_out`, `cost`, `currency`.

## 11. Especificação

| Campo | Obrigatório | Significado |
|---|---|---|
| `id` | sim | `NNN-nome`, igual ao nome do arquivo; a numeração de branch do spec-kit |
| `title` | sim | uma linha |
| `status` | sim | abaixo; ausente significa `draft` |
| `issue` | não | a issue que é a fonte do escopo |
| `milestone` | não | o marco (§12) a que a spec inteira pertence |
| `supersedes` | não | IDs de spec que esta substitui; todos precisam existir |
| `depends_on` | não | IDs de spec em que esta se apoia; todos precisam existir |
| `assets` | não | o que a mudança toca, para o revisor de segurança: identificadores livres |
| `trust_boundaries` | não | as fronteiras que ela cruza (`browser → api`) |
| `controls` | não | os controles que ela afeta (`ASVS V4.1`); identificadores livres |
| `evidence` | não | comandos que um revisor precisa ver rodar: programa e argumentos, sem shell, como `checks` de task (§3) |
| `requirements` | não | requisitos externos que esta spec responde, um bloco cada: `id`, `source`, `text` |

Os quatro campos de segurança são o **impacto de segurança** da spec. O protocolo os carrega e
não julga nada sobre eles: se os controles bastam é decisão do revisor. O pacote de contexto
(§6) os entrega ao agente ao lado do aceite e dos checks da task; uma spec `approved` que não
declara nenhum deles é um aviso do `doctor`, não uma falha.

### Requisitos

Quando a fonte do escopo é um documento fora do repositório — a lista de requisitos de um
edital, um artigo de lei, um KPI de contrato — a spec carrega a referência e as tasks apontam
de volta para ela:

```
requirements:
  - id: D2-R8
    source: "cp-01-2026#anexo-I"
    text: informar o cidadao sobre o andamento
```

Um `id` é identificador livre sem espaço nem vírgula, para que `covers: [D2-R8, D2-R9]` nunca
parta um ao meio; é único no projeto, porque é por ele que `covers` o nomeia. O protocolo não
julga `source` nem `text`: carrega, entrega ao agente no pacote de contexto (§6) e responde a
matriz — requisito → tasks → status → quantidade de evidência — em `spec show --coverage` e em
`trilha_coverage` (§8). O `doctor` aponta requisito que nenhuma task cobre (aviso: o escopo
está declarado, o trabalho ainda não foi cortado), task que cobre um id que nenhuma spec
declara, e o mesmo id declarado por duas specs.

O corpo é a especificação: por quê, o que muda, fora de escopo, aceitação. `draft` está sendo
escrita; `approved` foi acordada e pode virar tasks; `done` tem toda task entregue; `rejected`
foi julgada e recusada; `superseded` foi substituída por uma spec que a cita em `supersedes`.

| De | Para |
|---|---|
| draft | approved, rejected, superseded |
| approved | done, rejected, superseded, draft |
| done | superseded |
| rejected | draft |
| superseded | — |

Regras que um escritor faz valer: uma spec nunca referencia a si mesma; toda referência existe
(`doctor` aponta a que não existe); uma spec `superseded` é citada em `supersedes` de pelo menos
uma outra spec, ou `doctor` a aponta como órfã. Estado de task nunca move uma spec: isso é uma
decisão.

## 12. Projeto

`project.md` é a primeira coisa que um agente lê. O corpo é prosa; o front matter é o que as
ferramentas consomem.

| Campo | Obrigatório | Significado |
|---|---|---|
| `name` | sim | o projeto |
| `description` | não | uma linha |
| `default_agent` | não | o agente que uma task sem `agent` recebe |
| `verify` | não | comandos que toda task roda além dos próprios checks |
| `milestones` | não | os marcos datados do programa, um bloco cada: `id`, `title`, `due`, `gate` |
| `limits` | não | um mapa de limiares numéricos, abaixo |
| `paused` | não | `true` para a fila: `next` não responde nada e diz por quê |
| `pause_reason` | não | texto livre; `breaker:<limite>` quando um control plane disparou por um limite |
| `paused_at` | não | RFC 3339 UTC; obrigatório quando `paused` |

`limits` é o envelope do projeto. O protocolo nomeia três chaves e carrega qualquer outra:

| Chave | Significado |
|---|---|
| `max_cost_per_hour` | na moeda da evidência (§5) |
| `max_failure_rate` | uma fração, 0..1, sobre as últimas execuções |
| `max_repeated_failure_class` | o mesmo `failure_class` tantas vezes seguidas |

### Marcos

Um contrato com marcos pagos, ou um programa reportado contra um calendário, declara-os uma
vez em `project.md`:

```
milestones:
  - id: M2
    title: PoC entregue
    due: 2027-04-30
    gate: aceite do cliente
```

O `id` segue a regra do id de requisito (§11); `due` é um dia de calendário, `AAAA-MM-DD`, não
um instante. Uma spec e uma task nomeiam um em `milestone`, e uma task sem o próprio herda o da
spec, como uma task sem `agent` recebe o `default_agent`. `task list --milestone M2` filtra uma
listagem; `next` prefere o prazo mais próximo entre o que pode rodar; o pacote de contexto (§6)
entrega ao agente o marco e a data. O `doctor` aponta marco sem nenhuma task e task não
concluída com o prazo do marco vencido — os dois como aviso, porque o protocolo reporta o
calendário e são as pessoas que o mantêm — e, como falha, task ou spec que nomeia um marco que
o `project.md` não declara. Estimativa, capacidade e quanto vale um marco são assunto do
control plane.

O protocolo *carrega* limites, marcos e pausa; fazê-los valer — recusar iniciar uma task, parar uma
tentativa em curso, disparar o disjuntor — é trabalho do runner e do control plane, como com os
manifestos de agente (§7). O pacote de contexto (§6) inclui `limits` e a pausa, para o agente
conhecer seu envelope. Todo leitor pode ignorar todos esses campos.
