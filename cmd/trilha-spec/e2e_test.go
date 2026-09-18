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

// needs skips a test when a program it relies on is not on this machine.
func needs(t *testing.T, program string) {
	t.Helper()
	if _, err := exec.LookPath(program); err != nil {
		t.Skipf("%s is not on PATH", program)
	}
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

func TestProjectLimitsAndPauseCLI(t *testing.T) {
	dir := t.TempDir()
	must(t, dir, "init", "--name", "demo")
	must(t, dir, "task", "add", "One", "--status", "ready", "--accept", "ok")
	if out := must(t, dir, "task", "next"); !strings.Contains(out, "TASK-001") {
		t.Fatalf("next:\n%s", out)
	}
	must(t, dir, "project", "limit", "max_cost_per_hour", "5.00")
	must(t, dir, "project", "limit", "max_repeated_failure_class", "3")
	if out, err := cli(t, dir, "project", "limit", "max_failure_rate", "half"); err == nil || !strings.Contains(out, `"half" is not a number`) {
		t.Fatalf("text limit accepted:\n%s", out)
	}
	if out := must(t, dir, "project", "show"); !strings.Contains(out, "limits:\n  max_cost_per_hour: 5\n  max_repeated_failure_class: 3\n") {
		t.Fatalf("show:\n%s", out)
	}
	if out := must(t, dir, "context", "TASK-001"); !strings.Contains(out, "### Limits\n\n- max_cost_per_hour: 5\n") {
		t.Fatalf("context:\n%s", out)
	}
	if out := must(t, dir, "project", "pause", "--reason", "breaker:max_repeated_failure_class"); !strings.Contains(out, "demo is paused: breaker:max_repeated_failure_class (since 20") {
		t.Fatalf("pause:\n%s", out)
	}
	if out := must(t, dir, "task", "next"); !strings.Contains(out, "project is paused: breaker:max_repeated_failure_class (since ") || strings.Contains(out, "TASK-001") {
		t.Fatalf("paused next:\n%s", out)
	}
	// --json stays a list — an empty one — and the reason goes to stderr.
	if out := must(t, dir, "task", "next", "--json"); !strings.Contains(out, "\n[]\n") || !strings.Contains(out, "project is paused") || strings.Contains(out, "TASK-001") {
		t.Fatalf("paused next --json:\n%s", out)
	}
	if out := must(t, dir, "project", "show", "--json"); !strings.Contains(out, `"paused": true`) || !strings.Contains(out, `"pause_reason": "breaker:max_repeated_failure_class"`) || !strings.Contains(out, `"max_cost_per_hour": 5`) {
		t.Fatalf("show --json:\n%s", out)
	}
	if out, err := cliEnv(t, dir, []string{"TRILHA_LANG=pt"}, "task", "next"); err != nil || !strings.Contains(out, "projeto pausado: breaker") {
		t.Fatalf("pt: %v\n%s", err, out)
	}
	must(t, dir, "project", "resume")
	if out := must(t, dir, "task", "next"); !strings.Contains(out, "TASK-001") {
		t.Fatalf("resumed next:\n%s", out)
	}
	must(t, dir, "project", "limit", "max_cost_per_hour", "-")
	if out := must(t, dir, "project", "show"); strings.Contains(out, "max_cost_per_hour") || strings.Contains(out, "paused") {
		t.Fatalf("state left:\n%s", out)
	}
}

func TestRunEvidenceCLI(t *testing.T) {
	dir := t.TempDir()
	must(t, dir, "init", "--name", "demo")
	must(t, dir, "task", "add", "One", "--status", "ready", "--accept", "ok")
	if out, err := cli(t, dir, "evidence", "TASK-001", "add", "--run", "--cost", "0.5"); err == nil || !strings.Contains(out, "cost needs a currency") {
		t.Fatalf("cost without currency accepted:\n%s", out)
	}
	must(t, dir, "evidence", "TASK-001", "add", "--run", "--by", "runner", "--provider", "anthropic", "--model", "claude-sonnet-5",
		"--tokens-in", "12345", "--tokens-out", "678", "--cost", "0.0421", "--currency", "USD", "--note", "run-000001")
	if out := must(t, dir, "evidence", "TASK-001"); !strings.Contains(out, "run      runner               anthropic/claude-sonnet-5 12345+678 tokens 0.0421 USD run-000001") {
		t.Fatalf("list:\n%s", out)
	}
	out := must(t, dir, "evidence", "TASK-001", "--json")
	for _, want := range []string{`"kind": "run"`, `"provider": "anthropic"`, `"tokens_in": 12345`, `"tokens_out": 678`, `"cost": 0.0421`, `"currency": "USD"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("json lacks %s:\n%s", want, out)
		}
	}
	if out := must(t, dir, "context", "TASK-001"); !strings.Contains(out, "#1 run by runner: claude-sonnet-5 0.0421 USD — run-000001") {
		t.Fatalf("context:\n%s", out)
	}
	must(t, dir, "evidence", "TASK-001", "add", "--run", "--failed", "--model", "x")
	if out := must(t, dir, "evidence", "TASK-001"); !strings.Contains(out, "#2   ✗ run") {
		t.Fatalf("failed run:\n%s", out)
	}
}

func TestSignedEvidenceCLI(t *testing.T) {
	dir := t.TempDir()
	keys := filepath.Join(t.TempDir(), "private")
	must(t, dir, "init", "--name", "demo")
	must(t, dir, "task", "add", "One", "--status", "ready", "--accept", "ok")
	if out, err := cli(t, dir, "keygen", "Bad Id"); err == nil || !strings.Contains(out, "usage: trilha-spec keygen") {
		t.Fatalf("bad key id accepted:\n%s", out)
	}
	out := must(t, dir, "keygen", "runner-01", "--out", keys)
	if !strings.Contains(out, "private key "+filepath.Join(keys, "runner-01.key")) || !strings.Contains(out, "public key  .trilha/keys/runner-01.pub") {
		t.Fatalf("keygen:\n%s", out)
	}
	// Windows does not carry Unix permission bits, so the mode is only
	// checked where it means something.
	if st, err := os.Stat(filepath.Join(keys, "runner-01.key")); err != nil || (runtime.GOOS != "windows" && st.Mode().Perm() != 0o600) {
		t.Fatalf("private key: %v %v", err, st)
	}
	if out, err := cli(t, dir, "keygen", "runner-01", "--out", keys); err == nil || !strings.Contains(out, "exists; pick another key id") {
		t.Fatalf("overwrote a key:\n%s", out)
	}
	if out, err := cli(t, dir, "evidence", "TASK-001", "add", "--note", "n", "--key-id", "x"); err == nil || !strings.Contains(out, "--key-id needs --sign-key") {
		t.Fatalf("key-id alone accepted:\n%s", out)
	}
	out = must(t, dir, "evidence", "TASK-001", "add", "--run", "--by", "runner-01", "--model", "m", "--cost", "0.1", "--currency", "USD",
		"--sign-key", filepath.Join(keys, "runner-01.key"))
	if !strings.Contains(out, "recorded #1 (.trilha/evidence/TASK-001/001-run.json), signed by runner-01") {
		t.Fatalf("signed add:\n%s", out)
	}
	must(t, dir, "evidence", "TASK-001", "add", "--note", "by hand", "--by", "ana")
	out = must(t, dir, "evidence", "TASK-001", "--verify")
	for _, want := range []string{"#1   ✓ run      runner-01            valid runner-01\n", "#2   ✓ note     ana                  unsigned\n", "2 record(s), 1 key(s) in .trilha/keys\n"} {
		if !strings.Contains(out, want) {
			t.Fatalf("verify lacks %q:\n%s", want, out)
		}
	}
	if out := must(t, dir, "evidence", "TASK-001", "--verify", "--json"); !strings.Contains(out, `"verdict": "valid"`) || !strings.Contains(out, `"key_id": "runner-01"`) || !strings.Contains(out, `"verdict": "unsigned"`) {
		t.Fatalf("verify json:\n%s", out)
	}
	if out := must(t, dir, "context", "TASK-001"); !strings.Contains(out, "#1 run by runner-01: m 0.1 USD · signed by runner-01\n") || !strings.Contains(out, "#2 note by ana: by hand · unverified\n") {
		t.Fatalf("context:\n%s", out)
	}
	// Edit the record after the fact: the signature no longer matches.
	rec := filepath.Join(dir, ".trilha", "evidence", "TASK-001", "001-run.json")
	b, _ := os.ReadFile(rec)
	os.WriteFile(rec, []byte(strings.Replace(string(b), `"passed": true`, `"passed": false`, 1)), 0o644)
	out, err := cli(t, dir, "evidence", "TASK-001", "--verify")
	if err == nil || !strings.Contains(out, "invalid runner-01 (signature does not match the record)") || !strings.Contains(out, "error: 1 invalid signature(s)") {
		t.Fatalf("edited record passed:\n%s", out)
	}
	if out, err := cliEnv(t, dir, []string{"TRILHA_LANG=pt"}, "evidence", "TASK-001", "--verify"); err == nil || !strings.Contains(out, "erro: 1 assinatura(s) inválida(s)") {
		t.Fatalf("pt:\n%s", out)
	}
	// An unknown key directory is an empty keyring: every signature is invalid, none unsigned.
	if out, err := cli(t, dir, "evidence", "TASK-001", "--verify", "--keys", filepath.Join(dir, "nowhere")); err == nil || !strings.Contains(out, "unknown key runner-01") {
		t.Fatalf("empty keyring:\n%s", out)
	}
	// A private key inside .trilha is a doctor problem, in both languages.
	os.WriteFile(filepath.Join(dir, ".trilha", "keys", "runner-01.key"), []byte("x"), 0o600)
	if out, err := cli(t, dir, "doctor"); err == nil || !strings.Contains(out, "private key .trilha/keys/runner-01.key is inside .trilha") {
		t.Fatalf("doctor:\n%s", out)
	}
	if out, err := cliEnv(t, dir, []string{"TRILHA_LANG=pt"}, "doctor"); err == nil || !strings.Contains(out, "a chave privada .trilha/keys/runner-01.key está dentro de .trilha") {
		t.Fatalf("doctor pt:\n%s", out)
	}
	if gi, _ := os.ReadFile(filepath.Join(dir, ".trilha", ".gitignore")); !strings.Contains(string(gi), "keys/*.key") {
		t.Fatalf("gitignore:\n%s", gi)
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

// TestRequirementCoverageCLI walks the traceability matrix end to end: a spec
// declares external requirements, tasks cover them, doctor reports the gaps
// and `spec show --coverage` answers the question a buyer asks.
func TestRequirementCoverageCLI(t *testing.T) {
	dir := t.TempDir()
	must(t, dir, "init", "--name", "edital")
	must(t, dir, "spec", "new", "Desafio 2")
	// Requirements are front matter a person writes; the body stays the body.
	p := filepath.Join(dir, ".trilha", "specs", "001-desafio-2.md")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	withReqs := strings.Replace(string(b), "status: draft\n",
		"status: draft\nrequirements:\n  - id: D2-R8\n    source: cp-01-2026\n    text: informar o cidadao\n  - id: D2-R9\n    text: prazo de 24h\n", 1)
	if err := os.WriteFile(p, []byte(withReqs), 0o644); err != nil {
		t.Fatal(err)
	}
	must(t, dir, "task", "add", "Painel do cidadao", "--spec", "001-desafio-2", "--covers", "D2-R8", "--status", "ready", "--accept", "mostra o andamento")
	if out := must(t, dir, "task", "show", "TASK-001"); !strings.Contains(out, "covers:\n  - D2-R8\n") {
		t.Fatalf("covers not written:\n%s", out)
	}
	out := must(t, dir, "spec", "show", "001-desafio-2", "--coverage")
	if !strings.Contains(out, "D2-R8") || !strings.Contains(out, "TASK-001 (ready, 0 ev)") || !strings.Contains(out, "D2-R9") {
		t.Fatalf("coverage:\n%s", out)
	}
	must(t, dir, "evidence", "TASK-001", "add", "--note", "revisado")
	if out := must(t, dir, "spec", "show", "001-desafio-2", "--coverage", "--json"); !strings.Contains(out, `"evidence": 1`) || !strings.Contains(out, `"id": "D2-R9"`) {
		t.Fatalf("coverage json:\n%s", out)
	}
	// An uncovered requirement is a warning; a task citing an unknown id is a
	// fault that fails doctor.
	out = must(t, dir, "doctor")
	if !strings.Contains(out, "requirement no task covers: 001-desafio-2 D2-R9") {
		t.Fatalf("doctor warning:\n%s", out)
	}
	must(t, dir, "task", "add", "Fora do edital", "--covers", "D9-R9")
	if out, err := cli(t, dir, "doctor"); err == nil || !strings.Contains(out, "task covers a requirement no spec declares: TASK-002 D9-R9") {
		t.Fatalf("unknown requirement accepted:\n%s", out)
	}
	// The context pack hands the agent the words of the requirement.
	if out := must(t, dir, "context", "TASK-001"); !strings.Contains(out, "### Requirements covered") || !strings.Contains(out, "**D2-R8** (cp-01-2026) — informar o cidadao") {
		t.Fatalf("context:\n%s", out)
	}
}

// TestMilestonesCLI puts a calendar on the project: `next` prefers the nearest
// deadline, `task list --milestone` narrows to one, and doctor reports what the
// schedule says.
func TestMilestonesCLI(t *testing.T) {
	dir := t.TempDir()
	must(t, dir, "init", "--name", "programa")
	must(t, dir, "project", "milestone", "M1", "--title", "Descoberta", "--due", "2026-01-31")
	must(t, dir, "project", "milestone", "M2", "--title", "PoC", "--due", "2027-04-30", "--gate", "aceite do cliente")
	must(t, dir, "project", "milestone", "M3", "--due", "2028-01-01")
	if out := must(t, dir, "project", "show"); !strings.Contains(out, "milestones:\n  - id: M1\n    title: Descoberta\n    due: 2026-01-31\n") {
		t.Fatalf("project show:\n%s", out)
	}
	if out, err := cli(t, dir, "project", "milestone", "M4", "--due", "30/04/2027"); err == nil || !strings.Contains(out, "is not a date") {
		t.Fatalf("bad due accepted:\n%s", out)
	}
	must(t, dir, "task", "add", "Piloto", "--milestone", "M3", "--status", "ready", "--accept", "ok")
	must(t, dir, "task", "add", "Entrega", "--milestone", "M2", "--status", "ready", "--accept", "ok")
	must(t, dir, "task", "add", "Levantamento", "--milestone", "M1", "--status", "ready", "--accept", "ok")
	must(t, dir, "task", "add", "Sem data", "--status", "ready", "--accept", "ok")
	// The nearest deadline first; a task with no milestone last.
	out := must(t, dir, "task", "next")
	if want := "TASK-003  Levantamento\nTASK-002  Entrega\nTASK-001  Piloto\nTASK-004  Sem data\n"; out != want {
		t.Fatalf("next:\n%s\nwanted:\n%s", out, want)
	}
	if out := must(t, dir, "task", "list", "--milestone", "M2"); !strings.Contains(out, "TASK-002") || strings.Contains(out, "TASK-001") {
		t.Fatalf("list --milestone:\n%s", out)
	}
	// The context pack carries the milestone and its date.
	if out := must(t, dir, "context", "TASK-002"); !strings.Contains(out, "Milestone: M2, due 2027-04-30 (aceite do cliente)") {
		t.Fatalf("context:\n%s", out)
	}
	if out := must(t, dir, "context", "TASK-002", "--json"); !strings.Contains(out, `"due": "2027-04-30"`) {
		t.Fatalf("context json:\n%s", out)
	}
	// M1 is in the past and its task is not done: a warning, never a failure.
	out = must(t, dir, "doctor")
	if !strings.Contains(out, "task past its milestone's due date: TASK-003 M1 2026-01-31") {
		t.Fatalf("doctor:\n%s", out)
	}
	must(t, dir, "project", "milestone", "M3", "-")
	if out := must(t, dir, "project", "show", "--json"); strings.Contains(out, `"M3"`) {
		t.Fatalf("milestone not removed:\n%s", out)
	}
	// TASK-001 now names a milestone project.md does not declare: a fault.
	if out, err := cli(t, dir, "doctor"); err == nil || !strings.Contains(out, "milestone project.md does not declare: TASK-001 M3") {
		t.Fatalf("unknown milestone accepted:\n%s", out)
	}
}

// TestEvalEvidenceCLI records numbers instead of hiding them in an exit code:
// a harness prints one JSON line per metric, verify turns each into an `eval`,
// and a metric below its threshold fails the verification.
func TestEvalEvidenceCLI(t *testing.T) {
	needs(t, "cat")
	dir := t.TempDir()
	must(t, dir, "init", "--name", "triagem")
	// The harness prints one JSON line per metric; a file keeps the shell out
	// of the way of the quotes.
	os.WriteFile(filepath.Join(dir, "metrics.txt"), []byte("rodando\n"+
		`{"metric":"triage_top1","value":0.87,"threshold":0.85,"comparator":">=","dataset":{"id":"golden-2026","sha256":"abc"}}`+"\n"), 0o644)
	must(t, dir, "task", "add", "Triagem", "--status", "ready",
		"--accept", "metric: triage_top1 >= 0.85", "--accept", "metric: p95_latency <= 5",
		"--check", "cat metrics.txt")
	must(t, dir, "task", "move", "TASK-001", "running")
	must(t, dir, "task", "move", "TASK-001", "verify")
	must(t, dir, "verify", "TASK-001")
	out := must(t, dir, "evidence", "TASK-001")
	if !strings.Contains(out, "✓ eval     trilha-spec verify   triage_top1 0.87 >= 0.85 on golden-2026") {
		t.Fatalf("evidence:\n%s", out)
	}
	// The record is readable: a comparator is `>=`, not an escape.
	raw, err := os.ReadFile(filepath.Join(dir, ".trilha", "evidence", "TASK-001", "002-eval.json"))
	if err != nil || !strings.Contains(string(raw), `"comparator": ">="`) {
		t.Fatalf("record: %v\n%s", err, raw)
	}
	// One gate has no evidence yet: doctor says so once the task is on its way out.
	if out := must(t, dir, "doctor"); !strings.Contains(out, "acceptance metric with no eval evidence: TASK-001 p95_latency") {
		t.Fatalf("doctor:\n%s", out)
	}
	// A metric added by hand, and the context pack showing where the numbers stand.
	must(t, dir, "evidence", "TASK-001", "add", "--eval", "--metric", "p95_latency", "--value", "7", "--unit", "s", "--threshold", "5", "--comparator", "<=", "--by", "harness")
	if out := must(t, dir, "evidence", "TASK-001", "--json"); !strings.Contains(out, `"value": 7`) || !strings.Contains(out, `"comparator": "<="`) {
		t.Fatalf("evidence json:\n%s", out)
	}
	if out := must(t, dir, "context", "TASK-001"); !strings.Contains(out, "### Metrics so far") || !strings.Contains(out, "✗ p95_latency 7 s <= 5") {
		t.Fatalf("context:\n%s", out)
	}
	if out, err := cli(t, dir, "evidence", "TASK-001", "add", "--eval", "--metric", "x_y", "--value", "many"); err == nil || !strings.Contains(out, "is not a number") {
		t.Fatalf("bad value accepted:\n%s", out)
	}
	// A failing gate fails verification even though the command exits 0.
	os.WriteFile(filepath.Join(dir, "slow.txt"), []byte(`{"metric":"p95_latency","value":7,"threshold":5,"comparator":"<="}`+"\n"), 0o644)
	must(t, dir, "task", "add", "Latencia", "--status", "ready", "--accept", "metric: p95_latency <= 5",
		"--check", "cat slow.txt")
	must(t, dir, "task", "move", "TASK-002", "running")
	must(t, dir, "task", "move", "TASK-002", "verify")
	if out, err := cli(t, dir, "verify", "TASK-002"); err == nil || !strings.Contains(out, "TASK-002 is now failed") {
		t.Fatalf("a failing metric must fail verify:\n%s", out)
	}
}

// TestAttestationQuorumCLI closes a task the way a public-sector delivery
// does: two named people, in the roles the task asks for, each signing what
// they attest.
func TestAttestationQuorumCLI(t *testing.T) {
	dir := t.TempDir()
	keys := filepath.Join(t.TempDir(), "private")
	must(t, dir, "init", "--name", "homologacao")
	must(t, dir, "keygen", "ana", "--out", keys)
	must(t, dir, "keygen", "bruno", "--out", keys)
	must(t, dir, "task", "add", "PoC do cidadao", "--status", "ready", "--accept", "o cliente aceita")
	// The policy is front matter a person writes.
	p := filepath.Join(dir, ".trilha", "tasks", "TASK-001.md")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(strings.Replace(string(b), "checks: []\n", "checks: []\nreview:\n  quorum: 2\n  roles: [uat, legal]\n", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if out := must(t, dir, "task", "show", "TASK-001"); !strings.Contains(out, "review:\n  quorum: 2\n  roles:\n    - uat\n    - legal\n") {
		t.Fatalf("policy round trip:\n%s", out)
	}
	for _, to := range []string{"running", "verify", "review"} {
		must(t, dir, "task", "move", "TASK-001", to)
	}
	// The context pack tells the agent what is still owed.
	if out := must(t, dir, "context", "TASK-001"); !strings.Contains(out, "### Human review required") || !strings.Contains(out, "2 signed attestation(s) in roles uat, legal; 2 still missing.") {
		t.Fatalf("context:\n%s", out)
	}
	if out, err := cli(t, dir, "task", "move", "TASK-001", "done"); err == nil || !strings.Contains(out, "needs 2 signed attestation(s) in roles uat, legal") {
		t.Fatalf("closed without attestations:\n%s", out)
	}
	// An unsigned attestation is a claim, and the CLI says so.
	out := must(t, dir, "evidence", "TASK-001", "add", "--attestation", "--by", "Carla", "--role", "uat", "--statement", "parece ok")
	if !strings.Contains(out, "does not count towards a quorum") {
		t.Fatalf("unsigned attestation:\n%s", out)
	}
	must(t, dir, "evidence", "TASK-001", "add", "--attestation", "--by", "Ana Souza", "--role", "uat",
		"--statement", "Homologado com a equipe da prefeitura.", "--ref", "#1", "--sign-key", filepath.Join(keys, "ana.key"))
	if out, err := cli(t, dir, "task", "move", "TASK-001", "done"); err == nil || !strings.Contains(out, "has 1 (ana as uat)") {
		t.Fatalf("one attestation closed a quorum of two:\n%s", out)
	}
	must(t, dir, "evidence", "TASK-001", "add", "--attestation", "--by", "Bruno Lima", "--role", "legal",
		"--statement", "Sem impedimento juridico.", "--sign-key", filepath.Join(keys, "bruno.key"))
	must(t, dir, "task", "move", "TASK-001", "done")
	if out := must(t, dir, "evidence", "TASK-001"); !strings.Contains(out, "✓ attestation Ana Souza            uat: Homologado") {
		t.Fatalf("evidence:\n%s", out)
	}
	if out := must(t, dir, "evidence", "TASK-001", "--verify"); !strings.Contains(out, "valid ana") || !strings.Contains(out, "valid bruno") {
		t.Fatalf("verify:\n%s", out)
	}
	// A quorum with no roles is a fault: any role would satisfy it.
	must(t, dir, "task", "add", "Sem papeis", "--status", "ready", "--accept", "ok")
	p2 := filepath.Join(dir, ".trilha", "tasks", "TASK-002.md")
	b2, _ := os.ReadFile(p2)
	os.WriteFile(p2, []byte(strings.Replace(string(b2), "checks: []\n", "checks: []\nreview:\n  quorum: 1\n", 1)), 0o644)
	if out, err := cli(t, dir, "doctor"); err == nil || !strings.Contains(out, "TASK-002 asks for a review quorum but names no roles") {
		t.Fatalf("doctor:\n%s", out)
	}
}
