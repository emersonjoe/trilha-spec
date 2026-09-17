package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var bin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "trilha-spec-e2e")
	if err != nil {
		panic(err)
	}
	bin = filepath.Join(dir, "trilha-spec")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		panic(string(out))
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func cli(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	return cliEnv(t, dir, nil, args...)
}

// cliEnv runs the binary with extra environment entries (KEY=VALUE).
func cliEnv(t *testing.T, dir string, env []string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func must(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := cli(t, dir, args...)
	if err != nil {
		t.Fatalf("trilha-spec %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return out
}

func TestLifecycle(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh in checks")
	}
	dir := t.TempDir()
	if _, err := cli(t, dir, "task", "list"); err == nil {
		t.Fatal("worked without init")
	}
	out := must(t, dir, "init", "--name", "demo", "--description", "a demo")
	if !strings.Contains(out, "created .trilha/project.md") {
		t.Fatalf("init:\n%s", out)
	}
	must(t, dir, "doctor")
	must(t, dir, "spec", "new", "OAuth login")
	if out := must(t, dir, "spec", "list"); !strings.Contains(out, "001-oauth-login") {
		t.Fatalf("spec list:\n%s", out)
	}
	must(t, dir, "task", "add", "Provider config", "--spec", "001-oauth-login", "--status", "ready", "--accept", "config loads", "--check", `sh -c "test -f provider.txt"`)
	must(t, dir, "task", "add", "Login page", "--depends", "TASK-001", "--status", "ready", "--accept", "login works", "--check", "true")
	if out, err := cli(t, dir, "task", "add", "Loop", "--depends", "TASK-999"); err == nil {
		t.Fatalf("missing dependency accepted:\n%s", out)
	}
	if out := must(t, dir, "task", "list"); !strings.Contains(out, "TASK-002   ready    Login page") || !strings.Contains(out, "TASK-001\n") {
		t.Fatalf("task list:\n%s", out)
	}
	if out := must(t, dir, "task", "next"); !strings.HasPrefix(out, "TASK-001  Provider config") || strings.Contains(out, "TASK-002") {
		t.Fatalf("next:\n%s", out)
	}
	if out, err := cli(t, dir, "task", "move", "TASK-002", "running"); err == nil || !strings.Contains(out, "waiting on TASK-001") {
		t.Fatalf("ran blocked task:\n%s", out)
	}
	must(t, dir, "task", "move", "TASK-001", "running")
	if out := must(t, dir, "context", "TASK-001"); !strings.Contains(out, "## You are `coder`") || !strings.Contains(out, "config loads") {
		t.Fatalf("context:\n%s", out)
	}
	must(t, dir, "task", "move", "TASK-001", "verify")
	// The check fails: the task goes to failed, with the evidence recorded.
	if out, err := cli(t, dir, "verify", "TASK-001"); err == nil || !strings.Contains(out, "TASK-001 is now failed") {
		t.Fatalf("verify should fail:\n%s", out)
	}
	os.WriteFile(filepath.Join(dir, "provider.txt"), []byte("ok"), 0o644)
	must(t, dir, "task", "move", "TASK-001", "ready")
	must(t, dir, "task", "move", "TASK-001", "running")
	must(t, dir, "task", "move", "TASK-001", "verify")
	if out := must(t, dir, "verify", "TASK-001"); !strings.Contains(out, "TASK-001 is now review") {
		t.Fatalf("verify:\n%s", out)
	}
	must(t, dir, "evidence", "TASK-001", "add", "--note", "reviewed by hand", "--by", "reviewer")
	if out := must(t, dir, "evidence", "TASK-001"); !strings.Contains(out, "#3   ✓ note     reviewer") {
		t.Fatalf("evidence:\n%s", out)
	}
	must(t, dir, "task", "move", "TASK-001", "done")
	if out := must(t, dir, "task", "next"); !strings.HasPrefix(out, "TASK-002") {
		t.Fatalf("next after done:\n%s", out)
	}
	if out := must(t, dir, "task", "graph"); !strings.Contains(out, "TASK_001 --> TASK_002") {
		t.Fatalf("graph:\n%s", out)
	}
	if out := must(t, dir, "agent", "list"); !strings.Contains(out, "coder") || !strings.Contains(out, "reviewer") {
		t.Fatalf("agents:\n%s", out)
	}
	if out := must(t, dir, "task", "next", "--json"); !strings.Contains(out, `"id": "TASK-002"`) {
		t.Fatalf("json:\n%s", out)
	}
	// Every protocol file is committable: the .gitignore only hides runner state.
	gi, _ := os.ReadFile(filepath.Join(dir, ".trilha", ".gitignore"))
	if strings.Contains(string(gi), "\n*\n") || !strings.Contains(string(gi), "runs/") {
		t.Fatalf("gitignore:\n%s", gi)
	}
}

// TRILHA_LANG=pt translates what the operator reads; --json and the files
// are the protocol and do not move.
func TestPortugueseMessages(t *testing.T) {
	dir := t.TempDir()
	pt := []string{"TRILHA_LANG=pt-BR"}
	out, err := cliEnv(t, dir, pt, "task", "list")
	if err == nil || !strings.Contains(out, "erro:") {
		t.Fatalf("error prefix:\n%s", out)
	}
	if out, _ := cliEnv(t, dir, pt, "--help"); !strings.Contains(out, "uso: trilha-spec <comando>") {
		t.Fatalf("help:\n%s", out)
	}
	must(t, dir, "init", "--name", "demo")
	if out, err := cliEnv(t, dir, pt, "task", "add", "Uma task", "--status", "ready", "--accept", "ok"); err != nil || !strings.Contains(out, "criado TASK-001") {
		t.Fatalf("task add: %v\n%s", err, out)
	}
	if out, _ := cliEnv(t, dir, pt, "task", "list"); !strings.Contains(out, "TÍTULO") || !strings.Contains(out, "AGUARDANDO") {
		t.Fatalf("task list:\n%s", out)
	}
	if out, _ := cliEnv(t, dir, pt, "task", "list", "--json"); !strings.Contains(out, `"status": "ready"`) {
		t.Fatalf("--json must not translate:\n%s", out)
	}
	if out, _ := cliEnv(t, dir, pt, "task", "move", "TASK-001", "running"); !strings.Contains(out, "TASK-001 agora está running") {
		t.Fatalf("move:\n%s", out)
	}
	if out, _ := cliEnv(t, dir, pt, "doctor"); !strings.Contains(out, "está saudável") {
		t.Fatalf("doctor:\n%s", out)
	}
	os.WriteFile(filepath.Join(dir, ".trilha", ".gitignore"), []byte("*\n"), 0o644)
	if out, err := cliEnv(t, dir, pt, "doctor"); err == nil || !strings.Contains(out, "ignora tudo") || !strings.Contains(out, "1 problema(s)") {
		t.Fatalf("doctor pt:\n%s", out)
	}
	if out, _ := cliEnv(t, dir, []string{"TRILHA_LANG=fr"}, "task", "list"); !strings.Contains(out, "TITLE") {
		t.Fatalf("unknown language must fall back to English:\n%s", out)
	}
}

// --body and --body-file write the document body; TRILHA_LANG picks the
// language of what init and spec new write when no body is given.
func TestBodyAndTemplates(t *testing.T) {
	dir := t.TempDir()
	pt := []string{"TRILHA_LANG=pt-BR"}
	if out, err := cliEnv(t, dir, pt, "init", "--name", "demo"); err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	c, _ := os.ReadFile(filepath.Join(dir, ".trilha", "constitution.md"))
	if !strings.Contains(string(c), "# Constituição") {
		t.Fatalf("pt constitution:\n%s", c)
	}
	if out, err := cliEnv(t, dir, pt, "spec", "new", "Login OAuth", "--issue", "7"); err != nil || !strings.Contains(out, "criado 001-login-oauth") {
		t.Fatalf("spec new: %v\n%s", err, out)
	}
	if out := must(t, dir, "spec", "show", "001-login-oauth"); !strings.Contains(out, "issue: 7\n") || !strings.Contains(out, "## Por quê") {
		t.Fatalf("pt spec template:\n%s", out)
	}
	must(t, dir, "task", "add", "Provider config", "--body", "Read `config.yaml`; fail loudly when a key is missing.")
	if out := must(t, dir, "task", "show", "TASK-001"); !strings.Contains(out, "---\n\nRead `config.yaml`; fail loudly") || strings.Contains(out, "expected_files") {
		t.Fatalf("task body:\n%s", out)
	}
	if out := must(t, dir, "task", "show", "TASK-001", "--json"); !strings.Contains(out, `"body": "Read `) {
		t.Fatalf("task body json:\n%s", out)
	}
	bodyFile := filepath.Join(dir, "body.md")
	os.WriteFile(bodyFile, []byte("# From a file\n\nline two\n"), 0o644)
	must(t, dir, "spec", "new", "Second", "--body-file", bodyFile)
	if out := must(t, dir, "spec", "show", "002-second"); !strings.Contains(out, "# From a file\n\nline two\n") || strings.Contains(out, "## Why") {
		t.Fatalf("spec body-file:\n%s", out)
	}
	cmd := exec.Command(bin, "task", "add", "From stdin", "--body-file", "-")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader("piped body\n")
	if out, err := cmd.CombinedOutput(); err != nil || !strings.Contains(string(out), "created TASK-002") {
		t.Fatalf("stdin body: %v\n%s", err, out)
	}
	if out := must(t, dir, "task", "show", "TASK-002"); !strings.Contains(out, "piped body") {
		t.Fatalf("stdin body:\n%s", out)
	}
	if out, err := cli(t, dir, "task", "add", "Both", "--body", "a", "--body-file", bodyFile); err == nil || !strings.Contains(out, "exclusive") {
		t.Fatalf("both flags accepted:\n%s", out)
	}
	if out := must(t, dir, "--help"); !strings.Contains(out, "[--status S]") || !strings.Contains(out, "--body-file PATH") {
		t.Fatalf("help:\n%s", out)
	}
}

// A spec has a life of its own: it is judged, delivered, replaced; the CLI
// moves it and relates it without an editor.
func TestSpecLifecycleCLI(t *testing.T) {
	dir := t.TempDir()
	must(t, dir, "init", "--name", "demo")
	must(t, dir, "spec", "new", "First")
	must(t, dir, "spec", "new", "Second")
	if out, err := cli(t, dir, "spec", "move", "001-first", "done"); err == nil || !strings.Contains(out, "cannot move from draft to done") {
		t.Fatalf("illegal move:\n%s", out)
	}
	if out := must(t, dir, "spec", "move", "001-first", "approved"); !strings.Contains(out, "001-first is now approved") {
		t.Fatalf("move:\n%s", out)
	}
	if out := must(t, dir, "spec", "list", "--status", "approved"); !strings.Contains(out, "001-first") || strings.Contains(out, "002-second") {
		t.Fatalf("list --status:\n%s", out)
	}
	if out, err := cli(t, dir, "spec", "set", "002-second", "--supersedes", "009-nope"); err == nil || !strings.Contains(out, "spec reference does not exist: 002-second supersedes 009-nope") {
		t.Fatalf("missing reference accepted:\n%s", out)
	}
	must(t, dir, "spec", "set", "002-second", "--issue", "42", "--supersedes", "001-first")
	if out := must(t, dir, "spec", "show", "002-second"); !strings.Contains(out, "issue: 42\n") || !strings.Contains(out, "supersedes:\n  - 001-first\n") {
		t.Fatalf("set:\n%s", out)
	}
	must(t, dir, "spec", "move", "001-first", "superseded")
	must(t, dir, "doctor")
	// Drop the relation: the superseded spec is now an orphan and doctor says so.
	must(t, dir, "spec", "set", "002-second", "--issue", "-")
	if out := must(t, dir, "spec", "show", "002-second", "--json"); strings.Contains(out, `"issue"`) || !strings.Contains(out, `"supersedes"`) {
		t.Fatalf("issue not cleared:\n%s", out)
	}
	p := filepath.Join(dir, ".trilha", "specs", "002-second.md")
	b, _ := os.ReadFile(p)
	os.WriteFile(p, []byte(strings.Replace(string(b), "supersedes:\n  - 001-first\n", "", 1)), 0o644)
	if out, err := cli(t, dir, "doctor"); err == nil || !strings.Contains(out, "001-first is superseded but no spec names it") {
		t.Fatalf("doctor:\n%s", out)
	}
}

func TestSpecSecurityCLI(t *testing.T) {
	dir := t.TempDir()
	must(t, dir, "init", "--name", "demo")
	must(t, dir, "spec", "new", "Login", "--asset", "session cookie", "--control", "ASVS V4.1",
		"--evidence", "go test ./internal/auth/...", "--evidence", `sh -c "curl -sf localhost:3000/login | grep -q Sign"`)
	out := must(t, dir, "spec", "show", "001-login")
	for _, want := range []string{"assets:\n  - session cookie\n", "controls:\n  - ASVS V4.1\n", "evidence:\n  - go test ./internal/auth/...\n  - \"sh -c \\\"curl -sf localhost:3000/login | grep -q Sign\\\"\"\n"} {
		if !strings.Contains(out, want) {
			t.Fatalf("show lacks %q:\n%s", want, out)
		}
	}
	// set replaces the list it is given and keeps the others.
	must(t, dir, "spec", "set", "001-login", "--boundary", "browser → api", "--control", "ASVS V3.4")
	out = must(t, dir, "spec", "show", "001-login", "--json")
	for _, want := range []string{`"assets": [`, `"trust_boundaries": [`, `"ASVS V3.4"`, `"evidence": [`} {
		if !strings.Contains(out, want) {
			t.Fatalf("json lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "ASVS V4.1") {
		t.Fatalf("--control did not replace:\n%s", out)
	}
	// The pack hands the impact to the agent next to acceptance and checks.
	must(t, dir, "task", "add", "Form", "--spec", "001-login", "--status", "ready", "--accept", "ok")
	if out := must(t, dir, "context", "TASK-001"); !strings.Contains(out, "### Trust boundaries\n\n- browser → api") || !strings.Contains(out, "### Evidence the reviewer must see\n\n- `go test ./internal/auth/...`") {
		t.Fatalf("context:\n%s", out)
	}
	// An approved spec without any of it is a doctor warning, not a failure.
	must(t, dir, "spec", "new", "Bare")
	must(t, dir, "spec", "move", "002-bare", "approved")
	if out := must(t, dir, "doctor"); !strings.Contains(out, "! spec 002-bare is approved but declares no security impact") || !strings.Contains(out, "1 warning(s)") || !strings.Contains(out, "is healthy") {
		t.Fatalf("doctor:\n%s", out)
	}
	if out, err := cliEnv(t, dir, []string{"TRILHA_LANG=pt"}, "doctor"); err != nil || !strings.Contains(out, "não declara impacto de segurança") {
		t.Fatalf("doctor pt: %v\n%s", err, out)
	}
	must(t, dir, "spec", "set", "002-bare", "--evidence", "go vet ./...")
	if out := must(t, dir, "doctor"); strings.Contains(out, "warning") {
		t.Fatalf("doctor still warns:\n%s", out)
	}
}

func TestUsage(t *testing.T) {
	if out, err := cli(t, t.TempDir()); err == nil || !strings.Contains(out, "usage:") {
		t.Fatalf("no args:\n%s", out)
	}
	if out := must(t, t.TempDir(), "version"); !strings.HasPrefix(out, "trilha-spec 0.") {
		t.Fatalf("version:\n%s", out)
	}
}
