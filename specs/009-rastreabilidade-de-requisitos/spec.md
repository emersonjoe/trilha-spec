# Spec 009 — Rastreabilidade de requisitos

- **Issue**: [#10](https://github.com/emersonjoe/trilha-spec/issues/10) — a issue é a fonte do escopo.
- **Task**: `TASK-016` em `.trilha/tasks/`.
- **Versão**: 0.3 do protocolo (§2, §3, §6, §8 e §11); campos aditivos e uma extensão da gramática.

## Por quê

Uma spec tem `issue`, os campos de segurança e prosa. Quando a fonte do escopo é um
**documento externo** — a lista de requisitos de um edital ("ficha do desafio 2, requisito 8:
informar o cidadão"), um artigo de lei, um KPI de contrato — nada liga as tasks a ele. "Quais
requisitos estão cobertos, por qual task, com que evidência" vira planilha mantida à mão, e é
exatamente a pergunta que um comprador público faz em todo marco.

Contexto: um programa de setor público em quatro repositórios (detalhe na issue); o edital
lista dez requisitos para um desafio e oito para o outro, e a proposta responde um a um.

## O que muda

### Gramática (§2)

O front matter ganha duas formas, porque `requirements` é uma lista de registros e não cabe
no subconjunto anterior:

- um **bloco** pode conter listas, não só escalares (`review:` da spec 012 usa isso);
- uma **lista de blocos**, `- chave: valor` seguido das demais chaves indentadas.

O aninhamento para aí: um bloco tem escalares e listas, nunca outro bloco. Um item `- ` que se
lê como `sub: valor` sem aspas abre um bloco; um item escalar com dois-pontos vai entre aspas,
que é como o escritor já o emitia. Uma lista nunca mistura escalares e blocos — isso é erro de
parse com o número da linha.

### Requisitos (§11) e cobertura (§3)

- Front matter da spec: `requirements: [{id, source, text}]`. `id` é identificador livre sem
  espaço nem vírgula (para `covers: [A, B]` nunca partir um ao meio), único no projeto.
- Front matter da task: `covers: [D2-R8, …]`.
- `doctor`: `requirement-unknown` (task cobre id que nenhuma spec declara — falha),
  `requirement-duplicate` (mesmo id em duas specs — falha), `requirement-uncovered`
  (requisito que nenhuma task cobre — **aviso**: o escopo está declarado, o trabalho ainda não
  foi cortado).
- `trilha-spec spec show <id> --coverage [--json]`: requisito → tasks → status → quantidade de
  evidência. Requisito sem task aparece com `—`.
- MCP: `trilha_coverage {spec?}` (leitura) responde a mesma matriz.
- Pacote de contexto (§6): o texto dos requisitos que a task cobre, ao lado do aceite.
- CLI: `task add --covers A,B`.

```bash
trilha-spec spec show 001-desafio-2 --coverage
# REQUISITO        TEXTO                                    TASKS
# D2-R8            informar o cidadao                       TASK-001 (done, 4 ev)
# D2-R9            prazo de 24h                             —
```

## Fora de escopo

- Importar requisitos de PDF ou de issue tracker (ferramenta em cima).
- Hierarquia de requisitos (um requisito que contém outros).

## Constitution Check

| Princípio | Como respeita |
|---|---|
| I — o protocolo é o produto | §2, §3 e §11 descrevem a gramática e os campos antes do código |
| II — só biblioteca padrão | a gramática cresceu no `spec.Parse`, sem YAML de terceiros |
| IV — determinismo | a matriz sai em ordem de spec e de declaração; o escritor é estável |
| V — teste primeiro | `TestBlockLists`, `TestRequirementRoundTrip`, `TestCoverageMatrix`, e2e |

## Tarefas

- [x] T001 Gramática: bloco com lista e lista de blocos em `spec.Parse`/`Doc.Bytes`
- [x] T002 `spec.Requirement`, `Spec.Requirements`, `Task.Covers`, validação
- [x] T003 `task.Cover`, `task.CheckRequirements`, códigos de problema e i18n
- [x] T004 CLI `spec show --coverage`, `task add --covers`, MCP `trilha_coverage`, pacote de contexto
- [x] T005 Protocolo §2/§3/§6/§8/§11 (en, pt-BR)

## Aceitação

- **SC-001** Uma spec com `requirements` sobrevive a um round trip byte a byte.
- **SC-002** `spec show <id> --coverage` mostra a task que cobre cada requisito, com a contagem de evidência.
- **SC-003** `doctor` falha com `covers` apontando para id inexistente e avisa sobre requisito sem task.
- **SC-004** O pacote de contexto traz o texto do requisito que a task cobre.
