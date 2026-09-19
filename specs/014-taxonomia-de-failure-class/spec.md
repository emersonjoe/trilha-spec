# Spec 014 — Taxonomia de `failure_class`/`error_code` no contrato de execução

> Forma curta: documentação de um campo já existente, nenhum pacote novo, nenhuma convenção
> nova em `app/`, nenhuma mudança incompatível na API pública.

- **Issue**: [#15](https://github.com/emersonjoe/trilha-spec/issues/15) — a issue é a fonte do escopo.
- **Task**: `TASK-019` em `.trilha/tasks/`.
- **Versão**: 0.3 do protocolo (§10); só prosa, nenhum campo, transição ou nome de arquivo muda.

## Por quê

`contracts/execution/v1/schema.json` já reserva `failure_class` e `error_code` no objeto
`attempt`, mas o schema não restringe nem documenta os valores possíveis — nenhum `enum`,
nenhuma menção em `docs/protocol.md`. Na prática, hoje nenhum runner popula esses campos com um
valor estável: toda falha de attempt chega com o mesmo status genérico `checks_failed`, mesmo
quando a causa real é bem diferente (ex.: `exit 127` por ferramenta ausente no worker vs. um
teste que de fato rodou e reprovou). Isso foi observado ao vivo no projeto `acervo`
(`run-000007`, `TASK-011`): três tentativas seguidas de `sh -c "cd api && python -m pytest -q"`
falhando com `exit 127` por falta de `python` no worker, todas reportadas como `checks_failed`
sem distinção.

## O que muda

- `docs/protocol.md` e `docs/pt-BR/protocol.md`, §10, ganham "Failure taxonomy" /
  "Taxonomia de falha": uma tabela com o vocabulário recomendado para `failure_class`
  (`environment_missing_tool`, `test_failure`, `lint_failure`, `timeout`,
  `provider_transient`, `agent_error`) e o papel de `error_code` ao lado dele.
- `provider_transient` entra na lista porque já é o valor usado em
  `execution/testdata/run-v1.json` — a taxonomia documenta o que o fixture de compatibilidade
  já assume, não substitui.
- O campo continua texto livre no schema: a lista é o vocabulário recomendado para um runner
  reaproveitar quando a falha se encaixa, não um `enum` fechado. Um valor fora da lista
  continua protocolo válido — fechar a lista quebraria compatibilidade com qualquer runner que
  já grava uma categoria própria, o que contraria §9 (leitor tolera campo desconhecido).
- Nenhum campo, transição de status ou nome de arquivo muda; a versão do protocolo (0.3)
  permanece a mesma.

## Fora de escopo

- Popular `failure_class`/`error_code` no `trilha-runner` ao detectar `exit 127` —
  emersonjoe/trilha-runner#13, fora deste repositório.
- Renderizar esses valores no Attempt Timeline — emersonjoe/trilha-cloud#29, fora deste
  repositório.
- Fechar `failure_class` como `enum` no JSON Schema — ver acima, quebraria compatibilidade.

## Constitution Check

| Princípio | Como respeita |
|---|---|
| I — o protocolo é o produto | a taxonomia entra como prosa normativa em `docs/protocol.md` antes de qualquer código a consumir; não é mudança de formato ou transição, então não pede versão nova |
| Idioma | seção adicionada em inglês e em `docs/pt-BR/` no mesmo commit |

## Tarefas

- [x] T001 §10 de `docs/protocol.md` (en) com a tabela de `failure_class` e o papel de `error_code`
- [x] T002 §10 de `docs/pt-BR/protocol.md` (pt) com a mesma tabela
- [x] T003 `make test` verde

## Aceitação

- **SC-001** `docs/protocol.md` e `docs/pt-BR/protocol.md` documentam um vocabulário mínimo para
  `failure_class`, incluindo `provider_transient` (já em uso no fixture de compatibilidade).
- **SC-002** `contracts/execution/v1/schema.json` e `execution/testdata/run-v1.json` continuam
  validando sem mudança — o campo segue texto livre.
- **SC-003** `make test` passa.
