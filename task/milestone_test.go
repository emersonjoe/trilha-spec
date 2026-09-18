package task

import (
	"os"
	"strings"
	"testing"

	"github.com/emersonjoe/trilha-spec/spec"
)

func withMilestones(t *testing.T, l spec.Layout, ms ...spec.Milestone) *spec.Project {
	t.Helper()
	p, err := l.LoadProject()
	if err != nil {
		t.Fatal(err)
	}
	p.Milestones = ms
	if err := l.SaveProject(p); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestMilestoneRoundTrip(t *testing.T) {
	st := newStore(t)
	l := st.Layout
	withMilestones(t, l,
		spec.Milestone{ID: "M1", Title: "Descoberta", Due: "2027-01-31"},
		spec.Milestone{ID: "M2", Title: "PoC", Due: "2027-04-30", Gate: "aceite do cliente"})
	b, _ := os.ReadFile(l.Project())
	want := "milestones:\n  - id: M1\n    title: Descoberta\n    due: 2027-01-31\n  - id: M2\n    title: PoC\n    due: 2027-04-30\n    gate: aceite do cliente\n"
	if !strings.Contains(string(b), want) {
		t.Fatalf("milestones written as:\n%s", b)
	}
	p, err := l.LoadProject()
	if err != nil || len(p.Milestones) != 2 || p.Milestones[1].Gate != "aceite do cliente" {
		t.Fatalf("round trip: %v %+v", err, p.Milestones)
	}
	if d := p.Due(); d["M2"] != "2027-04-30" || len(d) != 2 {
		t.Fatalf("due = %v", d)
	}
	if p.DeleteMilestone("M1"); len(p.Milestones) != 1 {
		t.Fatalf("delete left %+v", p.Milestones)
	}
	p.Milestones = []spec.Milestone{{ID: "M3", Due: "30/04/2027"}}
	if err := p.Validate(); err == nil || !strings.Contains(err.Error(), "is not a date") {
		t.Fatalf("bad date accepted: %v", err)
	}
}

func TestMilestoneSchedulingAndDoctor(t *testing.T) {
	st := newStore(t)
	l := st.Layout
	p := withMilestones(t, l,
		spec.Milestone{ID: "M1", Due: "2026-01-31"},
		spec.Milestone{ID: "M2", Due: "2027-04-30"},
		spec.Milestone{ID: "M3", Due: "2028-01-01"})
	// A task takes the milestone of its spec when it declares none.
	s := &spec.Spec{ID: "010-poc", Title: "PoC", Status: spec.Approved, Milestone: "M2"}
	if err := l.SaveSpec(s); err != nil {
		t.Fatal(err)
	}
	ready := func(title, milestone, specID string) *Task {
		x, err := st.Create(title, func(x *Task) {
			x.Status = Ready
			x.Acceptance = []string{"ok"}
			x.Milestone = milestone
			x.Spec = specID
		})
		if err != nil {
			t.Fatal(err)
		}
		return x
	}
	ready("Late milestone", "M3", "")     // TASK-001, due 2028
	ready("From its spec", "", "010-poc") // TASK-002, due 2027 through the spec
	ready("Overdue", "M1", "")            // TASK-003, due 2026
	ready("No milestone", "", "")         // TASK-004, no date at all

	byID := SpecsByID([]*spec.Spec{s})
	if got := MilestoneOf(&Task{Spec: "010-poc"}, byID); got != "M2" {
		t.Fatalf("inherited milestone = %q", got)
	}
	if got := MilestoneOf(&Task{Spec: "010-poc", Milestone: "M9"}, byID); got != "M9" {
		t.Fatalf("own milestone must win: %q", got)
	}

	g, err := st.Graph()
	if err != nil {
		t.Fatal(err)
	}
	order := ByDue(g.Ready(), func(x *Task) string { return MilestoneOf(x, byID) }, p.Due())
	var ids []string
	for _, x := range order {
		ids = append(ids, x.ID)
	}
	if strings.Join(ids, ",") != "TASK-003,TASK-002,TASK-001,TASK-004" {
		t.Fatalf("next order = %v", ids)
	}

	tasks, err := st.List()
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, pb := range CheckMilestones(p, []*spec.Spec{s}, tasks, "2026-09-18") {
		got = append(got, pb.Code+":"+pb.Arg)
	}
	joined := strings.Join(got, " ")
	for _, want := range []string{"milestone-past-due:TASK-003 M1 2026-01-31"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("doctor said %q, wanted %q", joined, want)
		}
	}
	if strings.Contains(joined, "milestone-empty") {
		t.Fatalf("every milestone has a task: %s", joined)
	}
	// A milestone nobody uses, and a task naming one that is not declared.
	p.SetMilestone(spec.Milestone{ID: "M4", Due: "2029-01-01"})
	tasks[3].Milestone = "M9"
	got = nil
	for _, pb := range CheckMilestones(p, []*spec.Spec{s}, tasks, "2026-09-18") {
		got = append(got, pb.Code+":"+pb.Arg)
	}
	joined = strings.Join(got, " ")
	for _, want := range []string{"milestone-empty:M4", "milestone-unknown:TASK-004 M9"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("doctor said %q, wanted %q", joined, want)
		}
	}
}
