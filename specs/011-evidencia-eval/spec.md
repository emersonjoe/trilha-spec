# Spec 011 — Evidência `eval`: métricas com limiar

- **Issue**: [#8](https://github.com/emersonjoe/trilha-spec/issues/8) — a issue é a fonte do escopo.
- **Task**: `TASK-014` em `.trilha/tasks/`.
- **Versão**: 0.3 do protocolo (§3, §5 e §6); tipo novo de evidência e campos aditivos.

## Por quê

Os tipos de evidência eram `check`, `note`, `artifact` e `run`. Um programa de produto mede
qualidade com números — acurácia de triagem ≥ 0,85 num conjunto rotulado, nota humana de
tradução ≥ 4/5, latência p95 ≤ 5 s, violações WCAG = 0 — produzidos por um harness. O único
jeito de registrar isso era um `check` cujo código de saída **esconde o número**: o revisor não
vê o valor, nada consegue acompanhá-lo entre execuções e um control plane não consegue barrar
nada com ele.

Contexto: um programa de setor público com golden sets precisa de cinco portões desses por
produto (detalhe na issue).

## O que muda

- `kind: "eval"` com `metric`, `value`, `unit` (opcional), `threshold`, `comparator`
  (`>=`, `<=`, `==`), `dataset` (`{id, sha256}` do **manifesto** do golden set, nunca do
  conteúdo — conjuntos carregam dado pessoal) e `passed`.
- `metric` são palavras minúsculas unidas por `_`, `-` ou `.`. `comparator` é **obrigatório**
  quando há `threshold`: comparador adivinhado é portão errado. Sem threshold é medição, não
  portão, e passa. `value` e `threshold` são ponteiros no Go porque **zero é um número que
  interessa a um portão** ("violações WCAG == 0").
- `verify` grava um `eval` por linha JSON que um check imprime no stdout na forma documentada
  (`{"metric":"triage_top1","value":0.87,"threshold":0.85,"comparator":">="}`), para que um
  harness em qualquer linguagem a emita com um `echo`; check que imprime texto comum não muda.
  **Um `eval` reprovado reprova o `verify` mesmo com código de saída 0** — é para isso que
  serve um portão. Linha com threshold sem comparador vira medição com o motivo em `note`.
- `acceptance` da task aceita a forma métrica (`metric: triage_top1 >= 0.85`); `doctor` aponta
  métrica de aceite sem nenhum `eval` (aviso) a partir de `review`.
- `evidence` (lista e `--json`), `evidence add --eval`, pacote de contexto (último valor por
  métrica) e MCP `trilha_list_evidence` expõem os registros.
- Os registros e todo `--json` passam a ser escritos **sem escape de HTML**: um comparador se
  lê `>=`, não `>=`. A forma canônica da assinatura já era assim.

```bash
# o harness imprime; o verify grava
echo '{"metric":"triage_top1","value":0.87,"threshold":0.85,"comparator":">="}'
trilha-spec evidence TASK-001 add --eval --metric p95_latency --value 7 --unit s \
  --threshold 5 --comparator '<='
```

## Fora de escopo

- Guardar datasets ou histórico de métricas no protocolo (é papel do control plane).
- Testes estatísticos, intervalos de confiança, comparação entre execuções.

## Constitution Check

| Princípio | Como respeita |
|---|---|
| I — o protocolo é o produto | §5 define o registro, o comparador e a regra do portão antes do código |
| II — só biblioteca padrão | `encoding/json`, `regexp`, `strconv` |
| III — evidência verificável | o número fica no registro, assinável como qualquer outro |
| V — teste primeiro | `TestEvalRecordAndGate`, `TestParseMetricLine`, `TestVerifyRecordsMetrics`, e2e |

## Tarefas

- [x] T001 Campos de `eval` na evidência, validação, `Eval`, `Compare`, `Metrics`
- [x] T002 `ParseMetricLine` e integração no `RunChecks` (um `eval` por linha, portão reprova)
- [x] T003 `metric:` no aceite, `task.CheckMetrics`, código de problema e i18n
- [x] T004 CLI `evidence add --eval`, coluna na listagem, pacote de contexto
- [x] T005 JSON sem escape de HTML em registro, `--json` da CLI, pacote e MCP
- [x] T006 Protocolo §3/§5/§6 (en, pt-BR)

## Aceitação

- **SC-001** Um check que imprime a linha JSON produz um registro `eval` com valor e limiar.
- **SC-002** Um `eval` abaixo do limiar reprova o `verify` mesmo com código de saída 0.
- **SC-003** `value: 0` e `threshold: 0` sobrevivem ao round trip.
- **SC-004** `doctor` aponta métrica de aceite sem evidência em uma task em `review`.
- **SC-005** Um `eval` assinado responde `valid`, e o mesmo registro editado responde `invalid`.
