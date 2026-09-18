package task

import (
	"sort"

	"github.com/emersonjoe/trilha-spec/spec"
)

// CoverageTask is one task in a coverage row: enough to answer "is this
// requirement delivered, and what proves it" without opening the task.
type CoverageTask struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   Status `json:"status"`
	Evidence int    `json:"evidence"`
}

// Coverage is one requirement with the tasks that cover it. It is the matrix
// a public buyer asks for at every milestone: requirement → tasks → status →
// how much evidence there is.
type Coverage struct {
	Spec        string           `json:"spec"`
	Requirement spec.Requirement `json:"requirement"`
	Tasks       []CoverageTask   `json:"tasks"`
}

// Covered answers whether at least one task delivers the requirement.
func (c Coverage) Covered() bool { return len(c.Tasks) > 0 }

// Done answers whether every task covering the requirement is done, and
// there is at least one.
func (c Coverage) Done() bool {
	if len(c.Tasks) == 0 {
		return false
	}
	for _, t := range c.Tasks {
		if t.Status != Done {
			return false
		}
	}
	return true
}

// Cover builds the coverage matrix. An empty specID answers every
// requirement in the project, in spec order then declaration order.
func Cover(l spec.Layout, specID string) ([]Coverage, error) {
	specs, err := l.ListSpecs()
	if err != nil {
		return nil, err
	}
	st := &Store{Layout: l}
	tasks, err := st.List()
	if err != nil {
		return nil, err
	}
	byRequirement := map[string][]*Task{}
	for _, t := range tasks {
		for _, c := range t.Covers {
			byRequirement[c] = append(byRequirement[c], t)
		}
	}
	out := []Coverage{}
	for _, s := range specs {
		if specID != "" && s.ID != specID {
			continue
		}
		for _, r := range s.Requirements {
			row := Coverage{Spec: s.ID, Requirement: r, Tasks: []CoverageTask{}}
			for _, t := range byRequirement[r.ID] {
				ev, err := ListEvidence(l, t.ID)
				if err != nil {
					return nil, err
				}
				row.Tasks = append(row.Tasks, CoverageTask{ID: t.ID, Title: t.Title, Status: t.Status, Evidence: len(ev)})
			}
			out = append(out, row)
		}
	}
	return out, nil
}

// Requirements answers every requirement declared in the project, by id.
func Requirements(specs []*spec.Spec) map[string]spec.Requirement {
	out := map[string]spec.Requirement{}
	for _, s := range specs {
		for _, r := range s.Requirements {
			if _, dup := out[r.ID]; !dup {
				out[r.ID] = r
			}
		}
	}
	return out
}

// CheckRequirements answers what is wrong with the traceability: the same id
// declared by two specs, a task covering an id no spec declares, and — as a
// warning — a requirement no task covers yet.
func CheckRequirements(specs []*spec.Spec, tasks []*Task) []spec.Problem {
	var problems []spec.Problem
	owner := map[string][]string{}
	var order []string
	for _, s := range specs {
		for _, r := range s.Requirements {
			if len(owner[r.ID]) == 0 {
				order = append(order, r.ID)
			}
			owner[r.ID] = append(owner[r.ID], s.ID)
		}
	}
	for _, id := range order {
		if specIDs := owner[id]; len(specIDs) > 1 {
			problems = append(problems, spec.Problem{Code: spec.ProblemRequirementDuplicate, Arg: id + " (" + join(specIDs) + ")"})
		}
	}
	covered := map[string]bool{}
	for _, t := range tasks {
		for _, c := range t.Covers {
			covered[c] = true
			if len(owner[c]) == 0 {
				problems = append(problems, spec.Problem{Code: spec.ProblemRequirementUnknown, Arg: t.ID + " " + c})
			}
		}
	}
	for _, id := range order {
		if !covered[id] {
			problems = append(problems, spec.Problem{Code: spec.ProblemRequirementUncovered, Arg: owner[id][0] + " " + id})
		}
	}
	return problems
}

func join(ids []string) string {
	sorted := append([]string(nil), ids...)
	sort.Strings(sorted)
	out := ""
	for i, id := range sorted {
		if i > 0 {
			out += ", "
		}
		out += id
	}
	return out
}
