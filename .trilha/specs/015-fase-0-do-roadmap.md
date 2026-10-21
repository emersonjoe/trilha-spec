---
id: 015-fase-0-do-roadmap
title: "Fase 0 do roadmap da stack: CI, release por tag e fonte da verdade"
status: approved
assets:
  - workflows de CI e release (.github/workflows)
  - binários publicados em releases do GitHub e seus checksums
  - .trilha/ como fonte da verdade de specs e tasks
  - fixtures do contrato de execução comparados com os repositórios irmãos
trust_boundaries:
  - GitHub Actions → GitHub Releases (GITHUB_TOKEN com contents:write só no release)
  - trilha-spec → checkout de trilha-runner (leitura de fixture, nunca escrita)
  - usuário → binário baixado (verificado por sha256sums.txt)
controls:
  - ASVS V14.2 — ações pinadas por SHA de commit, permissões mínimas no workflow
  - ASVS V10.3 — build reproduzível (-trimpath, -s -w) e checksums publicados
  - ASVS V1.14 — a versão do binário é a da tag, verificada por teste antes de publicar
evidence:
  - make test
  - make conformance
  - go test ./cmd/trilha-spec -run TestVersionMatchesReleaseTag
  - go test ./spec -run TestCheckSpecStatus
---

# Fase 0 do roadmap da stack: CI, release por tag e fonte da verdade

- **Roadmap**: `trilha-cloud/docs/ROADMAP-STACK.md`, itens RM-007, RM-002, RM-003 (parte do
  spec) e RM-004.
- **Branch**: `rm-fase0`
- **Versão**: 0.3.0 do protocolo; nenhum campo ou transição muda.

## Por quê

Os quatro repositórios da stack divergem em coisas pequenas que custam confiança: o CI do
protocolo não formata `execution/`; o gate de conformidade do contrato de execução existe mas
nenhum CI o roda; não há tag nem binário, então quem não tem Go não instala; e um modelo menor
encontra duas verdades para a mesma feature (Spec Kit em `specs/NNN/` e `.trilha/specs/NNN.md`)
e escolhe a errada. Esta spec fecha a parte do `trilha-spec` da Fase 0 do roadmap.

## O que muda

- **RM-007** — `make vet` e o CI rodam `gofmt -l .` (sem lista de pacotes). O job `conformance`
  faz checkout de `emersonjoe/trilha-runner` (`ref: main`) como irmão e roda
  `scripts/check-execution-contract.sh`; o script diz `skipped: <repo> not present` para o irmão
  que faltar e só sai 0 quando todo irmão presente bate. `trilha-cloud` é privado e fica fora do
  job; roda em `make conformance` local.
- **RM-002** — `release.yml` em tag `v*`: `TestVersionMatchesReleaseTag` compara `version` com a
  tag; build para linux/amd64, linux/arm64, darwin/amd64 e darwin/arm64 com
  `-trimpath -ldflags "-s -w -X main.version=<tag>"`; `sha256sums.txt`; `gh release create`
  com os assets e as notas de `docs/roadmap.md`. `version` passa de `const` a `var` para o
  ldflags sobrescrever. README (EN e pt-BR) ganham o caminho por binário.
- **RM-004** — ADR 003: `.trilha/` é autoritativo em trilha-spec, trilha-runner e trilha-cloud;
  `.specify/` guarda só constituição e templates; `specs/NNN/` são registros históricos
  congelados; o framework `trilha` segue com Spec Kit até migrar. O bloco SPECKIT sai do
  `CLAUDE.md`; roadmap e ADR 001 param de descrever como pendente o que o framework 0.124.0
  entregou; as ideias abertas (RM-050, 052, 053, 054, 055) viram tasks `idea`.
- **doctor** — aviso `spec-tasks-done`: spec `approved` cujas tasks estão todas `done` deveria
  estar `done`. Aviso, não falha.

## Fora de escopo

- Cortar a tag `v0.3.0` e publicar a release (ação humana; o workflow só reage à tag).
- Corrigir o fixture do runner (é PR no `trilha-runner`; até ela sair o job `conformance`
  fica vermelho de propósito).
- Homebrew, imagens Docker, assinatura cosign (RM-040).
- Mover specs `approved` antigas para `done`: nenhuma neste repositório está nesse estado.

## Impacto de segurança

- **Ativos**: workflows de CI/release, binários e checksums publicados, `.trilha/` como fonte
  da verdade, fixtures do contrato de execução.
- **Fronteiras de confiança**: GitHub Actions → GitHub Releases (`GITHUB_TOKEN` com
  `contents: write` apenas no job de release, `contents: read` no CI); trilha-spec → checkout
  irmão do runner (leitura, nunca escrita); usuário → binário baixado, verificado por
  `sha256sums.txt`.
- **Controles**: ações pinadas por SHA de commit com o comentário da versão (ASVS V14.2);
  build reproduzível e checksums (V10.3); a versão do binário é a da tag, verificada por teste
  antes do build (V1.14). Nenhum segredo novo; o workflow não lê `secrets.*` além do token
  padrão.

## Evidência

- `make test` — gofmt completo, vet, testes (inclui `TestVersionMatchesReleaseTag`, que só
  compara quando `RELEASE_TAG` está definido, e `TestCheckSpecStatus`).
- `RELEASE_TAG=v0.3.0 go test ./cmd/trilha-spec -run TestVersionMatchesReleaseTag` passa;
  `RELEASE_TAG=v9.9.9` falha.
- `make conformance` — hoje acusa o fixture do runner (sem `provider`, `tokens_in`,
  `tokens_out`, `cost`, `currency`), que é exatamente o que RM-003 corrige no runner.
- `trilha-spec task list --status idea` lista as cinco ideias do roadmap.
- `trilha-spec doctor` limpo neste repositório.

## Aceitação

- **SC-001** `gofmt -l .` vazio e CI usando o mesmo comando.
- **SC-002** o job `conformance` existe, faz checkout do runner e falha quando o fixture diverge.
- **SC-003** `release.yml` produz 4 binários + `sha256sums.txt` e recusa tag diferente de `version`.
- **SC-004** ADR 003 publicada; `CLAUDE.md` sem bloco SPECKIT; roadmap e ADR 001 sem texto obsoleto.
- **SC-005** `doctor` avisa `spec NNN: all tasks done, status should be done` sem falhar.
