# Spec 005 — Impacto de segurança e comandos de evidência na spec

- **Issue**: [#1](https://github.com/emersonjoe/trilha-spec/issues/1) — a issue é a fonte do escopo.
- **Task**: `TASK-009` em `.trilha/tasks/`.
- **Versão**: 0.2 do protocolo (§11 ganha campos); arquivos existentes continuam válidos.

## Por quê

Uma spec só tinha `id`, `title`, `status`, `issue` e as relações. A constituição do
trilha-cloud exige em toda spec ativos, fronteiras de confiança, controles afetados e os
comandos de evidência que o revisor precisa ver — e isso vivia em seções livres
(`## Security impact`, `## Evidence`) que o `doctor` e o pacote de contexto ignoravam.
Descoberto ao escrever a spec 011 do trilha-cloud com o `trilha-spec` no lugar do spec-kit.

## O que muda

- Front matter de spec ganha `assets`, `trust_boundaries`, `controls` (identificadores
  livres) e `evidence` (comandos: programa e argumentos, sem shell, como `checks` de task).
  Listas de escalares; leitor que não conhece as chaves as preserva.
- `spec.Spec` expõe os quatro campos (JSON: `assets`, `trust_boundaries`, `controls`,
  `evidence`); `HasSecurityImpact()`; item vazio é erro de validação. `Bytes` escreve as
  chaves em ordem fixa.
- `doctor`: spec `approved` sem nenhum dos quatro é **aviso** (`spec.Problem.Warning()`,
  código `spec-no-security`), impresso com `!` e contado; não falha o comando.
- Pacote de contexto: `### Assets touched`, `### Trust boundaries`, `### Controls affected`,
  `### Evidence the reviewer must see`, logo após a spec e antes da task; o JSON já carrega
  os campos dentro de `spec`.
- CLI: `spec new` e `spec set` ganham `--asset`, `--boundary`, `--control`, `--evidence`
  (repetíveis; em `set`, a lista dada substitui a existente).
- Fora de escopo, por decisão: qualquer julgamento sobre os controles declarados.

```bash
trilha-spec spec set 001-login --asset "cookie de sessão" --boundary "browser → api" \
  --control "ASVS V3.4" --evidence "go test ./internal/auth/..."
trilha-spec doctor      # "! spec 002-x is approved but declares no security impact"
```

## Fora de escopo

- Rodar os comandos de `evidence` (isso é `verify` de task; a spec só os declara).
- Mudança no ciclo de vida de task.

## Constitution Check

| Princípio | Como respeita |
|---|---|
| I — o protocolo é o produto | §11 descreve os campos antes do código |
| II — Markdown + front matter | quatro listas de escalares; chaves desconhecidas preservadas |
| V — teste primeiro | `TestSpecSecurityImpact`, `TestBuild` (pacote), e2e `TestSpecSecurityCLI` |

## Tarefas

- [x] T001 Testes: round trip, validação, aviso do doctor, pacote de contexto, CLI
- [x] T002 Campos em `spec.Spec`, `CheckSpecs`, flags, `doctor` com avisos
- [x] T003 Protocolo §11 (en, pt-BR), README (en, pt-BR), capítulo hands-on nas duas locales
- [x] T004 `make test` verde; TASK-009 com evidência

## Aceitação

- **SC-001** `spec set X --evidence CMD` grava `evidence:` e `spec show X --json` expõe.
- **SC-002** `doctor` avisa spec `approved` sem impacto e sai com 0.
- **SC-003** `context TASK` mostra os quatro campos ao lado do aceite.
