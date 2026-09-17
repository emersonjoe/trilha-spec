// Package ai builds what a model receives: the context pack of a task. It is
// the protocol's answer to "what does an agent need to know?" — project,
// constitution, specification, task, the tasks it depends on, the evidence so
// far, the manifest of the agent itself and whatever lives in
// .trilha/context/. One function, two renderings: Markdown for a prompt,
// JSON for a tool.
package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/emersonjoe/trilha-spec/agent"
	"github.com/emersonjoe/trilha-spec/spec"
	"github.com/emersonjoe/trilha-spec/task"
)

// Pack is everything an agent gets for one task.
type Pack struct {
	Project      *spec.Project `json:"project"`
	Constitution string        `json:"constitution,omitempty"`
	Spec         *spec.Spec    `json:"spec,omitempty"`
	Task         *task.Task    `json:"task"`
	Dependencies []*task.Task  `json:"dependencies,omitempty"`
	// Evidence carries a verdict per record, checked against .trilha/keys:
	// the agent sees what is proven and what is only claimed.
	Evidence []task.Checked `json:"evidence,omitempty"`
	Agent    *agent.Agent   `json:"agent,omitempty"`
	// Context is .trilha/context/*.md, by file name.
	Context map[string]string `json:"context,omitempty"`
}

// Build assembles the pack for a task. A missing spec or agent is not an
// error — the pack says what it has — but a missing task is.
func Build(l spec.Layout, id string) (*Pack, error) {
	st := &task.Store{Layout: l}
	t, err := st.Get(id)
	if err != nil {
		return nil, err
	}
	p := &Pack{Task: t}
	if p.Project, err = l.LoadProject(); err != nil {
		return nil, err
	}
	if p.Constitution, err = l.LoadConstitution(); err != nil {
		return nil, err
	}
	if t.Spec != "" {
		if s, err := l.LoadSpec(t.Spec); err == nil {
			p.Spec = s
		}
	}
	for _, d := range t.DependsOn {
		if dep, err := st.Get(d); err == nil {
			p.Dependencies = append(p.Dependencies, dep)
		}
	}
	evidence, err := task.ListEvidence(l, id)
	if err != nil {
		return nil, err
	}
	if len(evidence) > 0 {
		keys, err := task.ProjectKeys(l)
		if err != nil {
			return nil, err
		}
		p.Evidence = keys.CheckAll(evidence)
	}
	name := t.Agent
	if name == "" {
		name = p.Project.DefaultAgent
	}
	if name != "" {
		if a, err := agent.Load(l, name); err == nil {
			p.Agent = a
		}
	}
	if entries, err := os.ReadDir(l.Context()); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			b, err := os.ReadFile(filepath.Join(l.Context(), e.Name()))
			if err != nil {
				return nil, err
			}
			if p.Context == nil {
				p.Context = map[string]string{}
			}
			p.Context[e.Name()] = string(b)
		}
	}
	return p, nil
}

// JSON renders the pack for a tool.
func (p *Pack) JSON() []byte {
	b, _ := json.MarshalIndent(p, "", "  ")
	return append(b, '\n')
}

// Markdown renders the pack as a prompt: the fixed order below is the order
// a reader needs — rules first, then the goal, then the unit of work, then
// what already exists.
func (p *Pack) Markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Task %s — %s\n\n", p.Task.ID, p.Task.Title)
	fmt.Fprintf(&b, "Status: %s", p.Task.Status)
	if p.Agent != nil {
		fmt.Fprintf(&b, " · Agent: %s", p.Agent.Name)
	}
	b.WriteString("\n\n")
	if p.Project != nil && (p.Project.Name != "" || p.Project.Body != "") {
		fmt.Fprintf(&b, "## Project: %s\n\n", p.Project.Name)
		if p.Project.Description != "" {
			b.WriteString(p.Project.Description + "\n\n")
		}
		if p.Project.Body != "" {
			b.WriteString(strings.TrimSpace(p.Project.Body) + "\n\n")
		}
		// The envelope the agent works inside; the runner enforces it.
		if len(p.Project.Limits) > 0 {
			b.WriteString("### Limits\n\n")
			for _, k := range spec.SortedKeys(p.Project.Limits) {
				fmt.Fprintf(&b, "- %s: %s\n", k, strconv.FormatFloat(p.Project.Limits[k], 'f', -1, 64))
			}
			b.WriteString("\n")
		}
		if p.Project.Paused {
			fmt.Fprintf(&b, "**The project is paused**: %s (since %s). Do not start work.\n\n", p.Project.PauseReason, p.Project.PausedAt)
		}
	}
	if p.Constitution != "" {
		b.WriteString("## Constitution\n\n" + strings.TrimSpace(p.Constitution) + "\n\n")
	}
	if p.Agent != nil {
		fmt.Fprintf(&b, "## You are `%s`\n\n%s\n\n", p.Agent.Name, p.Agent.Role)
		if len(p.Agent.Tools) > 0 {
			fmt.Fprintf(&b, "Tools allowed: %s.\n\n", strings.Join(p.Agent.Tools, ", "))
		}
		list(&b, "Constraints", p.Agent.Constraints)
		if body := strings.TrimSpace(p.Agent.Body); body != "" {
			b.WriteString(body + "\n\n")
		}
	}
	if p.Spec != nil {
		fmt.Fprintf(&b, "## Specification %s — %s\n\n%s\n\n", p.Spec.ID, p.Spec.Title, strings.TrimSpace(p.Spec.Body))
		// The security impact sits next to the acceptance and the checks:
		// what the reviewer will look at is what the agent must keep in view.
		list(&b, "Assets touched", p.Spec.Assets)
		list(&b, "Trust boundaries", p.Spec.TrustBoundaries)
		list(&b, "Controls affected", p.Spec.Controls)
		if len(p.Spec.Evidence) > 0 {
			b.WriteString("### Evidence the reviewer must see\n\n")
			for _, c := range p.Spec.Evidence {
				fmt.Fprintf(&b, "- `%s`\n", c)
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("## The task\n\n")
	if body := strings.TrimSpace(p.Task.Body); body != "" {
		b.WriteString(body + "\n\n")
	}
	list(&b, "Acceptance criteria", p.Task.Acceptance)
	if len(p.Task.Checks) > 0 {
		b.WriteString("### Checks that will run\n\n")
		for _, c := range p.Task.Checks {
			fmt.Fprintf(&b, "- `%s`\n", c)
		}
		b.WriteString("\n")
	}
	if len(p.Dependencies) > 0 {
		b.WriteString("### Depends on\n\n")
		for _, d := range p.Dependencies {
			fmt.Fprintf(&b, "- %s — %s (%s)\n", d.ID, d.Title, d.Status)
		}
		b.WriteString("\n")
	}
	if len(p.Evidence) > 0 {
		b.WriteString("### Evidence so far\n\n")
		for _, e := range p.Evidence {
			fmt.Fprintf(&b, "- #%d %s by %s: ", e.Seq, e.Kind, e.By)
			switch {
			case e.Command != "":
				fmt.Fprintf(&b, "`%s` exit %d", e.Command, e.ExitCode)
			case e.Kind == "run" && e.Cost != 0:
				fmt.Fprintf(&b, "%s %s %s", e.Model, strconv.FormatFloat(e.Cost, 'f', -1, 64), e.Currency)
				if e.Note != "" {
					b.WriteString(" — " + e.Note)
				}
			case e.Note != "":
				b.WriteString(e.Note)
			}
			switch e.Verdict {
			case task.Valid:
				fmt.Fprintf(&b, " · signed by %s", e.Signature.KeyID)
			case task.Invalid:
				fmt.Fprintf(&b, " · SIGNATURE INVALID (%s)", e.Reason)
			default:
				b.WriteString(" · unverified")
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	if len(p.Context) > 0 {
		names := make([]string, 0, len(p.Context))
		for n := range p.Context {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			fmt.Fprintf(&b, "## Context: %s\n\n%s\n\n", n, strings.TrimSpace(p.Context[n]))
		}
	}
	b.WriteString("## When you finish\n\n")
	b.WriteString("Make every acceptance criterion true, run the checks, and report what changed and where the proof is. Do not mark the task done: that is the reviewer's decision, taken from the evidence.\n")
	return b.String()
}

func list(b *strings.Builder, title string, items []string) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(b, "### %s\n\n", title)
	for _, it := range items {
		fmt.Fprintf(b, "- %s\n", it)
	}
	b.WriteString("\n")
}
