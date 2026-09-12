package agent

import (
	"strings"
	"testing"

	"github.com/emersonjoe/trilha-spec/spec"
)

func TestDefaultsAndRoundTrip(t *testing.T) {
	l, _, err := spec.Init(t.TempDir(), spec.InitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	all, err := List(l)
	if err != nil || len(all) != 2 || all[0].Name != "coder" || all[1].Name != "reviewer" {
		t.Fatalf("list = %v (%v)", all, err)
	}
	c, err := Load(l, "coder")
	if err != nil || !c.May("write") || c.May("network") || c.Driver != "exec" {
		t.Fatalf("coder = %+v (%v)", c, err)
	}
	c.Command = "claude -p -"
	c.Model = "m"
	if err := Save(l, c); err != nil {
		t.Fatal(err)
	}
	again, _ := Load(l, "coder")
	if again.Command != "claude -p -" || again.Model != "m" || len(again.Constraints) != 3 {
		t.Fatalf("again = %+v", again)
	}
	if _, err := Load(l, "Nope!"); err == nil {
		t.Fatal("bad name accepted")
	}
	if err := (&Agent{Name: "x", Role: "r", Tools: []string{"fly"}}).Validate(); err == nil || !strings.Contains(err.Error(), "fly") {
		t.Fatalf("err = %v", err)
	}
}
