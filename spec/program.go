package spec

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ProgramFile is the name of the optional manifest that sits above a set of
// repositories: `program.md` in a parent directory of the project.
const ProgramFile = "program.md"

var reAlias = regexp.MustCompile(`^[a-z0-9]+([.-][a-z0-9]+)*$`)

// ValidAlias answers whether a is a repository alias: lowercase words joined
// by `-` or `.`, the same shape as a key id.
func ValidAlias(a string) bool { return reAlias.MatchString(a) }

// Program is a set of repositories delivered together: the product, the
// framework it runs on, the control plane. It is optional and it is not part
// of `.trilha/` — one programme spans several of those, so it lives above
// them, next to the checkouts.
//
//	program.md
//	app/.trilha/
//	trilha/.trilha/
//
// A local-only project never has one and never notices it exists.
type Program struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	// Repos maps an alias to where that repository is, relative to this file
	// (`app`) or absolute. A remote dependency `app:TASK-004` resolves
	// through the alias.
	Repos map[string]string `json:"repos,omitempty"`
	// Milestones are shared across the repositories: the programme's
	// calendar, in the shape of §12.
	Milestones []Milestone `json:"milestones,omitempty"`
	// Dir is where the file was found; Repos resolve against it.
	Dir    string `json:"dir,omitempty"`
	Body   string `json:"body,omitempty"`
	Fields Fields `json:"-"`
}

// ParseProgram reads a program manifest.
func ParseProgram(src []byte) (*Program, error) {
	d, err := Parse(src)
	if err != nil {
		return nil, err
	}
	p := &Program{
		Name:        d.Fields.Get("name"),
		Description: d.Fields.Get("description"),
		Repos:       d.Fields.GetMap("repos"),
		Milestones:  milestonesFrom(d.Fields),
		Body:        d.Body,
		Fields:      d.Fields,
	}
	return p, p.Validate()
}

// Validate checks what a reader relies on.
func (p *Program) Validate() error {
	var errs []string
	if strings.TrimSpace(p.Name) == "" {
		errs = append(errs, "name is required")
	}
	for alias, path := range p.Repos {
		if !ValidAlias(alias) {
			errs = append(errs, fmt.Sprintf("repos %q is not an alias", alias))
		}
		if strings.TrimSpace(path) == "" {
			errs = append(errs, "repos."+alias+" has no path")
		}
	}
	errs = append(errs, validateMilestones(p.Milestones)...)
	sort.Strings(errs)
	if len(errs) > 0 {
		return errors.New("program: " + strings.Join(errs, "; "))
	}
	return nil
}

// Bytes writes the manifest back.
func (p *Program) Bytes() []byte {
	d := &Doc{Fields: p.Fields, Body: p.Body}
	d.Fields.Set("name", p.Name)
	if p.Description != "" || d.Fields.Has("description") {
		d.Fields.Set("description", p.Description)
	}
	if len(p.Repos) > 0 {
		d.Fields.SetMap("repos", p.Repos)
	} else {
		d.Fields.Delete("repos")
	}
	if len(p.Milestones) > 0 {
		d.Fields.SetItems("milestones", milestoneFields(p.Milestones))
	} else {
		d.Fields.Delete("milestones")
	}
	return d.Bytes()
}

// Checkouts answers each alias resolved to a directory that really holds a
// Trilha project, absolute. An alias pointing at something else — a URL, a
// directory that is not checked out — is simply left out: unresolved is a
// state the protocol has a word for, not an error.
func (p *Program) Checkouts() map[string]string {
	out := map[string]string{}
	if p == nil {
		return out
	}
	for alias, path := range p.Repos {
		dir := path
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(p.Dir, path)
		}
		if st, err := os.Stat(filepath.Join(dir, DirName)); err == nil && st.IsDir() {
			abs, err := filepath.Abs(dir)
			if err != nil {
				continue
			}
			out[alias] = abs
		}
	}
	return out
}

// AliasOf answers the alias the programme gives a checkout, or "" when it
// names none: which subgraph a repository's tasks belong to.
func (p *Program) AliasOf(root string) string {
	abs, err := filepath.Abs(root)
	if err != nil {
		return ""
	}
	for alias, dir := range p.Checkouts() {
		if sameDir(dir, abs) {
			return alias
		}
	}
	return ""
}

func sameDir(a, b string) bool {
	if a == b {
		return true
	}
	// Windows compares paths without case; a programme that names `App`
	// and a checkout at `app` are the same directory there.
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

// FindProgram walks up from the parent of root looking for `program.md`. A
// project with none answers nil and no error: the manifest is optional, and
// most projects are one repository.
func (l Layout) FindProgram() (*Program, error) {
	dir, err := filepath.Abs(l.Root)
	if err != nil {
		return nil, err
	}
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, nil
		}
		dir = parent
		p := filepath.Join(dir, ProgramFile)
		b, err := os.ReadFile(p)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		prog, err := ParseProgram(b)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		prog.Dir = dir
		return prog, nil
	}
}

// ProblemRepoUnknown: a task depends on `<alias>:TASK-NNN` with an alias
// `project.md` does not declare in `repos`. Arg is "TASK-NNN alias:TASK-NNN".
const ProblemRepoUnknown = "repo-unknown"
