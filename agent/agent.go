// Package agent is the who of the protocol: a manifest per agent in
// .trilha/agents/, saying what it does, how a runner starts it and what it
// may touch. The protocol does not run agents — trilha-runner does — it only
// says how one is described, so a task can name its executor and a reviewer
// can see what that executor was allowed to do.
package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/emersonjoe/trilha-spec/spec"
)

// Agent is one manifest.
type Agent struct {
	Name string `json:"name"`
	Role string `json:"role"`
	// Driver is how a runner starts it: "exec" runs Command with the context
	// pack on stdin; "ai" uses a chat-completion model; others are the
	// runner's business. The protocol only carries the name.
	Driver string `json:"driver,omitempty"`
	// Command is the program the exec driver starts, e.g. `claude -p -`.
	Command string `json:"command,omitempty"`
	// Model is the model the ai driver asks for.
	Model string `json:"model,omitempty"`
	// Tools is what it may do: read, write, run, git, network.
	Tools       []string    `json:"tools,omitempty"`
	Constraints []string    `json:"constraints,omitempty"`
	Body        string      `json:"body,omitempty"`
	Fields      spec.Fields `json:"-"`
}

var reName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ValidName answers whether name is a file-safe agent name.
func ValidName(name string) bool { return reName.MatchString(name) }

// Parse reads a manifest.
func Parse(src []byte) (*Agent, error) {
	d, err := spec.Parse(src)
	if err != nil {
		return nil, err
	}
	a := &Agent{
		Name:        d.Fields.Get("name"),
		Role:        d.Fields.Get("role"),
		Driver:      d.Fields.Get("driver"),
		Command:     d.Fields.Get("command"),
		Model:       d.Fields.Get("model"),
		Tools:       d.Fields.GetList("tools"),
		Constraints: d.Fields.GetList("constraints"),
		Body:        d.Body,
		Fields:      d.Fields,
	}
	return a, a.Validate()
}

// Validate checks the fields a runner relies on.
func (a *Agent) Validate() error {
	var errs []string
	if !ValidName(a.Name) {
		errs = append(errs, fmt.Sprintf("name %q must be lowercase words joined by -", a.Name))
	}
	if strings.TrimSpace(a.Role) == "" {
		errs = append(errs, "role is required")
	}
	for _, t := range a.Tools {
		switch t {
		case "read", "write", "run", "git", "network":
		default:
			errs = append(errs, fmt.Sprintf("tool %q is not one of read, write, run, git, network", t))
		}
	}
	if len(errs) > 0 {
		return errors.New("agent " + a.Name + ": " + strings.Join(errs, "; "))
	}
	return nil
}

// May answers whether the agent is allowed a tool.
func (a *Agent) May(tool string) bool {
	for _, t := range a.Tools {
		if t == tool {
			return true
		}
	}
	return false
}

// Bytes writes the manifest back.
func (a *Agent) Bytes() []byte {
	d := &spec.Doc{Fields: a.Fields, Body: a.Body}
	d.Fields.Set("name", a.Name)
	d.Fields.Set("role", a.Role)
	d.Fields.Set("driver", a.Driver)
	d.Fields.Set("command", a.Command)
	if a.Model != "" {
		d.Fields.Set("model", a.Model)
	}
	d.Fields.SetList("tools", a.Tools)
	d.Fields.SetList("constraints", a.Constraints)
	return d.Bytes()
}

// Load reads one agent by name.
func Load(l spec.Layout, name string) (*Agent, error) {
	if !ValidName(name) {
		return nil, fmt.Errorf("agent: %q is not a valid name", name)
	}
	b, err := os.ReadFile(l.AgentFile(name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("agent %s: not found in %s", name, l.Agents())
		}
		return nil, err
	}
	return Parse(b)
}

// Save writes one agent.
func Save(l spec.Layout, a *Agent) error {
	if err := a.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(l.Agents(), 0o755); err != nil {
		return err
	}
	return os.WriteFile(l.AgentFile(a.Name), a.Bytes(), 0o644)
}

// List answers every agent, sorted by name.
func List(l spec.Layout) ([]*Agent, error) {
	entries, err := os.ReadDir(l.Agents())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []*Agent
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(l.Agents(), e.Name()))
		if err != nil {
			return nil, err
		}
		a, err := Parse(b)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		if a.Name+".md" != e.Name() {
			return nil, fmt.Errorf("%s: file name and agent name %q disagree", e.Name(), a.Name)
		}
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
