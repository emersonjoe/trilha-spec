package ai

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emersonjoe/trilha-spec/spec"
	"github.com/emersonjoe/trilha-spec/task"
)

func TestBuild(t *testing.T) {
	l, _, err := spec.Init(t.TempDir(), spec.InitOptions{Name: "demo", Description: "a demo"})
	if err != nil {
		t.Fatal(err)
	}
	st := &task.Store{Layout: l}
	sp := spec.NewSpecDoc("001-oauth", "OAuth", "en", "")
	sp.Assets = []string{"session cookie"}
	sp.Controls = []string{"ASVS V4.1"}
	sp.Evidence = []string{"go test ./internal/auth/..."}
	if err := l.SaveSpec(sp); err != nil {
		t.Fatal(err)
	}
	dep, _ := st.Create("Base", func(x *task.Task) { x.Acceptance = []string{"x"} })
	tk, _ := st.Create("Login", func(x *task.Task) {
		x.Spec = "001-oauth"
		x.DependsOn = []string{dep.ID}
		x.Acceptance = []string{"OAuth login works"}
		x.Checks = []string{"go test ./..."}
		x.Body = "Use the provider in context."
	})
	os.WriteFile(filepath.Join(l.Context(), "arch.md"), []byte("# Arch\nmonolith"), 0o644)
	pr, _ := l.LoadProject()
	pr.Limits = map[string]float64{spec.LimitMaxCostPerHour: 5, spec.LimitMaxRepeatedFailureClass: 3}
	if err := l.SaveProject(pr); err != nil {
		t.Fatal(err)
	}
	task.Record(l, task.Evidence{Task: tk.ID, Kind: "note", By: "me", Note: "started"})
	p, err := Build(l, tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.Spec == nil || p.Agent == nil || p.Agent.Name != "coder" || len(p.Dependencies) != 1 || len(p.Evidence) != 1 {
		t.Fatalf("pack = %+v", p)
	}
	md := p.Markdown()
	for _, want := range []string{"# Task TASK-002 — Login", "### Limits\n\n- max_cost_per_hour: 5\n- max_repeated_failure_class: 3\n", "## Constitution", "## You are `coder`", "## Specification 001-oauth", "### Assets touched\n\n- session cookie", "### Controls affected\n\n- ASVS V4.1", "### Evidence the reviewer must see\n\n- `go test ./internal/auth/...`", "- OAuth login works", "- `go test ./...`", "TASK-001 — Base (idea)", "#1 note by me: started", "## Context: arch.md", "monolith"} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown lacks %q:\n%s", want, md)
		}
	}
	var out map[string]any
	if err := json.Unmarshal(p.JSON(), &out); err != nil || out["task"].(map[string]any)["id"] != "TASK-002" {
		t.Fatalf("json: %v", err)
	}
	if got := out["spec"].(map[string]any)["controls"].([]any); len(got) != 1 || got[0] != "ASVS V4.1" {
		t.Fatalf("json spec controls: %v", got)
	}
	if got := out["project"].(map[string]any)["limits"].(map[string]any); got["max_cost_per_hour"] != 5.0 {
		t.Fatalf("json limits: %v", got)
	}
	// A paused project is the first thing the agent reads after the project.
	pr.Pause("manual")
	l.SaveProject(pr)
	p, _ = Build(l, tk.ID)
	if !strings.Contains(p.Markdown(), "**The project is paused**: manual (since ") {
		t.Fatalf("paused pack:\n%s", p.Markdown())
	}
	if _, err := Build(l, "TASK-999"); err == nil {
		t.Fatal("missing task accepted")
	}
}
