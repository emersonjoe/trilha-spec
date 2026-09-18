---
id: 010-marcos-com-prazo
title: Marcos com prazo
status: done
issue: 12
depends_on:
  - 009-rastreabilidade-de-requisitos
assets:
  - front matter de project, spec e task
  - ordem da fila (`next`, `trilha_next`)
trust_boundaries:
  - repositório → comprador/auditor (cronograma físico-financeiro)
controls:
  - data validada como AAAA-MM-DD; marco não declarado é falha do doctor
evidence:
  - go test ./task/... -run TestMilestone
  - go test ./cmd/... -run TestMilestonesCLI
---

Ver `specs/010-marcos-com-prazo/spec.md` (issue #12). `project.md` ganha
`milestones: [{id, title, due, gate}]`; spec e task ganham `milestone`, que a task herda da
spec quando não declara o próprio. `task list --milestone`, ordem do `next` pelo prazo mais
próximo, `project milestone` na CLI, marco e prazo no pacote de contexto, e três checagens
novas no `doctor`.
