# Roadmap

> [🇺🇸 English](../roadmap.md) · 🇧🇷 Português

O que o protocolo ainda não tem, descoberto ao usá-lo. Cada item é uma task `idea` no
`.trilha/tasks/` deste repositório (`trilha-spec task list --status idea`); vira uma spec em
`specs/` quando for puxado, e a versão do protocolo sobe quando um campo ou uma transição
muda (§9 do protocolo).

## Em aberto

Nada no momento: `trilha-spec task list --status idea` está vazio. As próximas lacunas virão
do runner e do control plane usando o protocolo 0.2.

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

## Conflito conhecido

O dev server do framework `trilha` escreve `*` em `.trilha/.gitignore`, escondendo o protocolo
do git; `trilha-spec doctor` acusa e `init` reescreve o arquivo. A correção definitiva é a
mudança no framework registrada no [adr/001](../adr/001-tres-repositorios.md).
