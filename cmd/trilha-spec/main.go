// Command trilha-spec is the CLI of the Trilha protocol: init, spec, task,
// agent, verify, context, evidence, graph, mcp, doctor.
//
// It is a separate binary from the Trilha web framework's `trilha` on
// purpose — the two share a name and a project, not a command table. Once
// the framework dispatches unknown subcommands to `trilha-<name>` in PATH
// (git style), `trilha spec task next` and `trilha-spec task next` are the
// same call. See docs/adr/001.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/emersonjoe/trilha-spec/agent"
	"github.com/emersonjoe/trilha-spec/ai"
	"github.com/emersonjoe/trilha-spec/mcp"
	"github.com/emersonjoe/trilha-spec/spec"
	"github.com/emersonjoe/trilha-spec/task"
)

const version = "0.1.0"

const usage = `trilha-spec ` + version + ` — the open protocol for work agents can execute

usage: trilha-spec <command> [flags]

  init [dir]                    create .trilha/ (project, constitution, agents)
  spec new <title> | list | show <id>
  task add <title> [--spec ID] [--depends A,B] [--agent N] [--accept C]... [--check CMD]...
  task list [--status S] | show <id> | next | move <id> <status> | graph [--dot]
  agent list | show <name>
  context <task-id>             the context pack an agent receives (--json for tools)
  verify <task-id> [--dir D]    run the task's checks and record evidence
  evidence <task-id> [add --note TEXT]
  mcp [--write]                 serve the protocol over MCP on stdio
  doctor                        what a reader would trip on
  version

Every listing takes --json. Statuses: idea spec ready running verify review done blocked failed.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	err := run(os.Args[1], os.Args[2:], os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(cmd string, args []string, out io.Writer) error {
	switch cmd {
	case "init":
		return cmdInit(args, out)
	case "spec":
		return cmdSpec(args, out)
	case "task":
		return cmdTask(args, out)
	case "agent":
		return cmdAgent(args, out)
	case "context":
		return cmdContext(args, out)
	case "verify":
		return cmdVerify(args, out)
	case "evidence":
		return cmdEvidence(args, out)
	case "mcp":
		return cmdMCP(args)
	case "doctor":
		return cmdDoctor(args, out)
	case "version", "-v", "--version":
		fmt.Fprintln(out, "trilha-spec", version)
		return nil
	case "help", "-h", "--help":
		fmt.Fprint(out, usage)
		return nil
	}
	return fmt.Errorf("unknown command %q\n%s", cmd, usage)
}

// multi collects a repeatable flag.
type multi []string

func (m *multi) String() string     { return strings.Join(*m, ",") }
func (m *multi) Set(s string) error { *m = append(*m, s); return nil }

func flags(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func store() (*task.Store, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return task.Open(cwd)
}

func printJSON(out io.Writer, v any) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func cmdInit(args []string, out io.Writer) error {
	fs := flags("init")
	name := fs.String("name", "", "project name (default: directory name)")
	desc := fs.String("description", "", "one-line description")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	dir := "."
	if len(pos) > 0 {
		dir = pos[0]
	}
	l, wrote, err := spec.Init(dir, spec.InitOptions{Name: *name, Description: *desc})
	if err != nil {
		return err
	}
	if len(wrote) == 0 {
		fmt.Fprintf(out, "%s already initialized; nothing written\n", l.Dir())
		return nil
	}
	for _, w := range wrote {
		fmt.Fprintln(out, "  created", w)
	}
	fmt.Fprintln(out, "next: describe the project in .trilha/project.md, then `trilha-spec spec new \"<title>\"`")
	return nil
}

func cmdSpec(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: trilha-spec spec new <title> | list | show <id>")
	}
	st, err := store()
	if err != nil {
		return err
	}
	l := st.Layout
	switch args[0] {
	case "new":
		fs := flags("spec new")
		issue := fs.String("issue", "", "issue URL or number")
		pos, err := parse(fs, args[1:])
		if err != nil {
			return err
		}
		title := strings.TrimSpace(strings.Join(pos, " "))
		if title == "" {
			return errors.New("usage: trilha-spec spec new <title>")
		}
		id, err := l.NextSpecID(title)
		if err != nil {
			return err
		}
		s := spec.NewSpecDoc(id, title)
		s.Issue = *issue
		if err := l.SaveSpec(s); err != nil {
			return err
		}
		fmt.Fprintf(out, "created %s (%s)\n", id, rel(l, l.SpecFile(id)))
		return nil
	case "list":
		fs := flags("spec list")
		asJSON := fs.Bool("json", false, "")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		specs, err := l.ListSpecs()
		if err != nil {
			return err
		}
		if *asJSON {
			return printJSON(out, specs)
		}
		for _, s := range specs {
			fmt.Fprintf(out, "%-28s %-9s %s\n", s.ID, s.Status, s.Title)
		}
		return nil
	case "show":
		fs := flags("spec show")
		asJSON := fs.Bool("json", false, "")
		pos, err := parse(fs, args[1:])
		if err != nil {
			return err
		}
		if len(pos) != 1 {
			return errors.New("usage: trilha-spec spec show <id>")
		}
		s, err := l.LoadSpec(pos[0])
		if err != nil {
			return err
		}
		if *asJSON {
			return printJSON(out, s)
		}
		_, err = out.Write(s.Bytes())
		return err
	}
	return fmt.Errorf("unknown spec command %q", args[0])
}

func cmdTask(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: trilha-spec task add|list|show|next|move|graph")
	}
	st, err := store()
	if err != nil {
		return err
	}
	switch args[0] {
	case "add":
		fs := flags("task add")
		specID := fs.String("spec", "", "specification id")
		deps := fs.String("depends", "", "comma-separated task ids")
		ag := fs.String("agent", "", "agent name")
		status := fs.String("status", string(task.Idea), "initial status")
		var accept, checks multi
		fs.Var(&accept, "accept", "acceptance criterion (repeatable)")
		fs.Var(&checks, "check", "check command (repeatable)")
		pos, err := parse(fs, args[1:])
		if err != nil {
			return err
		}
		title := strings.TrimSpace(strings.Join(pos, " "))
		if title == "" {
			return errors.New("usage: trilha-spec task add <title> [flags]")
		}
		t, err := st.Create(title, func(t *task.Task) {
			t.Spec = *specID
			t.Agent = *ag
			t.Status = task.Status(*status)
			t.Acceptance = accept
			t.Checks = checks
			for _, d := range strings.Split(*deps, ",") {
				if d = strings.TrimSpace(d); d != "" {
					t.DependsOn = append(t.DependsOn, d)
				}
			}
		})
		if err != nil {
			return err
		}
		if _, err := st.Graph(); err != nil {
			os.Remove(st.Layout.TaskFile(t.ID))
			return err
		}
		fmt.Fprintf(out, "created %s (%s)\n", t.ID, rel(st.Layout, st.Layout.TaskFile(t.ID)))
		return nil
	case "list":
		fs := flags("task list")
		asJSON := fs.Bool("json", false, "")
		status := fs.String("status", "", "filter by status")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		tasks, err := st.List()
		if err != nil {
			return err
		}
		g, err := st.Graph()
		if err != nil {
			return err
		}
		var kept []*task.Task
		for _, t := range tasks {
			if *status == "" || string(t.Status) == *status {
				kept = append(kept, t)
			}
		}
		if *asJSON {
			return printJSON(out, kept)
		}
		fmt.Fprintf(out, "%-10s %-8s %-40s %s\n", "ID", "STATUS", "TITLE", "WAITING ON")
		for _, t := range kept {
			fmt.Fprintf(out, "%-10s %-8s %-40s %s\n", t.ID, t.Status, trunc(t.Title, 40), strings.Join(g.Blockers(t.ID), ","))
		}
		return nil
	case "show":
		fs := flags("task show")
		asJSON := fs.Bool("json", false, "")
		pos, err := parse(fs, args[1:])
		if err != nil {
			return err
		}
		if len(pos) != 1 {
			return errors.New("usage: trilha-spec task show <id>")
		}
		t, err := st.Get(pos[0])
		if err != nil {
			return err
		}
		if *asJSON {
			return printJSON(out, t)
		}
		_, err = out.Write(t.Bytes())
		return err
	case "next":
		fs := flags("task next")
		asJSON := fs.Bool("json", false, "")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		g, err := st.Graph()
		if err != nil {
			return err
		}
		ready := g.Ready()
		if *asJSON {
			if ready == nil {
				ready = []*task.Task{}
			}
			return printJSON(out, ready)
		}
		if len(ready) == 0 {
			fmt.Fprintln(out, "nothing ready: no task is `ready` with every dependency done")
			return nil
		}
		for _, t := range ready {
			fmt.Fprintf(out, "%s  %s\n", t.ID, t.Title)
		}
		return nil
	case "move":
		if len(args) != 3 {
			return errors.New("usage: trilha-spec task move <id> <status>")
		}
		t, err := st.Move(args[1], task.Status(args[2]))
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s is now %s\n", t.ID, t.Status)
		return nil
	case "graph":
		fs := flags("task graph")
		dot := fs.Bool("dot", false, "Graphviz instead of Mermaid")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		g, err := st.Graph()
		if err != nil {
			return err
		}
		if *dot {
			fmt.Fprint(out, g.DOT())
		} else {
			fmt.Fprint(out, g.Mermaid())
		}
		return nil
	}
	return fmt.Errorf("unknown task command %q", args[0])
}

func cmdAgent(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: trilha-spec agent list | show <name>")
	}
	st, err := store()
	if err != nil {
		return err
	}
	switch args[0] {
	case "list":
		fs := flags("agent list")
		asJSON := fs.Bool("json", false, "")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		agents, err := agent.List(st.Layout)
		if err != nil {
			return err
		}
		if *asJSON {
			return printJSON(out, agents)
		}
		for _, a := range agents {
			fmt.Fprintf(out, "%-12s %-6s %-22s %s\n", a.Name, a.Driver, strings.Join(a.Tools, ","), a.Role)
		}
		return nil
	case "show":
		if len(args) != 2 {
			return errors.New("usage: trilha-spec agent show <name>")
		}
		a, err := agent.Load(st.Layout, args[1])
		if err != nil {
			return err
		}
		_, err = out.Write(a.Bytes())
		return err
	}
	return fmt.Errorf("unknown agent command %q", args[0])
}

func cmdContext(args []string, out io.Writer) error {
	fs := flags("context")
	asJSON := fs.Bool("json", false, "")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return errors.New("usage: trilha-spec context <task-id>")
	}
	st, err := store()
	if err != nil {
		return err
	}
	p, err := ai.Build(st.Layout, pos[0])
	if err != nil {
		return err
	}
	if *asJSON {
		_, err = out.Write(p.JSON())
		return err
	}
	_, err = io.WriteString(out, p.Markdown())
	return err
}

// cmdVerify runs the checks and records the evidence. A task in `verify`
// moves on by itself — to review when everything passed, to failed
// otherwise — because that is what the status means; any other status keeps
// its place and just gains evidence.
func cmdVerify(args []string, out io.Writer) error {
	fs := flags("verify")
	dir := fs.String("dir", "", "directory to run the checks in (default: project root)")
	by := fs.String("by", "trilha-spec verify", "who is recorded as the author")
	asJSON := fs.Bool("json", false, "")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return errors.New("usage: trilha-spec verify <task-id> [--dir D]")
	}
	st, err := store()
	if err != nil {
		return err
	}
	t, err := st.Get(pos[0])
	if err != nil {
		return err
	}
	if *dir == "" {
		*dir = st.Layout.Root
	}
	v, err := task.RunChecks(context.Background(), st.Layout, t, *dir, *by)
	if err != nil {
		return err
	}
	moved := ""
	if t.Status == task.Verify {
		to := task.Failed
		if v.Passed {
			to = task.Review
		}
		if _, err := st.Move(t.ID, to); err != nil {
			return err
		}
		moved = string(to)
	}
	if *asJSON {
		return printJSON(out, map[string]any{"task": t.ID, "passed": v.Passed, "moved_to": moved, "evidence": v.Evidence, "paths": v.Paths})
	}
	for _, e := range v.Evidence {
		mark := "✗"
		if e.Passed {
			mark = "✓"
		}
		if e.Command != "" {
			fmt.Fprintf(out, "%s %s (exit %d)\n", mark, e.Command, e.ExitCode)
		} else {
			fmt.Fprintf(out, "%s %s\n", mark, e.Note)
		}
	}
	fmt.Fprintf(out, "evidence: %d record(s) in %s\n", len(v.Paths), rel(st.Layout, st.Layout.EvidenceDir(t.ID)))
	if moved != "" {
		fmt.Fprintf(out, "%s is now %s\n", t.ID, moved)
	}
	if !v.Passed {
		return errors.New("verification failed")
	}
	return nil
}

func cmdEvidence(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: trilha-spec evidence <task-id> [add --note TEXT]")
	}
	st, err := store()
	if err != nil {
		return err
	}
	id := args[0]
	if len(args) > 1 && args[1] == "add" {
		fs := flags("evidence add")
		note := fs.String("note", "", "text of the note")
		by := fs.String("by", "cli", "author")
		var files multi
		fs.Var(&files, "file", "file produced (repeatable)")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *note == "" && len(files) == 0 {
			return errors.New("evidence add needs --note or --file")
		}
		kind := "note"
		if len(files) > 0 {
			kind = "artifact"
		}
		e, p, err := task.Record(st.Layout, task.Evidence{Task: id, Kind: kind, Note: *note, Files: files, By: *by, Passed: true})
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "recorded #%d (%s)\n", e.Seq, rel(st.Layout, p))
		return nil
	}
	fs := flags("evidence")
	asJSON := fs.Bool("json", false, "")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	list, err := task.ListEvidence(st.Layout, id)
	if err != nil {
		return err
	}
	if *asJSON {
		if list == nil {
			list = []task.Evidence{}
		}
		return printJSON(out, list)
	}
	for _, e := range list {
		mark := "✗"
		if e.Passed {
			mark = "✓"
		}
		what := e.Note
		if e.Command != "" {
			what = fmt.Sprintf("%s (exit %d)", e.Command, e.ExitCode)
		}
		fmt.Fprintf(out, "#%-3d %s %-8s %-20s %s\n", e.Seq, mark, e.Kind, e.By, what)
	}
	return nil
}

func cmdMCP(args []string) error {
	fs := flags("mcp")
	write := fs.Bool("write", false, "offer the tools that change tasks and record evidence")
	if err := fs.Parse(args); err != nil {
		return err
	}
	st, err := store()
	if err != nil {
		return err
	}
	s := mcp.NewServer("trilha-spec", version, mcp.Tools(st.Layout, *write)...)
	// stderr, never stdout: stdout is the protocol.
	fmt.Fprintf(os.Stderr, "trilha-spec mcp %s · %s\ntools: %s\n", version, st.Layout.Root, strings.Join(s.Tools(), ", "))
	if !*write {
		fmt.Fprintln(os.Stderr, "read-only; pass --write to offer trilha_move, trilha_evidence and trilha_verify")
	}
	return s.ServeStdio(context.Background(), os.Stdin, os.Stdout)
}

func cmdDoctor(args []string, out io.Writer) error {
	st, err := store()
	if err != nil {
		return err
	}
	problems := st.Layout.Doctor()
	if _, err := st.List(); err != nil {
		problems = append(problems, err.Error())
	} else if _, err := st.Graph(); err != nil {
		problems = append(problems, err.Error())
	}
	if _, err := st.Layout.ListSpecs(); err != nil {
		problems = append(problems, err.Error())
	}
	if _, err := agent.List(st.Layout); err != nil {
		problems = append(problems, err.Error())
	}
	if len(problems) == 0 {
		fmt.Fprintln(out, "✓", st.Layout.Dir(), "is healthy")
		return nil
	}
	for _, p := range problems {
		fmt.Fprintln(out, "✗", p)
	}
	return fmt.Errorf("%d problem(s)", len(problems))
}

func rel(l spec.Layout, p string) string {
	if r, err := relPath(l.Root, p); err == nil {
		return r
	}
	return p
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
