---
id: 013-dependencias-entre-repositorios
title: Dependências entre repositórios e manifesto de programa
status: done
issue: 11
depends_on:
  - 010-marcos-com-prazo
assets:
  - front matter de project e task (repos, depends_on)
  - program.md fora de .trilha, acima dos checkouts
  - checkouts irmãos lidos por --repo
controls:
  - nada abre rede; control plane entra por interface, não por cliente HTTP
  - checkout irmão é lido, nunca escrito
  - alias validado; dependência de alias não declarado é falha do doctor
trust_boundaries:
  - repositório → repositório irmão (leitura de arquivo)
  - repositório → control plane (interface task.Resolver)
evidence:
  - go test ./task/... -run TestRemote
  - go test ./task/... -run TestProgramManifestAndGraph
  - go test ./cmd/... -run TestCrossRepositoryCLI
---

Ver `specs/013-dependencias-entre-repositorios/spec.md` (issue #11). `project.md` ganha
`repos: {alias: url}` e `depends_on` aceita `<alias>:TASK-NNN`; sem resolução a task fica
bloqueada com `waiting:<alias>:TASK-NNN`. `--repo alias=caminho` resolve por checkout irmão e
`task.Resolver` deixa o control plane responder sem que este módulo abra rede. `program.md`
opcional acima dos checkouts traz repositórios e marcos compartilhados, lido por
`graph --program`.
