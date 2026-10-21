package task

import (
	"testing"

	"github.com/emersonjoe/trilha-spec/spec"
)

func TestCheckSpecStatus(t *testing.T) {
	specs := []*spec.Spec{
		{ID: "001-delivered", Status: spec.Approved}, // every task done: warn
		{ID: "002-in-flight", Status: spec.Approved}, // one task still open: quiet
		{ID: "003-empty", Status: spec.Approved},     // no task cut yet: quiet
		{ID: "004-closed", Status: spec.Done},        // already done: quiet
		{ID: "005-draft", Status: spec.Draft},        // not approved: quiet
	}
	tasks := []*Task{
		{ID: "TASK-001", Spec: "001-delivered", Status: Done},
		{ID: "TASK-002", Spec: "001-delivered", Status: Done},
		{ID: "TASK-003", Spec: "002-in-flight", Status: Done},
		{ID: "TASK-004", Spec: "002-in-flight", Status: Review},
		{ID: "TASK-005", Spec: "004-closed", Status: Done},
		{ID: "TASK-006", Spec: "005-draft", Status: Done},
		{ID: "TASK-007", Status: Done}, // no spec: ignored
	}
	got := CheckSpecStatus(specs, tasks)
	if len(got) != 1 {
		t.Fatalf("want exactly one problem, got %v", got)
	}
	p := got[0]
	if p.Code != spec.ProblemSpecTasksDone || p.Arg != "001-delivered" {
		t.Fatalf("got %+v", p)
	}
	if !p.Warning() {
		t.Fatal("spec-tasks-done must be a warning: closing the spec is a human decision")
	}
	if want := "spec 001-delivered: all tasks done, status should be done"; p.String() != want {
		t.Fatalf("String() = %q, want %q", p.String(), want)
	}

	// The spec moves to done: the warning goes away.
	specs[0].Status = spec.Done
	if got := CheckSpecStatus(specs, tasks); len(got) != 0 {
		t.Fatalf("done spec still reported: %v", got)
	}
}
