# Spec 003 — Corpo da task, templates pt-BR e `--status` documentado

- **Issue**: [#5](https://github.com/emersonjoe/trilha-spec/issues/5) — a issue é a fonte do escopo.
- **Task**: `TASK-013` em `.trilha/tasks/`.
- **Versão**: 0.2.0 da CLI; formatos de arquivo inalterados.

## Por quê

`task add` cria a task sem corpo, e o autor abre o arquivo em seguida para escrever o detalhe
que o agente vai ler. `spec new` e `init` escrevem só o template em inglês, mesmo num projeto
cujas specs são em pt-BR, então a pessoa reescreve os títulos logo depois de criar. E
`--status` funciona, mas não aparece no `--help`.

## O que muda

- `task add` e `spec new` ganham `--body TEXTO` ou `--body-file CAMINHO` (`-` lê stdin);
  os dois juntos é erro. Em `spec new`, o corpo substitui o esqueleto.
- `TRILHA_LANG=pt` (mesma detecção da spec 002) seleciona os templates em pt-BR que `init`
  (project.md, constitution.md, coder.md, reviewer.md) e `spec new` escrevem. Chaves de front
  matter nunca mudam de língua.
- `spec.InitOptions.Lang`, `spec.Templates(lang) TemplateSet`,
  `spec.NewSpecDoc(id, title, lang, body)`.
- `task.Task.Bytes()` só escreve `expected_files`, `routes`, `scenarios`, `accessibility` e
  `security_controls` quando têm itens: a task não usa, a task não carrega.
- `--status S` e `--body` no `--help` e no capítulo hands-on.

```bash
TRILHA_LANG=pt-BR trilha-spec init --name agenda        # constituição e agentes em pt-BR
trilha-spec task add "Config" --status ready --accept "carrega" --body-file - < detalhe.md
```

## Fora de escopo

- Traduzir arquivos existentes.
- Template de corpo para `task add`: sem `--body`, a task continua sem corpo.

## Constitution Check

| Princípio | Como respeita |
|---|---|
| I — o protocolo é o produto | só muda o que a CLI escreve; formato e chaves iguais |
| V — teste primeiro | `TestTemplatesByLanguage`; e2e `TestBodyAndTemplates` |

## Tarefas

- [x] T001 Testes: templates por língua, corpo por flag/arquivo/stdin, flags exclusivas, help
- [x] T002 `spec.Templates`, `InitOptions.Lang`, `NewSpecDoc` com corpo; flags na CLI
- [x] T003 README (en, pt-BR) e capítulo hands-on nas duas locales
- [x] T004 `make test` verde; TASK-013 com evidência

## Aceitação

- **SC-001** `task add X --body T` grava `T` como corpo; `task show X --json` expõe `body`.
- **SC-002** `TRILHA_LANG=pt-BR trilha-spec init` escreve `# Constituição`; sem a variável, `# Constitution`.
