# Spec 006 — Limites e pausa do projeto

- **Issue**: [#3](https://github.com/emersonjoe/trilha-spec/issues/3) — a issue é a fonte do escopo.
- **Task**: `TASK-011` em `.trilha/tasks/`.
- **Versão**: 0.2 do protocolo (§2 ganha mapas de um nível; §12 novo).

## Por quê

`project.md` só tinha `name`, `description`, `default_agent` e `verify[]`. Uma task tem
`token_budget`, mas o projeto não tem **limites** nem estado **pausado**; cada control plane
inventa o próprio disjuntor e o runner não lê os mesmos limiares antes de iniciar uma task.
É o que a spec 011 do trilha-cloud (disjuntor de projeto e pausa da frota) precisa.

## O que muda

- Gramática de front matter (§2): além de escalar e lista, um **mapa de escalares de um
  nível** (`chave:` seguida de linhas `  sub: valor`; `chave: {}` é um mapa vazio).
  `Fields.SetMap/GetMap`; `Map()` devolve `map[string]string`; ordem de chaves preservada.
- `project.md` ganha `limits` (mapa numérico), `paused`, `pause_reason`, `paused_at` (RFC
  3339). Chaves nomeadas: `max_cost_per_hour`, `max_failure_rate` (0..1),
  `max_repeated_failure_class`; qualquer outra é carregada. Todo leitor pode ignorar.
- `spec.Project` expõe `Limits map[string]float64`, `Paused`, `PauseReason`, `PausedAt`;
  `ParseProject`, `Validate`, `Pause`, `Resume`, `Bytes`; `Layout.SaveProject` preserva
  campos desconhecidos e o corpo.
- CLI: `project show [--json]`, `project pause [--reason R]`, `project resume`,
  `project limit <chave> <valor|->`. `task next` pausado não lista nada e diz por quê
  (`--json` segue lista vazia; motivo em stderr).
- MCP `trilha_next` responde erro com o motivo quando pausado.
- Pacote de contexto: `### Limits` após o projeto e um aviso de pausa.

```bash
trilha-spec project limit max_cost_per_hour 5
trilha-spec project pause --reason "breaker:max_cost_per_hour"
trilha-spec task next     # project is paused: breaker:max_cost_per_hour (since …)
```

## Fora de escopo

- Fazer valer os limites (parar tentativa, disparar o disjuntor): runner e control plane.
- Limites por chave ou por agente; conversão de moeda.

## Constitution Check

| Princípio | Como respeita |
|---|---|
| I — o protocolo é o produto | §2 e §12 escritos antes do código |
| II — Markdown + front matter | mapa de um nível continua legível sem YAML |
| V — teste primeiro | `TestMapFields`, `TestProjectLimitsAndPause`, `TestBuild`, e2e `TestProjectLimitsAndPauseCLI`, MCP |

## Tarefas

- [x] T001 Testes: gramática de mapa, round trip do projeto, next pausado, MCP, pacote
- [x] T002 Parser/escritor, `spec.Project`, comandos `project`, `task next`, `trilha_next`
- [x] T003 Protocolo §2/§12 (en, pt-BR), README (en, pt-BR), capítulo hands-on nas duas locales
- [x] T004 `make test` verde; TASK-011 com evidência

## Aceitação

- **SC-001** `project limit max_cost_per_hour 5` grava `limits:\n  max_cost_per_hour: 5`.
- **SC-002** `project pause --reason R` faz `task next` responder `project is paused: R`.
- **SC-003** `context TASK` mostra `### Limits`.
