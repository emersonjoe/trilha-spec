---
id: TASK-019
title: Documentar taxonomia de failure_class/error_code
status: done
spec: 014-taxonomia-de-failure-class
depends_on: []
acceptance:
  - docs/protocol.md e docs/pt-BR/protocol.md documentam um vocabulário mínimo para failure_class, incluindo provider_transient
  - contracts/execution/v1/schema.json e execution/testdata/run-v1.json continuam validando sem mudança
  - make test passa
checks: []
created: "2026-09-19T15:54:29Z"
updated: "2026-09-19T15:55:10Z"
---

Issue: https://github.com/emersonjoe/trilha-spec/issues/15 — a issue é a fonte do escopo.

Observado ao vivo no projeto acervo (run-000007, TASK-011): três tentativas de "sh -c 'cd api && python -m pytest -q'" falhando com exit 127 por falta de python no worker, todas reportadas como checks_failed sem distinção.
