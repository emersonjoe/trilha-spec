package task

import (
	"fmt"
	"sort"
	"strings"

	"github.com/emersonjoe/trilha-spec/spec"
)

// ProgramGraph draws a whole programme: one subgraph per repository the
// manifest names and a checkout answers for, with the dependencies that cross
// between them. A repository nobody checked out still appears as the edges
// that wait for it, because that is the thing worth seeing.
func ProgramGraph(l spec.Layout, prog *spec.Program, checkouts Checkouts) (string, error) {
	local := ""
	if prog != nil {
		local = prog.AliasOf(l.Root)
	}
	if local == "" {
		local = "this"
	}
	// The repository we are standing in, plus every one we can read.
	repos := map[string][]*Task{}
	own, err := (&Store{Layout: l}).List()
	if err != nil {
		return "", err
	}
	repos[local] = own
	for alias, root := range checkouts {
		if alias == local {
			continue
		}
		side, err := spec.Find(root)
		if err != nil {
			continue
		}
		list, err := (&Store{Layout: side}).List()
		if err != nil {
			continue
		}
		repos[alias] = list
	}

	var b strings.Builder
	b.WriteString("graph TD\n")
	drawn := map[string]bool{}
	for _, alias := range spec.SortedKeys(repos) {
		fmt.Fprintf(&b, "  subgraph %s\n", alias)
		for _, t := range repos[alias] {
			id := alias + ":" + t.ID
			drawn[id] = true
			fmt.Fprintf(&b, "    %s[\"%s<br/>%s\"]\n", node(id), t.ID, strings.ReplaceAll(t.Title, `"`, "'"))
		}
		b.WriteString("  end\n")
	}
	// Edges last, so a node in a repository nobody checked out is declared
	// outside every subgraph and still shows up in the picture.
	var edges []string
	for _, alias := range spec.SortedKeys(repos) {
		for _, t := range repos[alias] {
			to := alias + ":" + t.ID
			for _, r := range t.Refs() {
				from := r.String()
				if !r.Remote() {
					from = alias + ":" + r.ID
				}
				if !drawn[from] {
					drawn[from] = true
					edges = append(edges, fmt.Sprintf("  %s[\"%s\"]\n", node(from), from))
				}
				edges = append(edges, fmt.Sprintf("  %s --> %s\n", node(from), node(to)))
			}
		}
	}
	sort.Strings(edges)
	for _, e := range edges {
		b.WriteString(e)
	}
	return b.String(), nil
}
