package spec

import (
	"fmt"
	"regexp"
	"strings"
)

// Requirement is one line of an external document a specification answers:
// a numbered item in a public tender, an article of a law, a KPI in a
// contract. The protocol does not import those documents — it carries the
// reference, so "which requirements are covered, by which task, with what
// evidence" stops being a spreadsheet kept by hand.
type Requirement struct {
	// ID is a free identifier, unique across the project ("D2-R8").
	ID string `json:"id"`
	// Source names the document the requirement comes from
	// ("cp-01-2026#anexo-I"); free text.
	Source string `json:"source,omitempty"`
	// Text is the requirement as the document words it.
	Text string `json:"text,omitempty"`
}

var reRequirementID = regexp.MustCompile(`^[^\s,]+$`)

// ValidRequirementID answers whether id can name a requirement: any text
// without a space or a comma, so `covers: [A, B]` never splits an id.
func ValidRequirementID(id string) bool { return reRequirementID.MatchString(id) }

// requirementsFrom reads the `requirements:` blocks of a document.
func requirementsFrom(f Fields) []Requirement {
	items := f.GetItems("requirements")
	if len(items) == 0 {
		return nil
	}
	out := make([]Requirement, 0, len(items))
	for _, it := range items {
		out = append(out, Requirement{ID: it.Get("id"), Source: it.Get("source"), Text: it.Get("text")})
	}
	return out
}

// requirementFields renders the blocks back, one key per field that has a
// value, so a requirement without a source does not carry an empty one.
func requirementFields(reqs []Requirement) []Fields {
	items := make([]Fields, 0, len(reqs))
	for _, r := range reqs {
		var f Fields
		f.Set("id", r.ID)
		if r.Source != "" {
			f.Set("source", r.Source)
		}
		if r.Text != "" {
			f.Set("text", r.Text)
		}
		items = append(items, f)
	}
	return items
}

// validateRequirements checks the shape of a spec's requirements: an id that
// can be written in a list, and no id declared twice in the same spec.
func validateRequirements(reqs []Requirement) []string {
	var errs []string
	seen := map[string]bool{}
	for _, r := range reqs {
		switch {
		case strings.TrimSpace(r.ID) == "":
			errs = append(errs, "a requirement has no id")
		case !ValidRequirementID(r.ID):
			errs = append(errs, fmt.Sprintf("requirement id %q holds a space or a comma", r.ID))
		case seen[r.ID]:
			errs = append(errs, fmt.Sprintf("requirement %s is declared twice", r.ID))
		}
		seen[r.ID] = true
	}
	return errs
}

// Requirement problem codes, for Doctor.
const (
	// ProblemRequirementDuplicate: the same id in two specs. Arg is
	// "ID (spec, spec)".
	ProblemRequirementDuplicate = "requirement-duplicate"
	// ProblemRequirementUnknown: a task covers an id no spec declares. Arg
	// is "TASK-NNN ID".
	ProblemRequirementUnknown = "requirement-unknown"
	// ProblemRequirementUncovered: a requirement no task covers. A warning:
	// the scope is declared, the work is not cut yet. Arg is "SPEC ID".
	ProblemRequirementUncovered = "requirement-uncovered"
)
