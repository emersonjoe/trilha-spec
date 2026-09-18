package task

import (
	"fmt"
	"sort"
	"strings"
)

// Graph is the dependency graph: which tasks can run now, in what order the
// rest follow, and whether a cycle makes any of it impossible.
type Graph struct {
	Tasks map[string]*Task
	// Remote answers for dependencies in other repositories. A nil resolver
	// answers nothing, and a task waiting on another repository stays
	// blocked with a reason that names it.
	Remote Resolver
	order  []string
}

// NewGraph builds a graph and refuses one with a cycle or a dependency that
// does not exist — both are mistakes in the files, and the files are what
// gets fixed.
func NewGraph(tasks []*Task) (*Graph, error) { return NewGraphWith(tasks, nil) }

// NewGraphWith is NewGraph with something that can answer for dependencies in
// other repositories. Those never join the topological order — they are not
// this repository's work — but they do decide whether a task can start.
func NewGraphWith(tasks []*Task, remote Resolver) (*Graph, error) {
	g := &Graph{Tasks: make(map[string]*Task, len(tasks)), Remote: remote}
	for _, t := range tasks {
		g.Tasks[t.ID] = t
	}
	for _, t := range tasks {
		for _, r := range t.Refs() {
			if r.Remote() {
				continue
			}
			if _, ok := g.Tasks[r.ID]; !ok {
				return nil, fmt.Errorf("task %s depends on %s, which does not exist", t.ID, r.ID)
			}
		}
	}
	// Kahn's algorithm, picking the smallest ID among the ready ones so the
	// order is the same every time it is asked for.
	indeg := map[string]int{}
	rev := map[string][]string{}
	for id, t := range g.Tasks {
		indeg[id] += 0
		for _, r := range t.Refs() {
			if r.Remote() {
				continue
			}
			indeg[id]++
			rev[r.ID] = append(rev[r.ID], id)
		}
	}
	var ready []string
	for id, n := range indeg {
		if n == 0 {
			ready = append(ready, id)
		}
	}
	for len(ready) > 0 {
		sort.Strings(ready)
		id := ready[0]
		ready = ready[1:]
		g.order = append(g.order, id)
		for _, next := range rev[id] {
			indeg[next]--
			if indeg[next] == 0 {
				ready = append(ready, next)
			}
		}
	}
	if len(g.order) != len(g.Tasks) {
		var stuck []string
		for id, n := range indeg {
			if n > 0 {
				stuck = append(stuck, id)
			}
		}
		sort.Strings(stuck)
		return nil, fmt.Errorf("dependency cycle among %s", strings.Join(stuck, ", "))
	}
	return g, nil
}

// Order answers every task ID in an order that respects dependencies.
func (g *Graph) Order() []string { return append([]string(nil), g.order...) }

// Blockers answers the dependencies of id that are not done yet. A
// dependency in another repository is named as it is written; one nobody
// could answer for is named `waiting:<alias>:TASK-NNN`, so the reason a task
// cannot start says which repository has to be reached.
func (g *Graph) Blockers(id string) []string {
	t, ok := g.Tasks[id]
	if !ok {
		return nil
	}
	var open []string
	for _, r := range t.Refs() {
		if !r.Remote() {
			if dep := g.Tasks[r.ID]; dep == nil || dep.Status != Done {
				open = append(open, r.ID)
			}
			continue
		}
		status, answered := Status(""), false
		if g.Remote != nil {
			status, answered = g.Remote.Status(r.Alias, r.ID)
		}
		switch {
		case !answered:
			open = append(open, r.Waiting())
		case status != Done:
			open = append(open, r.String())
		}
	}
	sort.Strings(open)
	return open
}

// Ready answers the tasks an agent may pick up now: status ready, every
// dependency done, in dependency order.
func (g *Graph) Ready() []*Task {
	var out []*Task
	for _, id := range g.order {
		t := g.Tasks[id]
		if t.Status == Ready && len(g.Blockers(id)) == 0 {
			out = append(out, t)
		}
	}
	return out
}

// Next answers the first ready task, or nil.
func (g *Graph) Next() *Task {
	if r := g.Ready(); len(r) > 0 {
		return r[0]
	}
	return nil
}

// Mermaid renders the graph for a README or a PR description.
func (g *Graph) Mermaid() string {
	var b strings.Builder
	b.WriteString("graph TD\n")
	for _, id := range g.order {
		t := g.Tasks[id]
		fmt.Fprintf(&b, "  %s[\"%s<br/>%s\"]\n", node(id), id, strings.ReplaceAll(t.Title, `"`, "'"))
	}
	for _, id := range g.order {
		for _, r := range g.Tasks[id].Refs() {
			if r.Remote() {
				// A dependency in another repository is drawn as itself, so
				// the picture shows what the work is actually waiting for.
				fmt.Fprintf(&b, "  %s[\"%s\"] --> %s\n", node(r.String()), r.String(), node(id))
				continue
			}
			fmt.Fprintf(&b, "  %s --> %s\n", node(r.ID), node(id))
		}
	}
	return b.String()
}

// DOT renders the graph for Graphviz.
func (g *Graph) DOT() string {
	var b strings.Builder
	b.WriteString("digraph tasks {\n  rankdir=LR;\n")
	for _, id := range g.order {
		t := g.Tasks[id]
		fmt.Fprintf(&b, "  %q [label=\"%s\\n%s\\n(%s)\"];\n", id, id, strings.ReplaceAll(t.Title, `"`, "'"), t.Status)
	}
	for _, id := range g.order {
		for _, r := range g.Tasks[id].Refs() {
			fmt.Fprintf(&b, "  %q -> %q;\n", r.String(), id)
		}
	}
	b.WriteString("}\n")
	return b.String()
}

// node turns a reference into a Mermaid identifier: `app:TASK-004`
// becomes `app_TASK_004`.
func node(id string) string { return strings.NewReplacer("-", "_", ":", "_", ".", "_").Replace(id) }
