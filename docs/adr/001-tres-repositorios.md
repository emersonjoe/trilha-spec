# ADR 001 — Três repositórios: spec público, runner público, cloud privado

- **Data**: 2026-09-12
- **Status**: aceito
- **Contexto**: a estratégia de transformar o Trilha de "framework Go com recursos de IA" em
  "infraestrutura open source para desenvolvimento orientado por agentes".

## Decisão

```
PUBLIC                                   PRIVATE
═══════════════════════════════════      ═══════════════════════════════
trilha-spec     o que fazer              trilha-cloud   control plane
  │  protocolo: specs, tasks, grafo,       organizações, times, billing,
  │  contexto, evidência, MCP              fila remota, workers, sandboxes,
  ▼                                        fleet, governança, auditoria
trilha-runner   como executar
     agentes locais, worktrees,
     verificação, evidência, fila
     local + cliente da fila remota
```

O runner **fica público** (a ressalva da proposta é aceita): a diferença entre "rodar um
agente num worktree local" e "rodar mil agentes em sandboxes de uma frota" não está no runner,
está no control plane. O runner público é o que faz o padrão valer — Claude Code, Codex ou
outro runner podem implementar o mesmo protocolo —, e a fronteira comercial fica onde há
custo de operação de verdade: fila, workers, sandbox, organização, auditoria.

## Análise: a CLI vai ter conflito?

**Sim, em três pontos, e os três têm solução barata.** Levantamento feito sobre
`emersonjoe/trilha` v0.123.0 (`cmd/trilha/main.go`).

### 1. Nome do binário

`go install github.com/emersonjoe/trilha/cmd/trilha` e um hipotético
`go install github.com/emersonjoe/trilha-spec/cmd/trilha` instalam **o mesmo arquivo**
`$GOBIN/trilha`: o segundo apaga o primeiro. Um usuário do framework que instalasse o protocolo
perderia `trilha dev`.

**Decisão**: o binário do protocolo chama-se `trilha-spec`; o do runner, `trilha-runner`.
Nenhum dos dois é `trilha`.

### 2. Tabela de comandos

| Proposta (Fase 0) | Já existe em `trilha` | Conflito |
|---|---|---|
| `trilha init` | `trilha new` | semântico: `new` cria app; `init` criaria `.trilha/`. Confuso lado a lado |
| `trilha spec` | — | livre |
| `trilha task` | — (pacote `task/` é fila in-process do framework) | nome de pacote colide se o protocolo importasse o framework; binários separados evitam |
| `trilha agent` | `trilha agents` (escreve `AGENTS.md`/`CLAUDE.md`) | **direto**: singular × plural com significados diferentes |
| `trilha verify` | `trilha check` (gen/gofmt/vet/test/audit/openapi) | sobreposição: `check` verifica o app; `verify` verificaria uma task |
| `trilha mcp` (protocolo) | `trilha mcp` (ferramentas do framework: check, routes, generate) | **direto** |
| context | `trilha ctx` (mapa do projeto para máquinas) | semântico |

**Decisão**: com binários separados, nada colide. Para que a experiência seja `trilha spec …`
(uma marca, uma CLI), o framework adota **despacho de subcomando externo no estilo git**: no
`default:` do `switch` em `cmd/trilha/main.go`, se existir `trilha-<cmd>` no `PATH`, executa
com os argumentos restantes e devolve o código de saída. Assim:

```
trilha spec task next     →  trilha-spec task next
trilha runner run TASK-1  →  trilha-runner run TASK-1
```

São ~15 linhas em `main.go` mais uma linha na mensagem de uso. O `trilha mcp` do framework
continua sendo o MCP do framework; o do protocolo é `trilha spec mcp`. Um host MCP pode
registrar os dois. Patch sugerido (não aplicado aqui — é do repositório `trilha`):

```go
default:
    if ext, err := exec.LookPath("trilha-" + os.Args[1]); err == nil {
        c := exec.Command(ext, os.Args[2:]...)
        c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
        if err := c.Run(); err != nil {
            var ee *exec.ExitError
            if errors.As(err, &ee) { os.Exit(ee.ExitCode()) }
            fatal(err)
        }
        return
    }
    fmt.Fprintf(os.Stderr, t("unknown command"), os.Args[1], t("usage"))
    os.Exit(2)
```

### 3. O diretório `.trilha/` — o conflito que ninguém veria

O framework **já usa `.trilha/`** como cache de build: `internal/dev/server.go` compila o app
em `.trilha/app`, `export.go` em `.trilha/export-app`, e o dev server grava
`.trilha/.gitignore` com conteúdo `*` a cada subida. O `trilha audit` ainda exige que
`.trilha` esteja no `.gitignore` da raiz, e o scaffold de `trilha new` já gera `.trilha/` no
`.gitignore`.

Resultado: num app Trilha, `trilha-spec init` criaria specs, tasks e evidência dentro de um
diretório que o próprio framework manda o git ignorar — e o `trilha dev` reescreveria o
`.gitignore` para `*` na primeira execução.

**Decisão**: o protocolo mantém `.trilha/` (é a marca e é o que a proposta pede) e resolve
em duas frentes:

1. **Aqui**: `spec.Init` grava `.trilha/.gitignore` ignorando só `runs/`, `cache/`, `app`,
   `app.exe`, `export-app*`; `trilha-spec doctor` acusa um `.gitignore` com `*`.
2. **No framework** (patch a fazer em `trilha`): mover o cache para `.trilha/cache/`
   (`internal/dev/server.go`, `export.go`, `internal/dev/watch.go`), gravar
   `.trilha/cache/.gitignore` em vez de `.trilha/.gitignore`, trocar a linha `.trilha/` do
   scaffold e do `audit` por `.trilha/cache/` e `.trilha/runs/`. É uma spec curta no
   repositório `trilha`; até ela sair, quem usa os dois roda `trilha-spec doctor` depois de
   `trilha dev`.

## Dependências entre módulos

```
trilha-spec    ── stdlib apenas
trilha-runner  ── trilha-spec + github.com/emersonjoe/trilha (só os pacotes ai, ai/mcp)
trilha-cloud   ── trilha-spec + github.com/emersonjoe/trilha (é um app Trilha)
```

O protocolo não depende do framework: uma ferramenta que não é Go precisa ler `.trilha/`
sem saber o que é Trilha. O runner reaproveita o `ai` do framework (cliente do protocolo
OpenAI, loop de agente, cliente MCP) em vez de reescrevê-lo. O cloud é um app Trilha —
roteamento por arquivos, `task` in-process como fila da Fase 0, `audit`, `Limit` — e por isso
é também o maior teste de uso do framework.

## O que não é decidido aqui

- Versão 1.0 do protocolo; esta é 0.1.
- Assinatura de evidência (cadeia de custódia entre runner e cloud).
- Se `trilha-cloud` vira open core no futuro.
