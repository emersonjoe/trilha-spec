# Spec 007 — Custo padronizado na evidência `run`

- **Issue**: [#4](https://github.com/emersonjoe/trilha-spec/issues/4) — a issue é a fonte do escopo.
- **Task**: `TASK-012` em `.trilha/tasks/`.
- **Versão**: 0.2 do protocolo (§5 e §10); contrato de execução v1 com campos aditivos.

## Por quê

A evidência `run` tinha só `meta{}` livre. Um runner escrevia `tokens`, outro `usage.total`,
um terceiro nada; um control plane com ledger não conciliava o que os runners declararam
com a fatura do provedor, e um `cost` auto-declarado não podia ser cruzado.

## O que muda

- `task.Evidence` ganha `provider`, `model`, `tokens_in`, `tokens_out`, `cost`, `currency`
  (opcionais, `omitempty`). Regras: tokens e custo nunca negativos; `cost` exige `currency`;
  `currency` é ISO 4217 (`^[A-Z]{3}$`). `meta{}` continua livre.
- `execution.Attempt` e `execution.Evidence` ganham os mesmos nomes (aditivo; `model`,
  `total_tokens` e `estimated_cost` permanecem). Schema `contracts/execution/v1/schema.json`
  e fixture `execution/testdata/run-v1.json` atualizados.
- CLI: `evidence add --run [--provider P] [--model M] [--tokens-in N] [--tokens-out N]
  [--cost C --currency USD] [--failed]`; a listagem resume o custo; `--json` expõe tudo;
  o pacote de contexto mostra `modelo custo moeda` na evidência.
- `cost` é o que o runner observou; o protocolo não afirma verificação.

```bash
trilha-spec evidence TASK-001 add --run --provider anthropic --model claude-sonnet-5 \
  --tokens-in 12345 --tokens-out 678 --cost 0.0421 --currency USD
```

## Fora de escopo

- Preço por modelo dentro do protocolo; conversão de moeda.
- Conciliação com a API de billing do provedor (control plane).

## Constitution Check

| Princípio | Como respeita |
|---|---|
| I — o protocolo é o produto | §5/§10 e o schema mudam antes do código |
| II — Markdown + front matter / JSON | evidência continua JSON, campos aditivos |
| V — teste primeiro | `TestRunEvidenceCost`, `TestCanonicalFixture`, e2e `TestRunEvidenceCLI` |

## Tarefas

- [x] T001 Testes: gravação e validação, fixture do contrato, CLI e pacote
- [x] T002 Campos em `task.Evidence` e `execution.Attempt`/`Evidence`; schema; `evidence add --run`
- [x] T003 Protocolo §5/§10 (en, pt-BR), README (en, pt-BR), capítulo hands-on nas duas locales
- [x] T004 `make test` verde; TASK-012 com evidência `run`

## Aceitação

- **SC-001** `evidence add --run --cost 1` sem `--currency` falha.
- **SC-002** `evidence <task> --json` expõe `provider`, `model`, `tokens_in`, `tokens_out`, `cost`, `currency`.
- **SC-003** O fixture `run-v1.json` valida com os campos na Attempt.
