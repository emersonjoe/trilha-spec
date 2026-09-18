package task

import (
	"os"
	"strings"
	"testing"

	"github.com/emersonjoe/trilha-spec/spec"
)

// writeSpec saves a specification with the requirements it declares.
func writeSpec(t *testing.T, l spec.Layout, id string, reqs ...spec.Requirement) *spec.Spec {
	t.Helper()
	s := &spec.Spec{ID: id, Title: id, Status: spec.Approved, Requirements: reqs}
	if err := l.SaveSpec(s); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestRequirementRoundTrip(t *testing.T) {
	st := newStore(t)
	l := st.Layout
	writeSpec(t, l, "009-edital",
		spec.Requirement{ID: "D2-R8", Source: "cp-01-2026#anexo-I", Text: "informar o cidadao"},
		spec.Requirement{ID: "D2-R9", Text: "prazo: 24h"})
	b, _ := os.ReadFile(l.SpecFile("009-edital"))
	want := "requirements:\n  - id: D2-R8\n    source: \"cp-01-2026#anexo-I\"\n    text: informar o cidadao\n  - id: D2-R9\n    text: \"prazo: 24h\"\n"
	if !strings.Contains(string(b), want) {
		t.Fatalf("requirements written as:\n%s", b)
	}
	got, err := l.LoadSpec("009-edital")
	if err != nil || len(got.Requirements) != 2 || got.Requirements[0].Source != "cp-01-2026#anexo-I" || got.Requirements[1].Text != "prazo: 24h" {
		t.Fatalf("round trip: %v %+v", err, got.Requirements)
	}
	// Dropping them drops the key.
	got.Requirements = nil
	if err := l.SaveSpec(got); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(l.SpecFile("009-edital")); strings.Contains(string(b), "requirements") {
		t.Fatalf("empty requirements written:\n%s", b)
	}
	// An id that would not survive `covers: [A, B]` is refused.
	bad := &spec.Spec{ID: "010-b", Title: "B", Status: spec.Draft, Requirements: []spec.Requirement{{ID: "D2 R8"}}}
	if err := bad.Validate(); err == nil || !strings.Contains(err.Error(), "holds a space or a comma") {
		t.Fatalf("id with a space accepted: %v", err)
	}
	dup := &spec.Spec{ID: "010-b", Title: "B", Status: spec.Draft, Requirements: []spec.Requirement{{ID: "R1"}, {ID: "R1"}}}
	if err := dup.Validate(); err == nil || !strings.Contains(err.Error(), "declared twice") {
		t.Fatalf("duplicate id accepted: %v", err)
	}
}

func TestCoverageMatrix(t *testing.T) {
	st := newStore(t)
	l := st.Layout
	writeSpec(t, l, "009-edital",
		spec.Requirement{ID: "D2-R8", Text: "informar o cidadao"},
		spec.Requirement{ID: "D2-R9", Text: "prazo"})
	writeSpec(t, l, "010-outro", spec.Requirement{ID: "D1-R1", Text: "outro"})
	one, err := st.Create("Notify", func(x *Task) {
		x.Status = Ready
		x.Acceptance = []string{"ok"}
		x.Covers = []string{"D2-R8"}
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Record(l, Evidence{Task: one.ID, Kind: "note", By: "ana", Note: "done by hand"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Create("Stray", func(x *Task) { x.Covers = []string{"D9-R9"} }); err != nil {
		t.Fatal(err)
	}
	rows, err := Cover(l, "009-edital")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %+v", rows)
	}
	if !rows[0].Covered() || rows[0].Tasks[0].ID != "TASK-001" || rows[0].Tasks[0].Evidence != 1 {
		t.Fatalf("first row = %+v", rows[0])
	}
	if rows[0].Done() {
		t.Fatal("a ready task should not read as delivered")
	}
	if rows[1].Covered() {
		t.Fatalf("D2-R9 covered by %+v", rows[1].Tasks)
	}
	if all, err := Cover(l, ""); err != nil || len(all) != 3 {
		t.Fatalf("whole project: %v %d", err, len(all))
	}

	specs, err := l.ListSpecs()
	if err != nil {
		t.Fatal(err)
	}
	tasks, err := st.List()
	if err != nil {
		t.Fatal(err)
	}
	var codes []string
	for _, p := range CheckRequirements(specs, tasks) {
		codes = append(codes, p.Code+":"+p.Arg)
	}
	got := strings.Join(codes, " ")
	for _, want := range []string{"requirement-unknown:TASK-002 D9-R9", "requirement-uncovered:009-edital D2-R9", "requirement-uncovered:010-outro D1-R1"} {
		if !strings.Contains(got, want) {
			t.Fatalf("doctor said %q, wanted %q", got, want)
		}
	}
	if strings.Contains(got, "requirement-uncovered:009-edital D2-R8") {
		t.Fatalf("a covered requirement reported: %s", got)
	}

	// The same id in two specs is a fault: `covers` would be ambiguous.
	writeSpec(t, l, "010-outro", spec.Requirement{ID: "D1-R1"}, spec.Requirement{ID: "D2-R8"})
	specs, _ = l.ListSpecs()
	found := false
	for _, p := range CheckRequirements(specs, tasks) {
		if p.Code == spec.ProblemRequirementDuplicate && p.Arg == "D2-R8 (009-edital, 010-outro)" {
			found = true
		}
	}
	if !found {
		t.Fatal("duplicate across specs not reported")
	}
}
