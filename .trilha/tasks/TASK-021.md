---
id: TASK-021
title: "RM-002: release por tag com binários e checksums"
status: ready
spec: 015-fase-0-do-roadmap
agent: coder
depends_on: []
acceptance:
  - "release.yml em tag v* compila linux/amd64, linux/arm64, darwin/amd64 e darwin/arm64 com -trimpath -ldflags '-s -w -X main.version=<tag>' e publica sha256sums.txt"
  - TestVersionMatchesReleaseTag falha quando RELEASE_TAG difere de version
  - README EN e pt-BR mostram o caminho por binário e mantêm go install
checks:
  - make test
  - RELEASE_TAG=v0.3.0 go test ./cmd/trilha-spec -run TestVersionMatchesReleaseTag
created: "2026-10-21T02:10:09Z"
---

Roadmap RM-002. A tag v0.3.0 é cortada por um humano depois do merge; o workflow só reage à tag.
