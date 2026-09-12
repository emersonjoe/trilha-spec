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
	order []string
}

// NewGraph builds a graph and refuses one with a cycle or a dependency that
// does not exist — both are mistakes in the files, and the files are what
// gets fixed.
func NewGraph(tasks []*Task) (*Graph, error) {
	g := &Graph{Tasks: make(map[string]*Task, len(tasks))}
	for _, t := range tasks {
		g.Tasks[t.ID] = t
	}
	for _, t := range tasks {
		for _, d := range t.DependsOn {
			if _, ok := g.Tasks[d]; !ok {
				return nil, fmt.Errorf("task %s depends on %s, which does not exist", t.ID, d)
			}
		}
	}
	// Kahn's algorithm, picking the smallest ID among the ready ones so the
	// order is the same every time it is asked for.
	indeg := map[string]int{}
	rev := map[string][]string{}
	for id, t := range g.Tasks {
		indeg[id] += 0
		for _, d := range t.DependsOn {
			indeg[id]++
			rev[d] = append(rev[d], id)
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

// Blockers answers the dependencies of id that are not done yet.
func (g *Graph) Blockers(id string) []string {
	t, ok := g.Tasks[id]
	if !ok {
		return nil
	}
	var open []string
	for _, d := range t.DependsOn {
		if dep := g.Tasks[d]; dep == nil || dep.Status != Done {
			open = append(open, d)
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
		for _, d := range g.Tasks[id].DependsOn {
			fmt.Fprintf(&b, "  %s --> %s\n", node(d), node(id))
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
		for _, d := range g.Tasks[id].DependsOn {
			fmt.Fprintf(&b, "  %q -> %q;\n", d, id)
		}
	}
	b.WriteString("}\n")
	return b.String()
}

func node(id string) string { return strings.ReplaceAll(id, "-", "_") }
