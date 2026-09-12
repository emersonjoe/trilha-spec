# trilha-spec Constitution

`trilha-spec` é o protocolo aberto do Trilha para trabalho executável por agentes: o formato de
`.trilha/`, o modelo de task, o grafo, o contexto e a evidência. Este documento fixa os
princípios que toda feature deve obedecer.

## Core Principles

### I. O protocolo é o produto (NON-NEGOTIABLE)
O que está em `docs/protocol.md` é a verdade; o código Go é uma implementação de referência.
Uma mudança de formato ou de transição de status é uma spec com versão do protocolo, nunca um
patch. Um arquivo em `.trilha/` precisa ser lido por outra ferramenta sem este módulo:
Markdown + front matter para o que pessoas editam, JSON para o que máquinas gravam.

### II. Só biblioteca padrão
O módulo depende apenas da biblioteca padrão do Go. Um parser de YAML completo não entra;
o subconjunto de front matter é o do `spec.Parse` e cresce por spec.

### III. Evidência verificável
Toda transição que fecha trabalho (`verify → review → done`) apoia-se em registros em
`evidence/`: comando, código de saída, hash da saída, autor, instante. Nenhum comando roda em
shell implícito; `SplitCommand` divide, e quem precisa de shell escreve `sh -c`.

### IV. Determinismo
Listagens, grafo e escrita de documentos são determinísticos: mesma entrada, mesmos bytes.
Ordem de campos fixa, IDs sequenciais, ordenação por ID.

### V. Teste primeiro
Parser, transições, grafo, evidência e CLI têm teste antes da implementação (`go test`, sem
framework). Nenhuma feature é "pronta" sem `make test` verde.

### VI. Segurança por padrão
O servidor MCP é só leitura sem `--write`; ferramenta não oferecida não existe. Checks têm
timeout e limite de saída. Nada abre rede.

## Idioma

Código, identificadores, mensagens e documentação pública em inglês; `README.pt-BR.md` e
`docs/pt-BR/` no mesmo commit. Specs, ADRs e esta constituição em português do Brasil.

## Fluxo de trabalho

Toda mudança começa por uma spec em `specs/NNN-nome/` (spec-kit) e, quando muda o protocolo,
por uma task em `.trilha/tasks/` do próprio repositório — o projeto usa o que publica.
Mudança pequena usa a spec curta. Commits pequenos por tarefa; `gofmt` e `go vet` limpos.

**O capítulo hands-on acompanha o código.** O site do Trilha ensina esta ferramenta em
`/learn/agentic-protocol` (pt: `/pt/aprender/agentico-protocolo`). Toda mudança no que o usuário digita ou vê — comando, flag, formato de
arquivo, saída — atualiza o capítulo nas duas locales (`site/internal/docs/content/en/learn/agentic-protocol.md` e `site/internal/docs/content/pt/aprender/agentico-protocolo.md` no repositório
`emersonjoe/trilha`) na mesma sessão, antes de a spec fechar; a tarefa entra na lista da spec.

## Governance

Esta constituição prevalece sobre qualquer outra prática do repositório. Emendas exigem nova
versão aqui e atualização dos templates em `.specify/templates/`.

**Version**: 1.0.0 | **Ratified**: 2026-09-12 | **Last Amended**: 2026-09-12
