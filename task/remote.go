package task

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/emersonjoe/trilha-spec/spec"
)

// Ref is one entry of `depends_on`: a task in this repository, or a task in
// another one, named `<alias>:TASK-NNN` after an alias `project.md` declares
// in `repos`.
type Ref struct {
	// Alias is "" for a local dependency.
	Alias string
	ID    string
}

// ParseRef reads a dependency. It answers false for anything that is neither
// a task id nor `<alias>:TASK-NNN`.
func ParseRef(s string) (Ref, bool) {
	alias, id, remote := strings.Cut(s, ":")
	if !remote {
		if !ValidID(s) {
			return Ref{}, false
		}
		return Ref{ID: s}, true
	}
	if !spec.ValidAlias(alias) || !ValidID(id) {
		return Ref{}, false
	}
	return Ref{Alias: alias, ID: id}, true
}

// Remote answers whether the dependency lives in another repository.
func (r Ref) Remote() bool { return r.Alias != "" }

// String writes the reference back the way it is read.
func (r Ref) String() string {
	if r.Remote() {
		return r.Alias + ":" + r.ID
	}
	return r.ID
}

// Waiting is what a blocker looks like when nobody could answer for it:
// `waiting:<alias>:TASK-NNN`. The task cannot start, and the reason says
// exactly which repository has to be reached.
func (r Ref) Waiting() string { return "waiting:" + r.String() }

// Refs answers the parsed dependencies of a task, skipping what does not
// parse — Validate is what refuses those.
func (t *Task) Refs() []Ref {
	out := make([]Ref, 0, len(t.DependsOn))
	for _, d := range t.DependsOn {
		if r, ok := ParseRef(d); ok {
			out = append(out, r)
		}
	}
	return out
}

// Resolver answers the status of a task in another repository. The protocol
// names the question; who answers it is not its business. A sibling checkout
// answers from disk (Checkouts); a control plane answers over its own
// transport, and nothing in this module opens a socket to find out.
type Resolver interface {
	// Status answers the status of alias:id and whether it could be
	// answered at all.
	Status(alias, id string) (Status, bool)
}

// Checkouts resolves remote references from sibling checkouts on disk, by
// alias. It is what `--repo alias=path` builds.
type Checkouts map[string]string

// Status reads the task from the sibling checkout. A path that is not a
// Trilha project, or a task that is not in it, answers false: unresolved is
// not the same as not done.
func (c Checkouts) Status(alias, id string) (Status, bool) {
	root, ok := c[alias]
	if !ok {
		return "", false
	}
	b, err := os.ReadFile(filepath.Join(root, spec.DirName, "tasks", id+".md"))
	if err != nil {
		return "", false
	}
	t, err := Parse(b)
	if err != nil {
		return "", false
	}
	return t.Status, true
}

// ParseCheckout reads one `alias=path` pair.
func ParseCheckout(s string) (alias, path string, ok bool) {
	alias, path, ok = strings.Cut(s, "=")
	if !ok || !spec.ValidAlias(alias) || strings.TrimSpace(path) == "" {
		return "", "", false
	}
	return alias, path, true
}

var reRepoURL = regexp.MustCompile(`\S`)

// CheckRepos reports a remote dependency whose alias `project.md` does not
// declare in `repos`. Without the declaration nobody knows what repository
// the alias means, and the dependency is a dead reference with a colon in it.
func CheckRepos(p *spec.Project, tasks []*Task) []spec.Problem {
	declared := map[string]bool{}
	if p != nil {
		for alias := range p.Repos {
			declared[alias] = true
		}
	}
	var problems []spec.Problem
	for _, t := range tasks {
		for _, r := range t.Refs() {
			if r.Remote() && !declared[r.Alias] {
				problems = append(problems, spec.Problem{Code: spec.ProblemRepoUnknown, Arg: t.ID + " " + r.String()})
			}
		}
	}
	return problems
}
