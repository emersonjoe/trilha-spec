package spec

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DirName is the directory the protocol lives in, at the root of a project.
const DirName = ".trilha"

// Layout is where each kind of document lives. Every path is derived from
// Root; nothing is configurable, because a layout two tools disagree on is
// two protocols.
//
//	.trilha/
//	├── project.md        what this project is, for an agent that just arrived
//	├── constitution.md   the rules every task obeys
//	├── specs/            NNN-name.md — what to build and why
//	├── tasks/            TASK-NNN.md — executable units with acceptance criteria
//	├── agents/           name.md — who may execute, with what
//	├── context/          extra documents an agent receives (architecture, glossary)
//	├── evidence/         TASK-NNN/NNN-kind.json — proof a task produced
//	└── runs/             local runner state; never committed
type Layout struct{ Root string }

// ErrNotInitialized is a directory without a .trilha/ above it.
var ErrNotInitialized = errors.New("spec: no " + DirName + "/ found here or above (run `trilha-spec init`)")

// Find walks up from start until it meets a directory holding .trilha/.
func Find(start string) (Layout, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return Layout{}, err
	}
	for {
		if st, err := os.Stat(filepath.Join(dir, DirName)); err == nil && st.IsDir() {
			return Layout{Root: dir}, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return Layout{}, ErrNotInitialized
		}
		dir = parent
	}
}

// Dir is .trilha/ itself.
func (l Layout) Dir() string { return filepath.Join(l.Root, DirName) }

// Project is .trilha/project.md.
func (l Layout) Project() string { return filepath.Join(l.Dir(), "project.md") }

// Constitution is .trilha/constitution.md.
func (l Layout) Constitution() string { return filepath.Join(l.Dir(), "constitution.md") }

// Specs is .trilha/specs/.
func (l Layout) Specs() string { return filepath.Join(l.Dir(), "specs") }

// Tasks is .trilha/tasks/.
func (l Layout) Tasks() string { return filepath.Join(l.Dir(), "tasks") }

// Agents is .trilha/agents/.
func (l Layout) Agents() string { return filepath.Join(l.Dir(), "agents") }

// Context is .trilha/context/.
func (l Layout) Context() string { return filepath.Join(l.Dir(), "context") }

// Evidence is .trilha/evidence/.
func (l Layout) Evidence() string { return filepath.Join(l.Dir(), "evidence") }

// Runs is .trilha/runs/, the runner's scratch space. It is git-ignored by
// Init because it holds worktrees and logs, not protocol.
func (l Layout) Runs() string { return filepath.Join(l.Dir(), "runs") }

// TaskFile is the path of one task.
func (l Layout) TaskFile(id string) string { return filepath.Join(l.Tasks(), id+".md") }

// SpecFile is the path of one specification.
func (l Layout) SpecFile(id string) string { return filepath.Join(l.Specs(), id+".md") }

// AgentFile is the path of one agent manifest.
func (l Layout) AgentFile(name string) string { return filepath.Join(l.Agents(), name+".md") }

// EvidenceDir is where one task's evidence goes.
func (l Layout) EvidenceDir(id string) string { return filepath.Join(l.Evidence(), id) }

// Keys is .trilha/keys: the public keys evidence signatures are checked
// against, one `<key_id>.pub` per key. Private keys never live here.
func (l Layout) Keys() string { return filepath.Join(l.Dir(), "keys") }

// gitignore is what Init writes in .trilha/.gitignore. The Trilha web
// framework's dev server writes `*` in this same file because it keeps its
// build cache in .trilha/; the protocol needs the opposite — everything
// committed except the runner scratch — so the two have to agree on this
// list. See docs/adr/001.
const gitignore = `# Written by trilha-spec init. Protocol files are committed; runner state is not.
runs/
cache/
app
app.exe
export-app
export-app.exe
# A private signing key is never committed; keygen writes it elsewhere.
keys/*.key
`

// InitOptions configures Init.
type InitOptions struct {
	// Name of the project, for project.md. Defaults to the directory name.
	Name string
	// Description is the one-line summary in project.md.
	Description string
	// Lang selects the language of the templates written: "en" (default) or
	// "pt". Field names stay English whatever the language; only prose moves.
	Lang string
}

// Init creates .trilha/ with its directories, a project.md, a constitution
// and two default agents. It never overwrites a file that exists: running it
// twice is safe, and a project with a hand-written constitution keeps it.
// It answers the files it wrote.
func Init(root string, o InitOptions) (Layout, []string, error) {
	l := Layout{Root: root}
	name := o.Name
	if name == "" {
		abs, _ := filepath.Abs(root)
		name = filepath.Base(abs)
	}
	var wrote []string
	for _, d := range []string{l.Dir(), l.Specs(), l.Tasks(), l.Agents(), l.Context(), l.Evidence()} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return l, wrote, err
		}
	}
	tpl := Templates(o.Lang)
	files := map[string][]byte{
		filepath.Join(l.Dir(), ".gitignore"): []byte(gitignore),
		l.Project():                          projectTemplate(name, o.Description, tpl),
		l.Constitution():                     []byte(tpl.Constitution),
		l.AgentFile("coder"):                 []byte(tpl.AgentCoder),
		l.AgentFile("reviewer"):              []byte(tpl.AgentReviewer),
		filepath.Join(l.Specs(), ".keep"):    nil,
		filepath.Join(l.Tasks(), ".keep"):    nil,
		filepath.Join(l.Context(), ".keep"):  nil,
		filepath.Join(l.Evidence(), ".keep"): nil,
	}
	for _, p := range SortedKeys(files) {
		if _, err := os.Stat(p); err == nil {
			continue
		}
		if err := os.WriteFile(p, files[p], 0o644); err != nil {
			return l, wrote, err
		}
		rel, _ := filepath.Rel(root, p)
		wrote = append(wrote, filepath.ToSlash(rel))
	}
	sort.Strings(wrote)
	return l, wrote, nil
}

// Problem is one thing Doctor found. Code is stable so a CLI can render it
// in its own language; String is the English rendering.
type Problem struct {
	Code string
	// Arg is the path or ID the problem is about, relative to Root.
	Arg string
}

// Problem codes.
const (
	ProblemMissingDir   = "missing-dir"
	ProblemMissingFile  = "missing-file"
	ProblemGitignoreAll = "gitignore-all"
	// ProblemPrivateKey: a `.key` file under .trilha/keys; Arg is the path.
	ProblemPrivateKey = "private-key-in-repo"
)

func (p Problem) String() string {
	switch p.Code {
	case ProblemMissingDir:
		return fmt.Sprintf("missing directory %s (run `trilha-spec init`)", p.Arg)
	case ProblemMissingFile:
		return fmt.Sprintf("missing %s (run `trilha-spec init`)", p.Arg)
	case ProblemGitignoreAll:
		return ".trilha/.gitignore ignores everything (`*`): specs, tasks and evidence will not be committed; run `trilha-spec init` to rewrite it"
	case ProblemPrivateKey:
		return "private key " + p.Arg + " is inside .trilha; move it out (only `.pub` files belong in keys/)"
	case ProblemSpecRefMissing:
		return "spec reference does not exist: " + p.Arg
	case ProblemSpecNoSuccessor:
		return "spec " + p.Arg + " is superseded but no spec names it in `supersedes`"
	case ProblemSpecNoSecurity:
		return "spec " + p.Arg + " is approved but declares no security impact (assets, trust_boundaries, controls, evidence)"
	case ProblemRequirementDuplicate:
		return "requirement declared by more than one spec: " + p.Arg
	case ProblemRequirementUnknown:
		return "task covers a requirement no spec declares: " + p.Arg
	case ProblemRequirementUncovered:
		return "requirement no task covers: " + p.Arg
	case ProblemMilestoneUnknown:
		return "milestone project.md does not declare: " + p.Arg
	case ProblemMilestoneEmpty:
		return "milestone " + p.Arg + " has no task"
	case ProblemMilestonePastDue:
		return "task past its milestone's due date: " + p.Arg
	}
	return p.Code + " " + p.Arg
}

// Doctor lists what is wrong with a layout that a reader would trip on: a
// missing directory, or a .gitignore that hides the protocol from git.
func (l Layout) Doctor() []Problem {
	var problems []Problem
	for _, d := range []string{l.Specs(), l.Tasks(), l.Agents(), l.Evidence()} {
		if st, err := os.Stat(d); err != nil || !st.IsDir() {
			rel, _ := filepath.Rel(l.Root, d)
			problems = append(problems, Problem{ProblemMissingDir, filepath.ToSlash(rel)})
		}
	}
	for _, f := range []string{l.Project(), l.Constitution()} {
		if _, err := os.Stat(f); err != nil {
			rel, _ := filepath.Rel(l.Root, f)
			problems = append(problems, Problem{ProblemMissingFile, filepath.ToSlash(rel)})
		}
	}
	if b, err := os.ReadFile(filepath.Join(l.Dir(), ".gitignore")); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if strings.TrimSpace(line) == "*" {
				problems = append(problems, Problem{Code: ProblemGitignoreAll})
				break
			}
		}
	}
	if entries, err := os.ReadDir(l.Keys()); err == nil {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".key") {
				problems = append(problems, Problem{ProblemPrivateKey, DirName + "/keys/" + e.Name()})
			}
		}
	}
	return problems
}

func projectTemplate(name, desc string, tpl TemplateSet) []byte {
	d := &Doc{}
	d.Fields.Set("name", name)
	d.Fields.Set("description", desc)
	d.Fields.Set("default_agent", "coder")
	d.Fields.SetList("verify", []string{})
	d.Body = "# " + name + "\n\n" + tpl.ProjectBody
	return d.Bytes()
}

// TemplateSet is the prose `init` and `spec new` write, in one language.
// Front matter keys are never translated: they are the protocol.
type TemplateSet struct {
	ProjectBody   string
	Constitution  string
	AgentCoder    string
	AgentReviewer string
	// SpecBody is the body of a new specification; %s is the title.
	SpecBody string
}

// Templates answers the set for a language: "pt" (Brazilian Portuguese) or
// anything else for English.
func Templates(lang string) TemplateSet {
	if lang == "pt" {
		return templatesPT
	}
	return templatesEN
}

var templatesEN = TemplateSet{
	ProjectBody: `What this project is, in the words an agent reading it for the first time needs:
what it does, who uses it, where the code lives, how to run it and how to test it.

## Commands

- build:
- test:

## Where things are

- ` + "`app/`" + `:
`,
	Constitution: `# Constitution

The rules every task in this project obeys. An agent reads this before it
reads the task; a reviewer checks the evidence against it.

## Principles

1. **Tests first.** A task that changes behaviour ships the test that proves it.
2. **Small commits, one task each.** The branch of a task holds that task's work and nothing else.
3. **No new dependency without a spec.** A dependency is a decision, and decisions live in ` + "`specs/`" + `.
4. **Evidence or it did not happen.** A task is done when its acceptance criteria have recorded evidence.

## Style

- Language of code and identifiers:
- Formatting and lint:
`,
	AgentCoder: `---
name: coder
role: Implements a task inside its own worktree and produces evidence.
driver: exec
command: ""
tools:
  - read
  - write
  - run
constraints:
  - Stay inside the worktree of the task.
  - Do not touch tasks other than the one assigned.
  - Run the checks listed in the task before reporting.
---

# coder

Reads the context pack, implements the task, runs its checks and reports what
changed. ` + "`driver`" + ` and ` + "`command`" + ` are how the runner starts it; see the
trilha-runner documentation for the drivers available.
`,
	AgentReviewer: `---
name: reviewer
role: Reads the evidence of a task and decides whether it moves to done.
driver: exec
command: ""
tools:
  - read
constraints:
  - Never edits code; a rejected task goes back to ready with a note.
---

# reviewer

Checks each acceptance criterion against the evidence recorded for the task
and the constitution. Approves (review → done) or returns (review → ready).
`,
	SpecBody: `# %s

## Why

The problem, and what people do today without this.

## What changes

The contract as the documentation will tell it.

## Out of scope

## Acceptance

- **SC-001**
`,
}

var templatesPT = TemplateSet{
	ProjectBody: `O que é este projeto, nas palavras de que um agente que o lê pela primeira vez precisa:
o que faz, quem usa, onde está o código, como rodar e como testar.

## Comandos

- build:
- test:

## Onde ficam as coisas

- ` + "`app/`" + `:
`,
	Constitution: `# Constituição

As regras que toda task deste projeto obedece. Um agente lê isto antes de ler a
task; um revisor confere a evidência contra isto.

## Princípios

1. **Teste primeiro.** Uma task que muda comportamento entrega o teste que prova a mudança.
2. **Commits pequenos, um por task.** A branch de uma task contém o trabalho daquela task e nada mais.
3. **Nenhuma dependência nova sem spec.** Dependência é decisão, e decisões vivem em ` + "`specs/`" + `.
4. **Sem evidência não aconteceu.** Uma task está pronta quando seus critérios de aceitação têm evidência gravada.

## Estilo

- Língua do código e dos identificadores:
- Formatação e lint:
`,
	AgentCoder: `---
name: coder
role: Implementa uma task dentro da própria worktree e produz evidência.
driver: exec
command: ""
tools:
  - read
  - write
  - run
constraints:
  - Fique dentro da worktree da task.
  - Não toque em tasks além da que foi atribuída.
  - Rode os checks listados na task antes de reportar.
---

# coder

Lê o pacote de contexto, implementa a task, roda seus checks e reporta o que
mudou. ` + "`driver`" + ` e ` + "`command`" + ` são como o runner o inicia; veja a
documentação do trilha-runner para os drivers disponíveis.
`,
	AgentReviewer: `---
name: reviewer
role: Lê a evidência de uma task e decide se ela vai para done.
driver: exec
command: ""
tools:
  - read
constraints:
  - Nunca edita código; uma task rejeitada volta para ready com uma nota.
---

# reviewer

Confere cada critério de aceitação contra a evidência gravada para a task e
contra a constituição. Aprova (review → done) ou devolve (review → ready).
`,
	SpecBody: `# %s

## Por quê

O problema, e o que as pessoas fazem hoje sem isto.

## O que muda

O contrato, do jeito que a documentação vai contar.

## Fora de escopo

## Aceitação

- **SC-001**
`,
}
