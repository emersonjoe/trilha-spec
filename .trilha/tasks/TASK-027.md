---
id: TASK-027
title: "RM-054: resolver remoto de dependências"
status: idea
depends_on: []
acceptance: []
checks: []
created: "2026-10-21T02:10:09Z"
---

Roadmap trilha-cloud/docs/ROADMAP-STACK.md, RM-054. task/remote.go declara task.Resolver sem implementação remota; o runner resolve por conta própria em queue.Remote.DependencyStatus. Entrega: queue.Remote implementa task.Resolver e é injetado em next/run; sem cloud, cai no --repo alias=path. Um só caminho de resolução, testado contra a interface.
