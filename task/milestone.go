package task

import (
	"sort"
	"time"

	"github.com/emersonjoe/trilha-spec/spec"
)

// Today is the day the calendar checks are made against, RFC 3339's date
// part. A variable so a test can pin it, like Now.
var Today = func() string { return time.Now().UTC().Format(spec.DueFormat) }

// MilestoneOf answers the milestone a task belongs to: its own, or the one
// of the spec it implements. A task with neither belongs to no milestone,
// which is not a fault — a calendar is optional.
func MilestoneOf(t *Task, specs map[string]*spec.Spec) string {
	if t.Milestone != "" {
		return t.Milestone
	}
	if s := specs[t.Spec]; s != nil {
		return s.Milestone
	}
	return ""
}

// SpecsByID indexes specifications for MilestoneOf.
func SpecsByID(specs []*spec.Spec) map[string]*spec.Spec {
	out := make(map[string]*spec.Spec, len(specs))
	for _, s := range specs {
		out[s.ID] = s
	}
	return out
}

// ByDue reorders tasks so the nearest deadline comes first. It is a stable
// sort, so tasks sharing a due date — and tasks with none, which go last —
// keep the dependency order they arrived in.
func ByDue(tasks []*Task, milestone func(*Task) string, due map[string]string) []*Task {
	out := append([]*Task(nil), tasks...)
	key := func(t *Task) string {
		if d := due[milestone(t)]; d != "" {
			return d
		}
		return "9999-12-31"
	}
	sort.SliceStable(out, func(i, j int) bool { return key(out[i]) < key(out[j]) })
	return out
}

// CheckMilestones answers what is wrong with the calendar: a task or a spec
// naming a milestone the project does not declare (a fault — a typo silently
// drops work out of every report), a milestone no task belongs to and an
// unfinished task whose milestone is past due (both warnings: the protocol
// reports the calendar, the people keep it).
func CheckMilestones(p *spec.Project, specs []*spec.Spec, tasks []*Task, today string) []spec.Problem {
	if p == nil {
		return nil
	}
	declared := map[string]spec.Milestone{}
	for _, m := range p.Milestones {
		declared[m.ID] = m
	}
	var problems []spec.Problem
	for _, s := range specs {
		if s.Milestone != "" {
			if _, ok := declared[s.Milestone]; !ok {
				problems = append(problems, spec.Problem{Code: spec.ProblemMilestoneUnknown, Arg: s.ID + " " + s.Milestone})
			}
		}
	}
	byID := SpecsByID(specs)
	used := map[string]bool{}
	for _, t := range tasks {
		id := MilestoneOf(t, byID)
		if id == "" {
			continue
		}
		m, ok := declared[id]
		if !ok {
			// A spec's unknown milestone is already reported once; only the
			// task's own naming adds anything.
			if t.Milestone != "" {
				problems = append(problems, spec.Problem{Code: spec.ProblemMilestoneUnknown, Arg: t.ID + " " + id})
			}
			continue
		}
		used[id] = true
		if t.Status != Done && m.Overdue(today) {
			problems = append(problems, spec.Problem{Code: spec.ProblemMilestonePastDue, Arg: t.ID + " " + m.ID + " " + m.Due})
		}
	}
	for _, m := range p.Milestones {
		if !used[m.ID] {
			problems = append(problems, spec.Problem{Code: spec.ProblemMilestoneEmpty, Arg: m.ID})
		}
	}
	return problems
}
