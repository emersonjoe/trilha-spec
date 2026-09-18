# Spec 010 — Marcos com prazo

- **Issue**: [#12](https://github.com/emersonjoe/trilha-spec/issues/12) — a issue é a fonte do escopo.
- **Task**: `TASK-018` em `.trilha/tasks/`.
- **Versão**: 0.3 do protocolo (§3, §4, §6, §8, §11 e §12); campos aditivos.
- **Depende de**: spec 009 (lista de blocos no front matter).

## Por quê

O protocolo tem status e dependências, mas não tem calendário. Um contrato com marcos pagos
(M0…M4 em doze meses) e um control plane com rodadas não conseguem dizer "esta task é do M2,
com prazo 2027-04-30", e o cronograma físico-financeiro que um comprador público exige acaba
morando fora do protocolo.

Contexto: um programa de setor público com fases e marcos de pagamento em contrato (detalhe
na issue).

## O que muda

- `project.md`: `milestones: [{id, title, due, gate}]`. `id` segue a regra do id de requisito;
  `due` é dia de calendário `AAAA-MM-DD`, não instante — um marco vence num dia.
- Front matter de spec e de task: `milestone: M2`. Task sem o próprio **herda o da spec**, como
  uma task sem `agent` recebe o `default_agent`.
- `task list --milestone M2` (e `--json`); `trilha_list_tasks {milestone}`.
- `next` prefere o prazo mais próximo entre o que pode rodar — ordenação estável, então a ordem
  de dependência sobrevive ao empate e a task sem prazo vai por último. Vale para a CLI e para
  `trilha_next`.
- `doctor`: `milestone-past-due` (task não concluída com prazo vencido — aviso),
  `milestone-empty` (marco sem task — aviso), `milestone-unknown` (task ou spec nomeando marco
  não declarado — **falha**, porque um erro de digitação tira trabalho de todo relatório).
- Pacote de contexto (§6): o marco, o prazo e o gate na primeira linha; `--json` com o bloco.
- CLI: `project milestone <id> [--title T] [--due AAAA-MM-DD] [--gate G]` e `<id> -` para
  remover; `task add --milestone M2`.

```bash
trilha-spec project milestone M2 --title "PoC entregue" --due 2027-04-30 --gate "aceite do cliente"
trilha-spec task add "Painel" --milestone M2 --status ready --accept "mostra o andamento"
trilha-spec task next     # o prazo mais próximo primeiro
```

## Fora de escopo

- Estimativa, calendário de trabalho, capacidade (control plane).
- Quanto vale um marco e se ele foi cumprido: decisão de pessoa.

## Constitution Check

| Princípio | Como respeita |
|---|---|
| I — o protocolo é o produto | §3, §4, §11 e §12 descrevem os campos e a ordenação antes do código |
| II — só biblioteca padrão | `time.Parse` para a data; nada mais |
| IV — determinismo | `ByDue` é `sort.SliceStable`: mesma entrada, mesma ordem |
| V — teste primeiro | `TestMilestoneRoundTrip`, `TestMilestoneSchedulingAndDoctor`, e2e |

## Tarefas

- [x] T001 `spec.Milestone`, `Project.Milestones`, `Spec.Milestone`, `Task.Milestone`, validação
- [x] T002 `task.MilestoneOf`, `task.ByDue`, `task.CheckMilestones`, códigos de problema e i18n
- [x] T003 CLI `project milestone`, `task add --milestone`, `task list --milestone`, ordem do `next`
- [x] T004 MCP `trilha_list_tasks {milestone}` e ordem do `trilha_next`; pacote de contexto
- [x] T005 Protocolo §3/§4/§6/§8/§11/§12 (en, pt-BR)

## Aceitação

- **SC-001** `project.md` com `milestones` sobrevive a um round trip byte a byte; `due` inválido é recusado.
- **SC-002** `task next` lista o prazo mais próximo primeiro e a task sem prazo por último.
- **SC-003** `task list --milestone M2` traz só as tasks do M2, inclusive as que herdam pela spec.
- **SC-004** `doctor` avisa sobre prazo vencido e marco vazio, e falha com marco não declarado.
