package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emersonjoe/trilha-spec/spec"
)

// program lays out what a real programme looks like on disk: a manifest and
// two checkouts side by side under it.
func program(t *testing.T) (dir string, here, there *Store) {
	t.Helper()
	dir = t.TempDir()
	Now = func() string { return "2026-09-12T00:00:00Z" }
	mk := func(name string) *Store {
		l, _, err := spec.Init(filepath.Join(dir, name), spec.InitOptions{Name: name})
		if err != nil {
			t.Fatal(err)
		}
		return &Store{Layout: l}
	}
	return dir, mk("app"), mk("trilha")
}

func TestRemoteRefShape(t *testing.T) {
	for _, s := range []string{"TASK-001", "app:TASK-004", "trilha-cloud:TASK-010", "cloud.2026:TASK-001"} {
		if _, ok := ParseRef(s); !ok {
			t.Fatalf("%q refused", s)
		}
	}
	for _, s := range []string{"", "task-1", "App:TASK-004", "app:TASK-4", "app:", ":TASK-004", "a:b:TASK-001"} {
		if _, ok := ParseRef(s); ok {
			t.Fatalf("%q accepted", s)
		}
	}
	r, _ := ParseRef("app:TASK-004")
	if !r.Remote() || r.String() != "app:TASK-004" || r.Waiting() != "waiting:app:TASK-004" {
		t.Fatalf("ref = %+v", r)
	}
	if local, _ := ParseRef("TASK-001"); local.Remote() || local.String() != "TASK-001" {
		t.Fatalf("local ref = %+v", local)
	}
	bad := &Task{ID: "TASK-001", Title: "One", Status: Idea, DependsOn: []string{"App:TASK-4"}}
	if err := bad.Validate(); err == nil || !strings.Contains(err.Error(), "neither a task id nor") {
		t.Fatalf("bad ref accepted: %v", err)
	}
}

func TestRemoteDependencyBlocksUntilAnswered(t *testing.T) {
	dir, here, there := program(t)
	p, err := here.Layout.LoadProject()
	if err != nil {
		t.Fatal(err)
	}
	p.Repos = map[string]string{"trilha": "https://github.com/emersonjoe/trilha"}
	if err := here.Layout.SaveProject(p); err != nil {
		t.Fatal(err)
	}
	if _, err := there.Create("Framework work", func(x *Task) {
		x.Status = Ready
		x.Acceptance = []string{"ok"}
	}); err != nil {
		t.Fatal(err)
	}
	mine, err := here.Create("Product work", func(x *Task) {
		x.Status = Ready
		x.Acceptance = []string{"ok"}
		x.DependsOn = []string{"trilha:TASK-001"}
	})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(here.Layout.TaskFile(mine.ID))
	// A scalar holding a colon is written quoted, which is what keeps it a
	// scalar rather than the start of a block (protocol §2).
	if !strings.Contains(string(b), "depends_on:\n  - \"trilha:TASK-001\"\n") {
		t.Fatalf("remote dependency written as:\n%s", b)
	}

	// Nobody to ask: the task waits, and the reason names the repository.
	g, err := here.Graph()
	if err != nil {
		t.Fatal(err)
	}
	if got := g.Blockers(mine.ID); len(got) != 1 || got[0] != "waiting:trilha:TASK-001" {
		t.Fatalf("blockers = %v", got)
	}
	if len(g.Ready()) != 0 {
		t.Fatal("a task waiting on another repository is not ready")
	}
	if _, err := here.Move(mine.ID, Running); err == nil || !strings.Contains(err.Error(), "waiting:trilha:TASK-001") {
		t.Fatalf("ran an unresolved task: %v", err)
	}

	// A sibling checkout answers: the dependency is real, and not done.
	here.Remote = Checkouts{"trilha": filepath.Join(dir, "trilha")}
	g, _ = here.Graph()
	if got := g.Blockers(mine.ID); len(got) != 1 || got[0] != "trilha:TASK-001" {
		t.Fatalf("blockers = %v", got)
	}
	// Once it is done there, the work can start here.
	for _, to := range []Status{Running, Verify, Review, Done} {
		if _, err := there.Move("TASK-001", to); err != nil {
			t.Fatal(err)
		}
	}
	g, _ = here.Graph()
	if got := g.Blockers(mine.ID); len(got) != 0 {
		t.Fatalf("still blocked by %v", got)
	}
	if ready := g.Ready(); len(ready) != 1 || ready[0].ID != mine.ID {
		t.Fatalf("ready = %+v", ready)
	}
	if _, err := here.Move(mine.ID, Running); err != nil {
		t.Fatalf("could not start: %v", err)
	}
	// A repository the project never declared is a doctor fault.
	tasks, _ := here.List()
	if got := CheckRepos(p, tasks); len(got) != 0 {
		t.Fatalf("a declared repo reported: %+v", got)
	}
	p.Repos = nil
	got := CheckRepos(p, tasks)
	if len(got) != 1 || got[0].Code != spec.ProblemRepoUnknown || got[0].Arg != mine.ID+" trilha:TASK-001" {
		t.Fatalf("doctor = %+v", got)
	}
}

func TestLocalOnlyProjectIsUnchanged(t *testing.T) {
	st := newStore(t)
	a, err := st.Create("One", func(x *Task) { x.Status = Ready; x.Acceptance = []string{"ok"} })
	if err != nil {
		t.Fatal(err)
	}
	b, err := st.Create("Two", func(x *Task) { x.Status = Ready; x.Acceptance = []string{"ok"}; x.DependsOn = []string{a.ID} })
	if err != nil {
		t.Fatal(err)
	}
	g, err := st.Graph()
	if err != nil {
		t.Fatal(err)
	}
	if got := g.Blockers(b.ID); len(got) != 1 || got[0] != a.ID {
		t.Fatalf("blockers = %v", got)
	}
	if ready := g.Ready(); len(ready) != 1 || ready[0].ID != a.ID {
		t.Fatalf("ready = %+v", ready)
	}
	if m := g.Mermaid(); !strings.Contains(m, "TASK_001 --> TASK_002") {
		t.Fatalf("mermaid:\n%s", m)
	}
	// A dependency that does not exist is still a mistake in the files.
	if _, err := NewGraph([]*Task{{ID: "TASK-009", Title: "X", Status: Idea, DependsOn: []string{"TASK-404"}}}); err == nil {
		t.Fatal("missing local dependency accepted")
	}
}

func TestProgramManifestAndGraph(t *testing.T) {
	dir, here, there := program(t)
	manifest := "---\nname: Platform programme\nrepos:\n  app: app\n  trilha: trilha\nmilestones:\n  - id: M2\n    due: 2027-04-30\n---\n\nO programa.\n"
	if err := os.WriteFile(filepath.Join(dir, spec.ProgramFile), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	prog, err := here.Layout.FindProgram()
	if err != nil || prog == nil {
		t.Fatalf("find: %v %v", err, prog)
	}
	if prog.Name != "Platform programme" || len(prog.Milestones) != 1 || prog.Milestones[0].Due != "2027-04-30" {
		t.Fatalf("program = %+v", prog)
	}
	if got := prog.Checkouts(); len(got) != 2 || got["trilha"] == "" {
		t.Fatalf("checkouts = %v", got)
	}
	if got := prog.AliasOf(here.Layout.Root); got != "app" {
		t.Fatalf("alias of this checkout = %q", got)
	}
	if string(prog.Bytes()) != manifest {
		t.Fatalf("not stable:\n%s", prog.Bytes())
	}
	if _, err := there.Create("Framework work", func(x *Task) { x.Status = Ready; x.Acceptance = []string{"ok"} }); err != nil {
		t.Fatal(err)
	}
	if _, err := here.Create("Product work", func(x *Task) {
		x.Status = Ready
		x.Acceptance = []string{"ok"}
		x.DependsOn = []string{"trilha:TASK-001", "cloud:TASK-007"}
	}); err != nil {
		t.Fatal(err)
	}
	g, err := ProgramGraph(here.Layout, prog, Checkouts(prog.Checkouts()))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"subgraph app\n", "subgraph trilha\n",
		`app_TASK_001["TASK-001<br/>Product work"]`,
		`trilha_TASK_001["TASK-001<br/>Framework work"]`,
		"  trilha_TASK_001 --> app_TASK_001\n",
		// A repository nobody checked out is still drawn, as the thing the
		// work waits for.
		"  cloud_TASK_007[\"cloud:TASK-007\"]\n",
		"  cloud_TASK_007 --> app_TASK_001\n",
	} {
		if !strings.Contains(g, want) {
			t.Fatalf("program graph missing %q:\n%s", want, g)
		}
	}
	// A project with no manifest above it answers nil, and nothing breaks.
	lone := newStore(t)
	if p, err := lone.Layout.FindProgram(); err != nil || p != nil {
		t.Fatalf("lone project: %v %+v", err, p)
	}
}
