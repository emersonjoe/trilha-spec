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
`

// InitOptions configures Init.
type InitOptions struct {
	// Name of the project, for project.md. Defaults to the directory name.
	Name string
	// Description is the one-line summary in project.md.
	Description string
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
	files := map[string][]byte{
		filepath.Join(l.Dir(), ".gitignore"): []byte(gitignore),
		l.Project():                          projectTemplate(name, o.Description),
		l.Constitution():                     []byte(constitutionTemplate),
		l.AgentFile("coder"):                 []byte(agentCoder),
		l.AgentFile("reviewer"):              []byte(agentReviewer),
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
)

func (p Problem) String() string {
	switch p.Code {
	case ProblemMissingDir:
		return fmt.Sprintf("missing directory %s (run `trilha-spec init`)", p.Arg)
	case ProblemMissingFile:
		return fmt.Sprintf("missing %s (run `trilha-spec init`)", p.Arg)
	case ProblemGitignoreAll:
		return ".trilha/.gitignore ignores everything (`*`): specs, tasks and evidence will not be committed; run `trilha-spec init` to rewrite it"
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
	return problems
}

func projectTemplate(name, desc string) []byte {
	d := &Doc{}
	d.Fields.Set("name", name)
	d.Fields.Set("description", desc)
	d.Fields.Set("default_agent", "coder")
	d.Fields.SetList("verify", []string{})
	d.Body = `# ` + name + `

What this project is, in the words an agent reading it for the first time needs:
what it does, who uses it, where the code lives, how to run it and how to test it.

## Commands

- build:
- test:

## Where things are

- ` + "`app/`" + `:
`
	return d.Bytes()
}

const constitutionTemplate = `# Constitution

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
`

const agentCoder = `---
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
`

const agentReviewer = `---
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
`
