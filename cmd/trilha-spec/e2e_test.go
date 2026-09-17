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

func TestUsage(t *testing.T) {
	if out, err := cli(t, t.TempDir()); err == nil || !strings.Contains(out, "usage:") {
		t.Fatalf("no args:\n%s", out)
	}
	if out := must(t, t.TempDir(), "version"); !strings.HasPrefix(out, "trilha-spec 0.") {
		t.Fatalf("version:\n%s", out)
	}
}
