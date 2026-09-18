# Spec 013 — Dependências entre repositórios e manifesto de programa

- **Issue**: [#11](https://github.com/emersonjoe/trilha-spec/issues/11) — a issue é a fonte do escopo.
- **Task**: `TASK-017` em `.trilha/tasks/`.
- **Versão**: 0.3 do protocolo (§3, §12 e a nova §13); campos aditivos e um arquivo opcional
  fora de `.trilha/`.
- **Depende de**: spec 010 (marcos, que o `program.md` compartilha).

## Por quê

`depends_on` era uma lista de ids locais. Um programa que atravessa repositórios — o app do
produto, o framework em que ele roda, o control plane, um gateway de IA — tem dependências do
tipo "a task X do produto precisa da task Y do framework (outro repo) em `done`". Hoje isso é
prosa no corpo: o `next` oferece trabalho que não pode começar e ninguém consegue desenhar o
grafo.

Contexto: o roadmap de um programa de setor público atravessa quatro repositórios — o app do
produto, o framework, o control plane e um gateway de IA — e as issues abertas a partir dele em
cada um dependem umas das outras.

## O que muda

- `project.md` ganha `repos: {alias: url}`. Alias são palavras minúsculas unidas por `-` ou
  `.`; a URL diz que repositório é aquele.
- `depends_on` aceita `<alias>:TASK-NNN`. Referência local continua exigindo que a task exista;
  a remota não entra na ordem topológica deste repositório, porque não é trabalho dele.
- `next`, `list`, `graph` e `doctor` aceitam `--repo alias=caminho` (repetível) para resolver a
  partir de um **checkout irmão**; um control plane resolve pela interface `task.Resolver` —
  **nada neste módulo abre socket**. Não resolvida, a dependência bloqueia a task com o motivo
  `waiting:<alias>:TASK-NNN`.
- `program.md` opcional, em um diretório acima dos checkouts, com `name`, `repos: {alias:
  caminho}` e `milestones` compartilhados (forma da §12). É encontrado subindo a partir do pai
  do projeto e resolve os aliases sozinho; `--repo` prevalece. `graph --program` desenha um
  subgrafo por repositório, e um repositório que ninguém baixou ainda aparece como aquilo que o
  trabalho está esperando.
- `doctor`: `repo-unknown` (dependência de um alias que `repos` não declara — falha).
- **Um projeto local não muda em nada**: sem `repos` e sem manifesto, nada disso aparece.

```bash
trilha-spec task list --repo trilha=../trilha
trilha-spec task next                  # o program.md acima resolve o alias sozinho
trilha-spec task graph --program
```

## Fora de escopo

- Buscar outros repositórios pela rede (clone, fetch, API): é papel do runner/control plane.
- Permissões entre repositórios (control plane).

## Constitution Check

| Princípio | Como respeita |
|---|---|
| I — o protocolo é o produto | §3, §12 e §13 descrevem alias, referência e resolução antes do código |
| II — só biblioteca padrão | resolução por leitura de arquivo; rede é interface, não implementação |
| VI — segurança por padrão | nada abre rede; um checkout irmão é lido, nunca escrito |
| V — teste primeiro | `TestRemoteDependencyBlocksUntilAnswered` e `TestProgramManifestAndGraph` com dois repositórios temporários |

## Tarefas

- [x] T001 `task.Ref`, `task.Resolver`, `task.Checkouts`, validação de `depends_on`
- [x] T002 `Project.Repos`, `spec.Program`, `Layout.FindProgram`, `Program.Checkouts`
- [x] T003 Grafo: referência remota fora da ordem topológica, `Blockers` com `waiting:`
- [x] T004 CLI `--repo`, `graph --program`, `doctor` com `repo-unknown`, i18n
- [x] T005 Protocolo §3/§12/§13 (en, pt-BR)

## Aceitação

- **SC-001** Sem resolução, a task fica bloqueada com `waiting:<alias>:TASK-NNN` e o `next` não a oferece.
- **SC-002** Com `--repo alias=caminho`, a dependência é respondida e a task começa quando o outro lado está `done`.
- **SC-003** `graph --program` desenha um subgrafo por repositório, com as arestas que cruzam.
- **SC-004** `doctor` falha com um alias que `project.md` não declara.
- **SC-005** Um projeto sem `repos` e sem `program.md` se comporta exatamente como na 0.2.
