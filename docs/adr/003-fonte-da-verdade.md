# ADR 003 — Uma fonte da verdade: `.trilha/` é autoritativo

- **Data**: 2026-10-20
- **Status**: aceito
- **Contexto**: este repositório nasceu com Spec Kit (`.specify/`, `specs/NNN-slug/spec.md`) e,
  a partir da spec 001, passou a dogfoodar o próprio protocolo em `.trilha/`. Cada feature
  ficou escrita duas vezes: a spec completa em `specs/NNN/spec.md` e um stub em
  `.trilha/specs/NNN.md`. O `CLAUDE.md` ainda carregava o bloco `SPECKIT START…END` mandando
  "ler o plano atual". Um modelo menor que abre o repositório encontra duas verdades e escolhe
  a errada; o roadmap da stack registra isso como RM-004.

## Decisão

1. **`.trilha/` é a fonte da verdade** de specs, tasks, evidência e agentes em `trilha-spec`,
   `trilha-runner` e `trilha-cloud`. Uma spec é `.trilha/specs/NNN-slug.md`, escrita por
   `trilha-spec spec new` e com o corpo completo nela — "Por quê", "O que muda", "Fora de
   escopo", **"Impacto de segurança"** e **"Evidência"** —, não um stub que aponta para outro
   arquivo. As tasks são `.trilha/tasks/TASK-NNN.md`; o estado delas é o que
   `trilha-spec task list` mostra, e o que o `doctor` acusa é o que está errado.
2. **`.specify/` guarda só a constituição e os templates** (`memory/constitution.md`,
   `templates/`). O restante do scaffold do Spec Kit que ainda está lá (`scripts/`,
   `workflows/`, `extensions/`) não é lido por nada neste repositório e sai quando as skills
   `speckit-*` forem aposentadas; até lá, não é referência para nada.
3. **`specs/NNN-slug/` são registros históricos, congelados.** As pastas das specs 001 a 014
   ficam como foram escritas, porque são a memória de como cada decisão foi tomada e os
   commits e issues apontam para elas. Nenhuma spec nova ganha pasta ali, e nenhuma pasta
   existente é editada para "atualizar" o que `.trilha/` já diz. A partir da spec 015 o
   histórico é o próprio arquivo em `.trilha/specs/` e o git.
4. **O framework `trilha` segue com Spec Kit** (`specs/NNN-slug/spec.md` + `plan.md` +
   `tasks.md`, release por `scripts/release.sh`) até migrar. A migração é uma spec dele, não
   desta ADR.

## Consequências

- O `CLAUDE.md` deste repositório aponta para `.trilha/` e para esta ADR, sem bloco SPECKIT.
- `docs/agents/issue-tracker.md` nos três repositórios do protocolo cita esta ADR: a issue é
  a fonte do escopo; a spec em `.trilha/specs/` é a fonte do plano; a task em
  `.trilha/tasks/` é a fonte do estado.
- Estado honesto é parte da decisão: uma spec `approved` cujas tasks estão todas `done` deve
  estar `done` (o `doctor` avisa: `spec-tasks-done`); uma ideia superada por outra spec vai
  para `done` com nota de evidência apontando a spec que a substituiu, ou é removida.
- As ideias abertas vivem como tasks `idea` (`trilha-spec task list --status idea`), cada uma
  citando o item do roadmap da stack que a originou.

## O que não é decidido aqui

- Quando o framework migra para `.trilha/`.
- Se as pastas `specs/NNN/` históricas um dia são movidas para `docs/history/`; hoje ficam
  onde os links existentes as encontram.
