# Roadmap

> [🇺🇸 English](../roadmap.md) · 🇧🇷 Português

O que o protocolo ainda não tem, descoberto ao usá-lo. Cada item é uma task `idea` no
`.trilha/tasks/` deste repositório (`trilha-spec task list --status idea`); vira uma spec em
`.trilha/specs/` quando for puxado (adr/003), e a versão do protocolo sobe quando um campo ou uma transição
muda (§9 do protocolo).

## Em aberto

Cada item é uma task `idea` aqui (`trilha-spec task list --status idea`) e um item do roadmap
da stack (`trilha-cloud/docs/ROADMAP-STACK.md`, fase 4). Nenhum começou.

| Task | Roadmap | Lacuna |
|---|---|---|
| TASK-024 | RM-050 | `failure_class` é vocabulário recomendado (spec 014), não enum; virar enum em `execution/contract.go` e no schema, validar no `doctor`, subir para 0.4. |
| TASK-025 | RM-052 | A superfície de escrita do MCP para em `trilha_evidence` (`note`/`artifact`) e num `trilha_attest` sem assinatura; acrescentar `run`/`eval`, `--sign-key`, `trilha_task_add`. |
| TASK-026 | RM-053 | O contrato de execução tem `cancelled` e `preview`; o ciclo de vida da task (§4) não tem `cancelled` e Preview só existe no `CONTEXT.md`. |
| TASK-027 | RM-054 | `task.Resolver` não tem implementação remota; o runner resolve por conta própria. Um só caminho de resolução, testado contra a interface. |
| TASK-028 | RM-055 | O caminho até a 1.0: JSON Schema de todo front matter, `doctor --schema`, seção de compatibilidade e um comando público de conformidade. |

A fase 0 do mesmo roadmap (spec 015: CI formatando o módulo inteiro, job `conformance` contra o
runner, releases com binários em tags `v*`, adr/003 sobre a fonte única da verdade e o aviso
`spec-tasks-done` do doctor) é `.trilha/specs/015-fase-0-do-roadmap.md`.

## Entregue na 0.3

Vieram do planejamento de um programa de setor público que atravessa quatro repositórios — o
app do produto, o framework, o runner e o control plane — e precisa ser auditável pelo
comprador em todo marco. Cada um é uma spec em `specs/`
e uma task `done` com evidência.

| Task | Lacuna | Por que importava |
|---|---|---|
| TASK-016 · spec 009 · [#10](https://github.com/emersonjoe/trilha-spec/issues/10) | Quando a fonte do escopo é um **documento externo** — a lista de requisitos de um edital, uma lei, um KPI de contrato — nada liga as tasks a ele. | "Quais requisitos estão cobertos, por qual task, com que evidência" era planilha mantida à mão, e é a pergunta que um comprador público faz em todo marco. |
| TASK-018 · spec 010 · [#12](https://github.com/emersonjoe/trilha-spec/issues/12) | O protocolo tinha status e dependências, mas **nenhum calendário**: não dava para dizer "esta task é do M2, com prazo 2027-04-30". | Um contrato com marcos pagos reportava o cronograma físico-financeiro fora do protocolo. |
| TASK-014 · spec 011 · [#8](https://github.com/emersonjoe/trilha-spec/issues/8) | Qualidade se mede com **números**, e o único jeito de gravar um era um `check` cujo código de saída o escondia. | O revisor não via o valor, nada o acompanhava entre execuções e um control plane não barrava nada com ele. |
| TASK-015 · spec 012 · [#9](https://github.com/emersonjoe/trilha-spec/issues/9) | `review → done` era decisão de uma pessoa só, e nada distinguia "alguém digitou uma nota" de "esta pessoa, neste papel, **atesta** isto — e assinou". | Entrega no setor público termina em aceite humano nomeado, às vezes por um comitê. |
| TASK-017 · spec 013 · [#11](https://github.com/emersonjoe/trilha-spec/issues/11) | `depends_on` era só local, então um programa entre repositórios escrevia as dependências reais em **prosa**. | O `next` oferecia trabalho que não podia começar, e ninguém desenhava o grafo. |

## Entregue na 0.2

Os itens abaixo vieram de escrever a spec 011 do trilha-cloud (disjuntor de projeto e pausa da
frota) com `trilha-spec` em vez de spec-kit. Cada um é uma spec em `specs/` e uma task `done`
com evidência.

| Task | Lacuna | Por que importava |
|---|---|---|
| TASK-009 · spec 005 · [#1](https://github.com/emersonjoe/trilha-spec/issues/1) | Uma spec não tem lugar para **impacto de segurança**: ativos, fronteiras de confiança, controles afetados (ASVS), comandos de evidência. A constituição do trilha-cloud exige isso, então vive como seções livres no corpo que o `doctor` e o pacote de contexto não enxergam. | O revisor decide pela evidência; a evidência de segurança deveria ser tão visível às ferramentas quanto `acceptance` e `checks`. |
| TASK-010 · spec 004 · [#2](https://github.com/emersonjoe/trilha-spec/issues/2) | O **status** da spec é `draft \| approved \| done`; nada para `rejected` ou `superseded`, nenhuma relação entre specs (`supersedes`, `depends_on`), e a CLI não muda status nem `issue` de uma spec (só `spec new \| list \| show`). | Specs são editadas à mão depois de criadas; as ferramentas não respondem "qual spec substituiu esta". |
| TASK-011 · spec 006 · [#3](https://github.com/emersonjoe/trilha-spec/issues/3) | `project.md` não tem **limites** (`max_cost_per_hour`, `max_failure_rate`, `max_repeated_failure_class`) nem estado **pausado**. Uma task carrega `token_budget`; um projeto não carrega nada. | Cada control plane inventa o próprio disjuntor, e um runner não consegue ler os mesmos limiares antes de começar. |
| TASK-012 · spec 007 · [#4](https://github.com/emersonjoe/trilha-spec/issues/4) | A evidência `run` tem um `meta{}` livre; **custo** não é padronizado (`provider`, `model`, `tokens_in`, `tokens_out`, `cost`, `currency`). | Dois runners reportam custo de formas diferentes; um control plane não concilia o ledger com a fatura do provedor. |
| TASK-013 · spec 003 · [#5](https://github.com/emersonjoe/trilha-spec/issues/5) | `task add` não tem `--body`; `spec new` e `task add` só escrevem template em inglês; `--status` funciona mas não aparece no `--help`. | Um projeto em pt-BR reescreve a spec à mão logo depois de criá-la; a CLI poderia fazer o trabalho inteiro. |

### Anteriores, também entregues

| Task | Lacuna |
|---|---|
| TASK-007 · spec 002 · [#6](https://github.com/emersonjoe/trilha-spec/issues/6) | Mensagens da CLI em pt-BR via `TRILHA_LANG`. |
| TASK-008 · spec 008 · [#7](https://github.com/emersonjoe/trilha-spec/issues/7) | Evidência assinada, para um runner remoto cuja evidência cruza uma fronteira de confiança. |

## Resolvido

O dev server do framework `trilha` escrevia `*` em `.trilha/.gitignore`, escondendo o protocolo
do git. O `trilha` 0.124.0 (2026-09-12) moveu o cache de build para `.trilha/cache/`, e o
`.gitignore` do scaffold passou a ignorar só `.trilha/cache/` e `.trilha/runs/`, como a
[adr/001](../adr/001-tres-repositorios.md) pedia. `trilha-spec doctor` continua acusando um
`.gitignore` com `*` e `init` continua reescrevendo-o, para projetos criados antes dessa versão.
