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
	"path/filepath"
	"strconv"
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
  spec new <title> [--issue N] [--body TEXT | --body-file PATH] [--asset A]... [--boundary B]... [--control C]... [--evidence CMD]...
  spec list [--status S] | show <id> | move <id> <status>
  spec set <id> [--issue N] [--supersedes A,B] [--depends A,B] [--asset A]... [--boundary B]... [--control C]... [--evidence CMD]...
  task add <title> [--spec ID] [--depends A,B] [--agent N] [--status S] [--accept C]... [--check CMD]...
           [--body TEXT | --body-file PATH]   (PATH "-" reads stdin)
  task list [--status S] | show <id> | next | move <id> <status> | graph [--dot]
  agent list | show <name>
  project show | pause [--reason R] | resume | limit <key> <value|->
  context <task-id>             the context pack an agent receives (--json for tools)
  verify <task-id> [--dir D]    run the task's checks and record evidence
  evidence <task-id> [--verify] [--keys DIR]   records; --verify checks signatures against DIR (default .trilha/keys)
  evidence <task-id> add --note TEXT | add --run [--provider P --model M --tokens-in N --tokens-out N --cost C --currency USD]
           [--sign-key FILE [--key-id ID]]   sign the record with an Ed25519 private key
  keygen <key-id> [--out DIR]   an Ed25519 pair: DIR/<key-id>.key (private, default ~/.trilha/keys) and .trilha/keys/<key-id>.pub
  mcp [--write]                 serve the protocol over MCP on stdio
  doctor                        what a reader would trip on
  version

Every listing takes --json. Task statuses: idea spec ready running verify review done blocked failed.
Spec statuses: draft approved done rejected superseded.
TRILHA_LANG=pt translates the messages and the templates init and spec new write; file formats and --json do not change.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, T(usage))
		os.Exit(2)
	}
	err := run(os.Args[1], os.Args[2:], os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, T("error:"), err)
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
	case "project":
		return cmdProject(args, out)
	case "context":
		return cmdContext(args, out)
	case "verify":
		return cmdVerify(args, out)
	case "evidence":
		return cmdEvidence(args, out)
	case "mcp":
		return cmdMCP(args)
	case "keygen":
		return cmdKeygen(args, out)
	case "doctor":
		return cmdDoctor(args, out)
	case "version", "-v", "--version":
		fmt.Fprintln(out, "trilha-spec", version)
		return nil
	case "help", "-h", "--help":
		fmt.Fprint(out, T(usage))
		return nil
	}
	return fmt.Errorf(T("unknown command %q\n%s"), cmd, T(usage))
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
	l, wrote, err := spec.Init(dir, spec.InitOptions{Name: *name, Description: *desc, Lang: lang})
	if err != nil {
		return err
	}
	if len(wrote) == 0 {
		fmt.Fprintf(out, T("%s already initialized; nothing written\n"), l.Dir())
		return nil
	}
	for _, w := range wrote {
		fmt.Fprintf(out, T("  created %s\n"), w)
	}
	fmt.Fprint(out, T("next: describe the project in .trilha/project.md, then `trilha-spec spec new \"<title>\"`\n"))
	return nil
}

func cmdSpec(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New(T("usage: trilha-spec spec new <title> | list | show <id> | move <id> <status> | set <id> [flags]"))
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
		body := bodyFlags(fs)
		sec := securityFlags(fs)
		pos, err := parse(fs, args[1:])
		if err != nil {
			return err
		}
		text, err := body.read()
		if err != nil {
			return err
		}
		title := strings.TrimSpace(strings.Join(pos, " "))
		if title == "" {
			return errors.New(T("usage: trilha-spec spec new <title>"))
		}
		id, err := l.NextSpecID(title)
		if err != nil {
			return err
		}
		s := spec.NewSpecDoc(id, title, lang, text)
		s.Issue = *issue
		sec.apply(s)
		if err := l.SaveSpec(s); err != nil {
			return err
		}
		fmt.Fprintf(out, T("created %s (%s)\n"), id, rel(l, l.SpecFile(id)))
		return nil
	case "list":
		fs := flags("spec list")
		asJSON := fs.Bool("json", false, "")
		status := fs.String("status", "", "filter by status")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		specs, err := l.ListSpecs()
		if err != nil {
			return err
		}
		var kept []*spec.Spec
		for _, s := range specs {
			if *status == "" || string(s.Status) == *status {
				kept = append(kept, s)
			}
		}
		if *asJSON {
			if kept == nil {
				kept = []*spec.Spec{}
			}
			return printJSON(out, kept)
		}
		for _, s := range kept {
			fmt.Fprintf(out, "%-28s %-10s %s\n", s.ID, s.Status, s.Title)
		}
		return nil
	case "move":
		if len(args) != 3 {
			return errors.New(T("usage: trilha-spec spec move <id> <status>"))
		}
		s, err := l.MoveSpec(args[1], spec.Status(args[2]))
		if err != nil {
			return err
		}
		fmt.Fprintf(out, T("%s is now %s\n"), s.ID, s.Status)
		return nil
	case "set":
		fs := flags("spec set")
		issue := fs.String("issue", "", "issue URL or number; \"\" keeps, \"-\" clears")
		supersedes := fs.String("supersedes", "", "comma-separated spec ids this one replaces")
		deps := fs.String("depends", "", "comma-separated spec ids this one builds on")
		sec := securityFlags(fs)
		pos, err := parse(fs, args[1:])
		if err != nil {
			return err
		}
		if len(pos) != 1 {
			return errors.New(T("usage: trilha-spec spec set <id> [--issue N] [--supersedes A,B] [--depends A,B] [--asset A]... [--boundary B]... [--control C]... [--evidence CMD]..."))
		}
		s, err := l.LoadSpec(pos[0])
		if err != nil {
			return err
		}
		switch *issue {
		case "":
		case "-":
			s.Issue = ""
		default:
			s.Issue = *issue
		}
		if *supersedes != "" {
			s.Supersedes = splitList(*supersedes)
		}
		if *deps != "" {
			s.DependsOn = splitList(*deps)
		}
		sec.apply(s)
		if err := s.Validate(); err != nil {
			return err
		}
		if all, err := l.ListSpecs(); err == nil {
			for _, p := range l.CheckSpecs(replaceSpec(all, s)) {
				if p.Code == spec.ProblemSpecRefMissing && strings.HasPrefix(p.Arg, s.ID+" ") {
					return errors.New(doctorMessage(p))
				}
			}
		}
		if err := l.SaveSpec(s); err != nil {
			return err
		}
		fmt.Fprintf(out, T("updated %s\n"), s.ID)
		return nil
	case "show":
		fs := flags("spec show")
		asJSON := fs.Bool("json", false, "")
		pos, err := parse(fs, args[1:])
		if err != nil {
			return err
		}
		if len(pos) != 1 {
			return errors.New(T("usage: trilha-spec spec show <id>"))
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
	return fmt.Errorf(T("unknown spec command %q"), args[0])
}

func cmdTask(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New(T("usage: trilha-spec task add|list|show|next|move|graph"))
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
		body := bodyFlags(fs)
		pos, err := parse(fs, args[1:])
		if err != nil {
			return err
		}
		text, err := body.read()
		if err != nil {
			return err
		}
		title := strings.TrimSpace(strings.Join(pos, " "))
		if title == "" {
			return errors.New(T("usage: trilha-spec task add <title> [flags]"))
		}
		t, err := st.Create(title, func(t *task.Task) {
			t.Spec = *specID
			t.Agent = *ag
			t.Status = task.Status(*status)
			t.Acceptance = accept
			t.Checks = checks
			t.Body = text
			t.DependsOn = splitList(*deps)
		})
		if err != nil {
			return err
		}
		if _, err := st.Graph(); err != nil {
			os.Remove(st.Layout.TaskFile(t.ID))
			return err
		}
		fmt.Fprintf(out, T("created %s (%s)\n"), t.ID, rel(st.Layout, st.Layout.TaskFile(t.ID)))
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
		fmt.Fprintf(out, "%-10s %-8s %-40s %s\n", T("ID"), T("STATUS"), T("TITLE"), T("WAITING ON"))
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
			return errors.New(T("usage: trilha-spec task show <id>"))
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
		// A paused project answers nothing and says why. The list stays a
		// list for tools; the reason goes to stderr, and `project show --json`
		// has it in full.
		if p, err := st.Layout.LoadProject(); err == nil && p.Paused {
			ready = nil
			if !*asJSON {
				fmt.Fprintf(out, T("project is paused: %s\n"), pauseReason(p))
				return nil
			}
			fmt.Fprintf(os.Stderr, T("project is paused: %s\n"), pauseReason(p))
		}
		if *asJSON {
			if ready == nil {
				ready = []*task.Task{}
			}
			return printJSON(out, ready)
		}
		if len(ready) == 0 {
			fmt.Fprint(out, T("nothing ready: no task is `ready` with every dependency done\n"))
			return nil
		}
		for _, t := range ready {
			fmt.Fprintf(out, "%s  %s\n", t.ID, t.Title)
		}
		return nil
	case "move":
		if len(args) != 3 {
			return errors.New(T("usage: trilha-spec task move <id> <status>"))
		}
		t, err := st.Move(args[1], task.Status(args[2]))
		if err != nil {
			return err
		}
		fmt.Fprintf(out, T("%s is now %s\n"), t.ID, t.Status)
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
	return fmt.Errorf(T("unknown task command %q"), args[0])
}

func cmdAgent(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New(T("usage: trilha-spec agent list | show <name>"))
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
			return errors.New(T("usage: trilha-spec agent show <name>"))
		}
		a, err := agent.Load(st.Layout, args[1])
		if err != nil {
			return err
		}
		_, err = out.Write(a.Bytes())
		return err
	}
	return fmt.Errorf(T("unknown agent command %q"), args[0])
}

func cmdContext(args []string, out io.Writer) error {
	fs := flags("context")
	asJSON := fs.Bool("json", false, "")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return errors.New(T("usage: trilha-spec context <task-id>"))
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
		return errors.New(T("usage: trilha-spec verify <task-id> [--dir D]"))
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
	fmt.Fprintf(out, T("evidence: %d record(s) in %s\n"), len(v.Paths), rel(st.Layout, st.Layout.EvidenceDir(t.ID)))
	if moved != "" {
		fmt.Fprintf(out, T("%s is now %s\n"), t.ID, moved)
	}
	if !v.Passed {
		return errors.New(T("verification failed"))
	}
	return nil
}

func cmdEvidence(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New(T("usage: trilha-spec evidence <task-id> [add --note TEXT | add --run [--provider P] [--model M] [--tokens-in N] [--tokens-out N] [--cost C --currency USD] [--failed]]"))
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
		run := fs.Bool("run", false, "a runner's execution record, with its cost")
		provider := fs.String("provider", "", "model provider (run)")
		model := fs.String("model", "", "model (run)")
		tokensIn := fs.Int("tokens-in", 0, "input tokens (run)")
		tokensOut := fs.Int("tokens-out", 0, "output tokens (run)")
		cost := fs.Float64("cost", 0, "cost as observed (run)")
		currency := fs.String("currency", "", "ISO 4217 code of --cost (run)")
		failed := fs.Bool("failed", false, "the run did not pass")
		signKey := fs.String("sign-key", "", "PEM Ed25519 private key to sign the record with")
		keyID := fs.String("key-id", "", "key id of --sign-key (default: its file name)")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		signer, err := loadSigner(*signKey, *keyID)
		if err != nil {
			return err
		}
		if *note == "" && len(files) == 0 && !*run {
			return errors.New(T("evidence add needs --note, --file or --run"))
		}
		kind := "note"
		switch {
		case *run:
			kind = "run"
		case len(files) > 0:
			kind = "artifact"
		}
		e, p, err := task.RecordSigned(st.Layout, task.Evidence{Task: id, Kind: kind, Note: *note, Files: files, By: *by, Passed: !*failed,
			Provider: *provider, Model: *model, TokensIn: *tokensIn, TokensOut: *tokensOut, Cost: *cost, Currency: *currency}, signer)
		if err != nil {
			return err
		}
		if signer != nil {
			fmt.Fprintf(out, T("recorded #%d (%s), signed by %s\n"), e.Seq, rel(st.Layout, p), signer.KeyID)
			return nil
		}
		fmt.Fprintf(out, T("recorded #%d (%s)\n"), e.Seq, rel(st.Layout, p))
		return nil
	}
	fs := flags("evidence")
	asJSON := fs.Bool("json", false, "")
	verify := fs.Bool("verify", false, "check every signature and report unsigned, valid and invalid")
	keysDir := fs.String("keys", "", "directory of <key_id>.pub files (default .trilha/keys)")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	list, err := task.ListEvidence(st.Layout, id)
	if err != nil {
		return err
	}
	if *verify {
		return verifyEvidence(st.Layout, list, *keysDir, *asJSON, out)
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
		switch {
		case e.Command != "":
			what = fmt.Sprintf("%s (exit %d)", e.Command, e.ExitCode)
		case e.Kind == "run" && (e.Model != "" || e.Cost != 0):
			what = runSummary(e) + " " + e.Note
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
	fmt.Fprintf(os.Stderr, T("trilha-spec mcp %s · %s\ntools: %s\n"), version, st.Layout.Root, strings.Join(s.Tools(), ", "))
	if !*write {
		fmt.Fprint(os.Stderr, T("read-only; pass --write to offer trilha_move, trilha_evidence and trilha_verify\n"))
	}
	return s.ServeStdio(context.Background(), os.Stdin, os.Stdout)
}

// pauseReason is the reason and the moment, for a person.
func pauseReason(p *spec.Project) string {
	reason := p.PauseReason
	if reason == "" {
		reason = "(no reason given)"
	}
	return reason + " (since " + p.PausedAt + ")"
}

func cmdProject(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New(T("usage: trilha-spec project show | pause [--reason R] | resume | limit <key> <value|->"))
	}
	st, err := store()
	if err != nil {
		return err
	}
	l := st.Layout
	p, err := l.LoadProject()
	if err != nil {
		return err
	}
	switch args[0] {
	case "show":
		fs := flags("project show")
		asJSON := fs.Bool("json", false, "")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *asJSON {
			return printJSON(out, p)
		}
		_, err = out.Write(p.Bytes())
		return err
	case "pause":
		fs := flags("project pause")
		reason := fs.String("reason", "", "why the queue stops")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		p.Pause(*reason)
		if err := l.SaveProject(p); err != nil {
			return err
		}
		fmt.Fprintf(out, T("%s is paused: %s\n"), p.Name, pauseReason(p))
		return nil
	case "resume":
		p.Resume()
		if err := l.SaveProject(p); err != nil {
			return err
		}
		fmt.Fprintf(out, T("%s resumed\n"), p.Name)
		return nil
	case "limit":
		if len(args) != 3 {
			return errors.New(T("usage: trilha-spec project limit <key> <value|->"))
		}
		key, val := args[1], args[2]
		if val == "-" {
			delete(p.Limits, key)
		} else {
			f, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return fmt.Errorf(T("limit %s: %q is not a number"), key, val)
			}
			if p.Limits == nil {
				p.Limits = map[string]float64{}
			}
			p.Limits[key] = f
		}
		if err := l.SaveProject(p); err != nil {
			return err
		}
		fmt.Fprintf(out, T("updated %s\n"), rel(l, l.Project()))
		return nil
	}
	return fmt.Errorf(T("unknown project command %q"), args[0])
}

// loadSigner reads the private key `evidence add --sign-key` names; "" is
// no signer. The key id defaults to the file name without `.key`.
func loadSigner(path, keyID string) (*task.Signer, error) {
	if path == "" {
		if keyID != "" {
			return nil, errors.New(T("--key-id needs --sign-key"))
		}
		return nil, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	priv, err := task.ParsePrivateKey(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if keyID == "" {
		keyID = strings.TrimSuffix(filepath.Base(path), ".key")
	}
	return &task.Signer{KeyID: keyID, Key: priv}, nil
}

// verifyEvidence prints every record with its verdict. The command fails
// when any signature is invalid: an edited record is a finding, not a
// listing. Unsigned records are reported, never failed — signing is opt-in.
func verifyEvidence(l spec.Layout, list []task.Evidence, keysDir string, asJSON bool, out io.Writer) error {
	if keysDir == "" {
		keysDir = l.Keys()
	}
	keys, err := task.LoadKeys(keysDir)
	if err != nil {
		return err
	}
	checked := keys.CheckAll(list)
	invalid := 0
	for _, c := range checked {
		if c.Verdict == task.Invalid {
			invalid++
		}
	}
	if asJSON {
		if err := printJSON(out, checked); err != nil {
			return err
		}
	} else {
		for _, c := range checked {
			mark := "✗"
			if c.Passed {
				mark = "✓"
			}
			verdict := string(c.Verdict)
			if c.Signature != nil {
				verdict += " " + c.Signature.KeyID
			}
			if c.Reason != "" {
				verdict += " (" + c.Reason + ")"
			}
			fmt.Fprintf(out, "#%-3d %s %-8s %-20s %s\n", c.Seq, mark, c.Kind, c.By, verdict)
		}
		fmt.Fprintf(out, T("%d record(s), %d key(s) in %s\n"), len(checked), len(keys), rel(l, keysDir))
	}
	if invalid > 0 {
		return fmt.Errorf(T("%d invalid signature(s)"), invalid)
	}
	return nil
}

func cmdKeygen(args []string, out io.Writer) error {
	fs := flags("keygen")
	outDir := fs.String("out", "", "where the private key goes (default ~/.trilha/keys)")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 || !task.ValidKeyID(pos[0]) {
		return errors.New(T("usage: trilha-spec keygen <key-id> [--out DIR]   (key-id: lowercase words joined by - or .)"))
	}
	id := pos[0]
	st, err := store()
	if err != nil {
		return err
	}
	if *outDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		*outDir = filepath.Join(home, ".trilha", "keys")
	}
	privPath := filepath.Join(*outDir, id+".key")
	pubPath := filepath.Join(st.Layout.Keys(), id+".pub")
	for _, p := range []string{privPath, pubPath} {
		if _, err := os.Stat(p); err == nil {
			return fmt.Errorf(T("%s exists; pick another key id"), p)
		}
	}
	priv, pub, err := task.GenerateKey()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(*outDir, 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(privPath, priv, 0o600); err != nil {
		return err
	}
	if err := os.MkdirAll(st.Layout.Keys(), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(pubPath, pub, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(out, T("private key %s (keep it out of the repository)\npublic key  %s (commit it)\n"), privPath, rel(st.Layout, pubPath))
	return nil
}

// runSummary is a run's cost in one glance: `anthropic/claude-sonnet-5 12345+678 tokens 0.0421 USD`.
func runSummary(e task.Evidence) string {
	var parts []string
	if e.Provider != "" || e.Model != "" {
		parts = append(parts, strings.TrimPrefix(e.Provider+"/"+e.Model, "/"))
	}
	if e.TokensIn != 0 || e.TokensOut != 0 {
		parts = append(parts, fmt.Sprintf("%d+%d tokens", e.TokensIn, e.TokensOut))
	}
	if e.Cost != 0 {
		parts = append(parts, strconv.FormatFloat(e.Cost, 'f', -1, 64)+" "+e.Currency)
	}
	return strings.Join(parts, " ")
}

func cmdDoctor(args []string, out io.Writer) error {
	st, err := store()
	if err != nil {
		return err
	}
	var problems, warns []string
	for _, p := range st.Layout.Doctor() {
		problems = append(problems, doctorMessage(p))
	}
	if _, err := st.List(); err != nil {
		problems = append(problems, err.Error())
	} else if _, err := st.Graph(); err != nil {
		problems = append(problems, err.Error())
	}
	if specs, err := st.Layout.ListSpecs(); err != nil {
		problems = append(problems, err.Error())
	} else {
		for _, p := range st.Layout.CheckSpecs(specs) {
			if p.Warning() {
				warns = append(warns, doctorMessage(p))
			} else {
				problems = append(problems, doctorMessage(p))
			}
		}
	}
	if _, err := agent.List(st.Layout); err != nil {
		problems = append(problems, err.Error())
	}
	// A warning is advice: it is printed, counted, and never fails doctor.
	for _, w := range warns {
		fmt.Fprintln(out, "!", w)
	}
	if len(warns) > 0 {
		fmt.Fprintf(out, T("%d warning(s)\n"), len(warns))
	}
	if len(problems) == 0 {
		fmt.Fprintf(out, T("✓ %s is healthy\n"), st.Layout.Dir())
		return nil
	}
	for _, p := range problems {
		fmt.Fprintln(out, "✗", p)
	}
	return fmt.Errorf(T("%d problem(s)"), len(problems))
}

func rel(l spec.Layout, p string) string {
	if r, err := relPath(l.Root, p); err == nil {
		return r
	}
	return p
}

// splitList cuts a comma-separated flag into its items, blanks dropped.
func splitList(v string) []string {
	var out []string
	for _, it := range strings.Split(v, ",") {
		if it = strings.TrimSpace(it); it != "" {
			out = append(out, it)
		}
	}
	return out
}

// replaceSpec answers all with s in place of the spec of the same ID, so a
// cross-check sees the spec as it is about to be saved.
func replaceSpec(all []*spec.Spec, s *spec.Spec) []*spec.Spec {
	out := make([]*spec.Spec, 0, len(all))
	for _, x := range all {
		if x.ID == s.ID {
			out = append(out, s)
		} else {
			out = append(out, x)
		}
	}
	return out
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
