package task

import "github.com/emersonjoe/trilha-spec/spec"

// CheckSpecStatus answers, as a warning, which approved specs have every
// task done. The work is delivered; the spec should say so (`spec move <id>
// done`), or a reader lists it among the things still open. A spec with no
// task yet is not reported: nothing was cut from it, nothing is late.
// Closing the spec is a human decision, which is why this never fails doctor.
func CheckSpecStatus(specs []*spec.Spec, tasks []*Task) []spec.Problem {
	total := map[string]int{}
	done := map[string]int{}
	for _, t := range tasks {
		if t.Spec == "" {
			continue
		}
		total[t.Spec]++
		if t.Status == Done {
			done[t.Spec]++
		}
	}
	var problems []spec.Problem
	for _, s := range specs {
		if s.Status != spec.Approved || total[s.ID] == 0 {
			continue
		}
		if done[s.ID] == total[s.ID] {
			problems = append(problems, spec.Problem{Code: spec.ProblemSpecTasksDone, Arg: s.ID})
		}
	}
	return problems
}
