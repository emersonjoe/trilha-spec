package task

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/emersonjoe/trilha-spec/spec"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	root := t.TempDir()
	l, _, err := spec.Init(root, spec.InitOptions{Name: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	Now = func() string { return "2026-09-12T00:00:00Z" }
	return &Store{Layout: l}
}

func TestParseKeepsUnknownFields(t *testing.T) {
	src := "---\nid: TASK-001\ntitle: One\nstatus: idea\npriority: high\nlabels:\n  - a\n---\n\nbody\n"
	tk, err := Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	out := string(tk.Bytes())
	if !strings.Contains(out, "priority: high") || !strings.Contains(out, "labels:\n  - a") {
		t.Fatalf("lost fields:\n%s", out)
	}
	if !strings.HasPrefix(out, "---\nid: TASK-001\ntitle: One\nstatus: idea\n") {
		t.Fatalf("protocol fields not first:\n%s", out)
	}
}

func TestRetryOfRequiresRunID(t *testing.T) {
	task := &Task{ID: "TASK-001", Title: "Repair checkout", Status: Idea, RetryOf: "run-000006"}
	if err := task.Validate(); err != nil {
		t.Fatalf("valid run lineage rejected: %v", err)
	}

	for _, retryOf := range []string{"TASK-006", "run-6", "run-00000x"} {
		task.RetryOf = retryOf
		if err := task.Validate(); err == nil || !strings.Contains(err.Error(), "retry_of must be a run id") {
			t.Fatalf("retry_of %q accepted: %v", retryOf, err)
		}
	}
}

func TestTransitions(t *testing.T) {
	tk := &Task{ID: "TASK-001", Title: "x", Status: Idea}
	if err := tk.Move(Done); err == nil {
		t.Fatal("idea → done accepted")
	}
	for _, s := range []Status{Spec, Ready} {
		tk.Acceptance = []string{"works"}
		if err := tk.Move(s); err != nil {
			t.Fatal(err)
		}
	}
	tk.Acceptance = nil
	if err := tk.Move(Running); err == nil {
		t.Fatal("running without acceptance accepted")
	}
	if err := tk.Move(Status("nope")); err == nil {
		t.Fatal("unknown status accepted")
	}
}

func TestStoreAndGraph(t *testing.T) {
	s := newStore(t)
	a, err := s.Create("First", func(x *Task) { x.Acceptance = []string{"ok"} })
	if err != nil {
		t.Fatal(err)
	}
	b, _ := s.Create("Second", func(x *Task) { x.DependsOn = []string{a.ID}; x.Acceptance = []string{"ok"} })
	c, _ := s.Create("Third", func(x *Task) { x.DependsOn = []string{a.ID, b.ID}; x.Acceptance = []string{"ok"} })
	if a.ID != "TASK-001" || b.ID != "TASK-002" || c.ID != "TASK-003" {
		t.Fatalf("ids %s %s %s", a.ID, b.ID, c.ID)
	}
	for _, id := range []string{a.ID, b.ID, c.ID} {
		if _, err := s.Move(id, Ready); err != nil {
			t.Fatal(err)
		}
	}
	g, err := s.Graph()
	if err != nil {
		t.Fatal(err)
	}
	if got := g.Order(); strings.Join(got, ",") != "TASK-001,TASK-002,TASK-003" {
		t.Fatalf("order %v", got)
	}
	if n := g.Next(); n == nil || n.ID != a.ID {
		t.Fatalf("next = %v", n)
	}
	if _, err := s.Move(b.ID, Running); err == nil || !strings.Contains(err.Error(), "waiting on TASK-001") {
		t.Fatalf("ran with open deps: %v", err)
	}
	for _, st := range []Status{Running, Verify, Review, Done} {
		if _, err := s.Move(a.ID, st); err != nil {
			t.Fatal(err)
		}
	}
	g, _ = s.Graph()
	if n := g.Next(); n == nil || n.ID != b.ID {
		t.Fatalf("next = %v", n)
	}
	if bl := g.Blockers(c.ID); strings.Join(bl, ",") != "TASK-002" {
		t.Fatalf("blockers %v", bl)
	}
	if m := g.Mermaid(); !strings.Contains(m, "TASK_001 --> TASK_002") {
		t.Fatalf("mermaid:\n%s", m)
	}
	if d := g.DOT(); !strings.Contains(d, `"TASK-002" -> "TASK-003"`) {
		t.Fatalf("dot:\n%s", d)
	}
	// A cycle is refused with the members named.
	a2, _ := s.Get(a.ID)
	a2.DependsOn = []string{c.ID}
	if err := s.Save(a2); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Graph(); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("cycle accepted: %v", err)
	}
	// A dependency that does not exist is refused.
	a2.DependsOn = []string{"TASK-999"}
	s.Save(a2)
	if _, err := s.Graph(); err == nil || !strings.Contains(err.Error(), "TASK-999") {
		t.Fatalf("missing dep accepted: %v", err)
	}
}

func TestFileNameMustMatchID(t *testing.T) {
	s := newStore(t)
	os.WriteFile(filepath.Join(s.Layout.Tasks(), "TASK-007.md"), []byte("---\nid: TASK-001\ntitle: x\n---\n"), 0o644)
	if _, err := s.List(); err == nil || !strings.Contains(err.Error(), "disagree") {
		t.Fatalf("err = %v", err)
	}
}

func TestSplitCommand(t *testing.T) {
	got := SplitCommand(`sh -c "go test ./... | tail -1"  extra`)
	want := []string{"sh", "-c", "go test ./... | tail -1", "extra"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %q", got)
	}
	if got := SplitCommand(`sh -c 'echo "hi there"'`); len(got) != 3 || got[2] != `echo "hi there"` {
		t.Fatalf("single quotes: %q", got)
	}
	if len(SplitCommand("   ")) != 0 {
		t.Fatal("blank not empty")
	}
}

func TestRunEvidenceCost(t *testing.T) {
	s := newStore(t)
	tk, _ := s.Create("Run", func(x *Task) { x.Acceptance = []string{"x"} })
	e, p, err := Record(s.Layout, Evidence{Task: tk.ID, Kind: "run", By: "runner", Passed: true,
		Provider: "anthropic", Model: "claude-sonnet-5", TokensIn: 12345, TokensOut: 678, Cost: 0.0421, Currency: "USD", Meta: map[string]string{"run": "run-000001"}})
	if err != nil || !strings.HasSuffix(p, "001-run.json") {
		t.Fatalf("%v %s", err, p)
	}
	b, _ := os.ReadFile(p)
	for _, want := range []string{`"provider": "anthropic"`, `"model": "claude-sonnet-5"`, `"tokens_in": 12345`, `"tokens_out": 678`, `"cost": 0.0421`, `"currency": "USD"`, `"meta"`} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("record lacks %s:\n%s", want, b)
		}
	}
	list, _ := ListEvidence(s.Layout, tk.ID)
	if list[0].Cost != e.Cost || list[0].TokensIn != 12345 {
		t.Fatalf("round trip: %+v", list[0])
	}
	// A note carries none of it, so the fields stay off the file.
	_, p, _ = Record(s.Layout, Evidence{Task: tk.ID, Kind: "note", By: "me", Note: "hi", Passed: true})
	if b, _ := os.ReadFile(p); strings.Contains(string(b), "tokens") || strings.Contains(string(b), "cost") {
		t.Fatalf("note carries cost:\n%s", b)
	}
	for _, bad := range []Evidence{
		{Task: tk.ID, Kind: "run", Cost: 1},
		{Task: tk.ID, Kind: "run", Cost: 1, Currency: "usd"},
		{Task: tk.ID, Kind: "run", Cost: -1, Currency: "USD"},
		{Task: tk.ID, Kind: "run", TokensIn: -1},
	} {
		if _, _, err := Record(s.Layout, bad); err == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
}

func TestVerifyRecordsEvidence(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	s := newStore(t)
	tk, _ := s.Create("Checked", func(x *Task) {
		x.Acceptance = []string{"prints ok"}
		x.Checks = []string{`sh -c "echo ok"`, `sh -c "echo bad; exit 3"`}
	})
	v, err := RunChecks(context.Background(), s.Layout, tk, s.Layout.Root, "test")
	if err != nil {
		t.Fatal(err)
	}
	if v.Passed || len(v.Evidence) != 2 {
		t.Fatalf("v = %+v", v)
	}
	if !v.Evidence[0].Passed || v.Evidence[1].ExitCode != 3 || !strings.Contains(v.Evidence[1].Output, "bad") {
		t.Fatalf("evidence = %+v", v.Evidence)
	}
	if v.Evidence[0].OutputSHA256 == "" || v.Evidence[0].Seq != 1 || v.Evidence[1].Seq != 2 {
		t.Fatalf("evidence = %+v", v.Evidence)
	}
	list, err := ListEvidence(s.Layout, tk.ID)
	if err != nil || len(list) != 2 || !strings.HasSuffix(v.Paths[0], "001-check.json") {
		t.Fatalf("list = %v (%v) paths %v", list, err, v.Paths)
	}
	// No checks at all is a failed verification with a note saying why.
	empty, _ := s.Create("Empty", func(x *Task) { x.Acceptance = []string{"x"} })
	v, err = RunChecks(context.Background(), s.Layout, empty, s.Layout.Root, "test")
	if err != nil || v.Passed || len(v.Paths) != 1 || !strings.HasSuffix(v.Paths[0], "001-note.json") {
		t.Fatalf("v = %+v (%v)", v, err)
	}
	// Project-level verify commands run on every task.
	os.WriteFile(s.Layout.Project(), []byte("---\nname: demo\nverify:\n  - \"sh -c true\"\n---\n"), 0o644)
	v, err = RunChecks(context.Background(), s.Layout, empty, s.Layout.Root, "test")
	if err != nil || !v.Passed || len(v.Evidence) != 1 {
		t.Fatalf("v = %+v (%v)", v, err)
	}
}
