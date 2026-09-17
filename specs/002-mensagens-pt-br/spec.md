# Spec 002 — Mensagens da CLI em pt-BR via `TRILHA_LANG`

- **Issue**: [#6](https://github.com/emersonjoe/trilha-spec/issues/6) — a issue é a fonte do escopo.
- **Task**: `TASK-007` em `.trilha/tasks/`.
- **Branch**: `main` (commit por tarefa)
- **Versão**: 0.2.0 da CLI; protocolo inalterado.

## Por quê

A documentação sai nas duas línguas e as specs são em pt-BR, mas quem opera a CLI em pt-BR lê
`created TASK-001`, `error: unknown spec command` e o `--help` em inglês. Hoje a pessoa
traduz de cabeça; não há mecanismo para a ferramenta falar a língua do operador.

## O que muda

- `TRILHA_LANG=pt` ou `pt-BR` (qualquer caixa, `-` ou `_`) traduz toda mensagem que a CLI
  produz: `--help`, resultados (`criado TASK-001`), erros de uso (`erro: uso: …`), cabeçalhos
  de listagem e os achados do `doctor`. Não definida ou desconhecida: inglês.
- Um catálogo por língua em `cmd/trilha-spec/i18n.go`, chaveado pelo texto em inglês
  (`T("created %s (%s)\n")`). Chave ausente cai para o inglês, nunca falha. Só biblioteca
  padrão.
- `spec.Layout.Doctor()` passa a responder `[]spec.Problem{Code, Arg}` em vez de strings, para
  que a CLI renderize cada achado na sua língua; `Problem.String()` é o inglês de sempre.
- Formatos de arquivo e `--json` não mudam: nomes de status, de campo e IDs continuam em
  inglês, porque são o protocolo, não mensagens.

```bash
TRILHA_LANG=pt-BR trilha-spec task list
# ID         STATUS   TÍTULO                                   AGUARDANDO
```

## Fora de escopo

- Descrições das ferramentas MCP: são protocolo, lidas por modelos.
- Erros de validação vindos dos pacotes (`task TASK-001: cannot move from …`) e os erros do
  pacote `flag`: continuam em inglês; traduzi-los exigiria i18n nos pacotes, o que a spec 001
  deixou de fora de propósito.
- Templates escritos em `.trilha/` (`init`, `spec new`): spec 003.

## Constitution Check

| Princípio | Como respeita |
|---|---|
| II — só biblioteca padrão | mapa de strings; sem gettext |
| V — teste primeiro | `TestDetectLang`, `TestTranslateFallsBack`, `TestCataloguesKeepVerbs`; e2e `TestPortugueseMessages` |
| Idioma | código e chaves em inglês; valores do catálogo `pt` em pt-BR |

## Tarefas

- [x] T001 Testes: detecção da língua, fallback, verbos preservados, e2e sob `TRILHA_LANG=pt-BR`
- [x] T002 `i18n.go` + `T()` em toda mensagem de `main.go`; `spec.Problem`
- [x] T003 README (en, pt-BR) e capítulo hands-on nas duas locales
- [x] T004 `make test` verde; TASK-007 com evidência

## Aceitação

- **SC-001** `TRILHA_LANG=pt-BR trilha-spec --help` sai em português; `TRILHA_LANG=fr` sai em inglês.
- **SC-002** `task list --json` sob `TRILHA_LANG=pt` é byte a byte o mesmo de sem a variável.
