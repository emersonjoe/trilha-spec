package spec

import (
	"fmt"
	"strings"
	"time"
)

// DueFormat is how a milestone writes its date: a calendar day, not an
// instant. A contract's milestone is due on a date, in whatever timezone the
// contract is read in.
const DueFormat = "2006-01-02"

// Milestone is a dated point a contract or a programme pays against: M2, due
// 2027-04-30, gated on "PoC accepted by the client". The protocol carries the
// calendar; deciding whether a milestone is met is a person's call, and
// estimates and capacity belong to a control plane.
type Milestone struct {
	// ID is a free identifier, unique in the project ("M2").
	ID string `json:"id"`
	// Title is one line naming what the milestone delivers.
	Title string `json:"title,omitempty"`
	// Due is the day it is due, YYYY-MM-DD.
	Due string `json:"due,omitempty"`
	// Gate is what has to be true for it to be considered met.
	Gate string `json:"gate,omitempty"`
}

// ValidMilestoneID answers whether id can name a milestone: the same rule as
// a requirement id, so both survive a list.
func ValidMilestoneID(id string) bool { return ValidRequirementID(id) }

// Overdue answers whether the milestone's due date is strictly before day
// (YYYY-MM-DD). A milestone without a due date is never overdue.
func (m Milestone) Overdue(day string) bool {
	return m.Due != "" && day != "" && m.Due < day
}

// milestonesFrom reads the `milestones:` blocks of a document.
func milestonesFrom(f Fields) []Milestone {
	items := f.GetItems("milestones")
	if len(items) == 0 {
		return nil
	}
	out := make([]Milestone, 0, len(items))
	for _, it := range items {
		out = append(out, Milestone{ID: it.Get("id"), Title: it.Get("title"), Due: it.Get("due"), Gate: it.Get("gate")})
	}
	return out
}

// milestoneFields renders the blocks back, one key per field that has a value.
func milestoneFields(ms []Milestone) []Fields {
	items := make([]Fields, 0, len(ms))
	for _, m := range ms {
		var f Fields
		f.Set("id", m.ID)
		for _, kv := range []struct{ k, v string }{{"title", m.Title}, {"due", m.Due}, {"gate", m.Gate}} {
			if kv.v != "" {
				f.Set(kv.k, kv.v)
			}
		}
		items = append(items, f)
	}
	return items
}

// validateMilestones checks the shape: an id that survives a list, a date
// that is a date, and no id declared twice.
func validateMilestones(ms []Milestone) []string {
	var errs []string
	seen := map[string]bool{}
	for _, m := range ms {
		switch {
		case strings.TrimSpace(m.ID) == "":
			errs = append(errs, "a milestone has no id")
		case !ValidMilestoneID(m.ID):
			errs = append(errs, fmt.Sprintf("milestone id %q holds a space or a comma", m.ID))
		case seen[m.ID]:
			errs = append(errs, fmt.Sprintf("milestone %s is declared twice", m.ID))
		}
		seen[m.ID] = true
		if m.Due != "" {
			if _, err := time.Parse(DueFormat, m.Due); err != nil {
				errs = append(errs, fmt.Sprintf("milestone %s: due %q is not a date (YYYY-MM-DD)", m.ID, m.Due))
			}
		}
	}
	return errs
}

// Milestone answers one milestone of the project by id.
func (p *Project) Milestone(id string) (Milestone, bool) {
	for _, m := range p.Milestones {
		if m.ID == id {
			return m, true
		}
	}
	return Milestone{}, false
}

// SetMilestone adds or replaces a milestone, keeping declaration order.
func (p *Project) SetMilestone(m Milestone) {
	for i, x := range p.Milestones {
		if x.ID == m.ID {
			p.Milestones[i] = m
			return
		}
	}
	p.Milestones = append(p.Milestones, m)
}

// DeleteMilestone removes one by id and answers whether it was there.
func (p *Project) DeleteMilestone(id string) bool {
	for i, m := range p.Milestones {
		if m.ID == id {
			p.Milestones = append(p.Milestones[:i], p.Milestones[i+1:]...)
			return true
		}
	}
	return false
}

// Due answers the due date of every milestone that has one, by id: what a
// scheduler needs and nothing else.
func (p *Project) Due() map[string]string {
	out := map[string]string{}
	for _, m := range p.Milestones {
		if m.Due != "" {
			out[m.ID] = m.Due
		}
	}
	return out
}

// Milestone problem codes, for Doctor.
const (
	// ProblemMilestoneUnknown: a task or a spec names a milestone project.md
	// does not declare. Arg is "TASK-NNN ID" or "SPEC ID".
	ProblemMilestoneUnknown = "milestone-unknown"
	// ProblemMilestoneEmpty: a milestone no task belongs to. A warning.
	ProblemMilestoneEmpty = "milestone-empty"
	// ProblemMilestonePastDue: an unfinished task whose milestone is past
	// its due date. A warning: the protocol reports the calendar, it does
	// not enforce it. Arg is "TASK-NNN ID DUE".
	ProblemMilestonePastDue = "milestone-past-due"
)
