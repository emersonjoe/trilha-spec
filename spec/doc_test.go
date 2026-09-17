package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sample = `---
id: TASK-123
title: Implement OAuth
depends_on:
  - TASK-100
  - TASK-101
tags: [auth, "web"]
acceptance:
  - OAuth login works
  - Tests pass
empty:
note: "a: b"
---

Body line one.

Body line two.
`

func TestParseRoundTrip(t *testing.T) {
	d, err := Parse([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	if got := d.Fields.Get("id"); got != "TASK-123" {
		t.Fatalf("id = %q", got)
	}
	if got := d.Fields.GetList("depends_on"); len(got) != 2 || got[1] != "TASK-101" {
		t.Fatalf("depends_on = %v", got)
	}
	if got := d.Fields.GetList("tags"); len(got) != 2 || got[1] != "web" {
		t.Fatalf("tags = %v", got)
	}
	if got := d.Fields.Get("note"); got != "a: b" {
		t.Fatalf("note = %q", got)
	}
	if got := d.Fields.GetList("empty"); len(got) != 0 {
		t.Fatalf("empty = %v", got)
	}
	if !strings.HasPrefix(d.Body, "Body line one.") {
		t.Fatalf("body = %q", d.Body)
	}
	// Writing back and parsing again is the same document.
	again, err := Parse(d.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if string(again.Bytes()) != string(d.Bytes()) {
		t.Fatalf("not stable:\n%s\n---\n%s", d.Bytes(), again.Bytes())
	}
	if got := d.Fields.Keys(); strings.Join(got, ",") != "id,title,depends_on,tags,acceptance,empty,note" {
		t.Fatalf("order = %v", got)
	}
}

func TestParseErrors(t *testing.T) {
	if _, err := Parse([]byte("no fence")); err != ErrNoFrontMatter {
		t.Fatalf("err = %v", err)
	}
	if _, err := Parse([]byte("---\nid: x\n")); err == nil {
		t.Fatal("unclosed front matter accepted")
	}
	if _, err := Parse([]byte("---\n- item\n---\n")); err == nil {
		t.Fatal("orphan list item accepted")
	}
	if _, err := Parse([]byte("---\nnot a field\n---\n")); err == nil {
		t.Fatal("bare line accepted")
	}
}

func TestQuotedScalarsSurvive(t *testing.T) {
	d := &Doc{}
	d.Fields.Set("check", `sh -c "test -f a\\b"`)
	d.Fields.SetList("list", []string{"a: b", "- dash", "", `q"q`})
	again, err := Parse(d.Bytes())
	if err != nil {
		t.Fatalf("%v\n%s", err, d.Bytes())
	}
	if got := again.Fields.Get("check"); got != `sh -c "test -f a\\b"` {
		t.Fatalf("check = %q\n%s", got, d.Bytes())
	}
	if got := again.Fields.GetList("list"); strings.Join(got, "|") != `a: b|- dash||q"q` {
		t.Fatalf("list = %q", got)
	}
	// Single quotes are accepted on read, as YAML writes them.
	d2, _ := Parse([]byte("---\nx: 'it''s'\n---\n"))
	if got := d2.Fields.Get("x"); got != "it''s" {
		t.Fatalf("x = %q", got)
	}
}

func TestScalarAsList(t *testing.T) {
	d, err := Parse([]byte("---\ndepends_on: TASK-1\n---\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := d.Fields.GetList("depends_on"); len(got) != 1 || got[0] != "TASK-1" {
		t.Fatalf("got %v", got)
	}
}

func TestInitAndFind(t *testing.T) {
	root := t.TempDir()
	l, wrote, err := Init(root, InitOptions{Name: "demo", Description: "a demo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(wrote) == 0 {
		t.Fatal("nothing written")
	}
	// Idempotent: a second run writes nothing and keeps the files.
	if _, again, err := Init(root, InitOptions{}); err != nil || len(again) != 0 {
		t.Fatalf("second init wrote %v (%v)", again, err)
	}
	p, err := l.LoadProject()
	if err != nil || p.Name != "demo" || p.DefaultAgent != "coder" {
		t.Fatalf("project = %+v (%v)", p, err)
	}
	sub := filepath.Join(root, "a", "b")
	os.MkdirAll(sub, 0o755)
	found, err := Find(sub)
	if err != nil || found.Root != l.Root {
		t.Fatalf("find = %v (%v)", found, err)
	}
	if problems := l.Doctor(); len(problems) != 0 {
		t.Fatalf("doctor: %v", problems)
	}
	// The framework's dev server clobbers .gitignore with `*`; doctor sees it.
	os.WriteFile(filepath.Join(l.Dir(), ".gitignore"), []byte("*\n"), 0o644)
	if problems := l.Doctor(); len(problems) != 1 || !strings.Contains(problems[0].String(), "ignores everything") {
		t.Fatalf("doctor: %v", problems)
	}
	if _, err := Find(t.TempDir()); err != ErrNotInitialized {
		t.Fatalf("err = %v", err)
	}
}

func TestSpecs(t *testing.T) {
	root := t.TempDir()
	l, _, err := Init(root, InitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	id, err := l.NextSpecID("Implement OAuth login!")
	if err != nil || id != "001-implement-oauth-login" {
		t.Fatalf("id = %q (%v)", id, err)
	}
	if err := l.SaveSpec(NewSpecDoc(id, "Implement OAuth login!", "en", "")); err != nil {
		t.Fatal(err)
	}
	if got := Slug("Fundação (Fase 0)"); got != "fundacao-fase-0" {
		t.Fatalf("slug = %q", got)
	}
	id2, _ := l.NextSpecID("second")
	if id2 != "002-second" {
		t.Fatalf("id2 = %q", id2)
	}
	specs, err := l.ListSpecs()
	if err != nil || len(specs) != 1 || specs[0].Status != "draft" {
		t.Fatalf("specs = %v (%v)", specs, err)
	}
	if err := (&Spec{ID: "bad", Title: "x", Status: "draft"}).Validate(); err == nil {
		t.Fatal("bad id accepted")
	}
}

func TestTemplatesByLanguage(t *testing.T) {
	root := t.TempDir()
	l, _, err := Init(root, InitOptions{Name: "demo", Lang: "pt"})
	if err != nil {
		t.Fatal(err)
	}
	c, _ := l.LoadConstitution()
	if !strings.Contains(c, "# Constituição") {
		t.Fatalf("pt constitution:\n%s", c)
	}
	p, err := l.LoadProject()
	if err != nil || p.Name != "demo" || p.DefaultAgent != "coder" || !strings.Contains(p.Body, "## Comandos") {
		t.Fatalf("pt project: %v %+v", err, p)
	}
	// Front matter keys never move: an agent manifest in pt-BR is still read by
	// a reader that knows only the English keys.
	b, _ := os.ReadFile(l.AgentFile("coder"))
	if !strings.Contains(string(b), "role: Implementa") || !strings.Contains(string(b), "tools:\n  - read") {
		t.Fatalf("pt agent:\n%s", b)
	}
	s := NewSpecDoc("001-x", "Login", "pt", "")
	if !strings.Contains(s.Body, "# Login\n\n## Por quê") {
		t.Fatalf("pt spec:\n%s", s.Body)
	}
	if s := NewSpecDoc("001-x", "Login", "xx", ""); !strings.Contains(s.Body, "## Why") {
		t.Fatalf("unknown language must be English:\n%s", s.Body)
	}
	if s := NewSpecDoc("001-x", "Login", "pt", "custom body\n"); s.Body != "custom body\n" {
		t.Fatalf("body must replace the template: %q", s.Body)
	}
}

func TestSpecLifecycle(t *testing.T) {
	s := &Spec{ID: "001-a", Title: "A", Status: Draft}
	if err := s.Move(Done); err == nil || !strings.Contains(err.Error(), "cannot move from draft to done") {
		t.Fatalf("draft → done accepted: %v", err)
	}
	for _, to := range []Status{Approved, Done, Superseded} {
		if err := s.Move(to); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Move(Draft); err == nil {
		t.Fatal("superseded is final")
	}
	if err := s.Move("nope"); err == nil || !strings.Contains(err.Error(), "is not a status") {
		t.Fatalf("unknown status: %v", err)
	}
	r := &Spec{ID: "002-b", Title: "B", Status: Rejected}
	if err := r.Move(Draft); err != nil {
		t.Fatal(err)
	}
	// Every status in the table is a real status, and every target too.
	for from, tos := range Transitions {
		if !from.Valid() {
			t.Errorf("%q in Transitions is not a status", from)
		}
		for _, to := range tos {
			if !to.Valid() {
				t.Errorf("%q → %q: target is not a status", from, to)
			}
		}
	}
	bad := &Spec{ID: "003-c", Title: "C", Status: Approved, Supersedes: []string{"003-c", "x"}, DependsOn: []string{"001-a"}}
	err := bad.Validate()
	if err == nil || !strings.Contains(err.Error(), "cannot reference itself in supersedes") || !strings.Contains(err.Error(), `supersedes "x" is not a spec id`) {
		t.Fatalf("validate: %v", err)
	}
}

func TestSpecRelationsRoundTripAndCheck(t *testing.T) {
	l, _, err := Init(t.TempDir(), InitOptions{Name: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	old := NewSpecDoc("001-old", "Old", "en", "")
	old.Status = Superseded
	succ := NewSpecDoc("002-new", "New", "en", "")
	succ.Supersedes = []string{"001-old"}
	succ.DependsOn = []string{"009-missing"}
	for _, s := range []*Spec{old, succ} {
		if err := l.SaveSpec(s); err != nil {
			t.Fatal(err)
		}
	}
	got, err := l.LoadSpec("002-new")
	if err != nil || got.Supersedes[0] != "001-old" || got.DependsOn[0] != "009-missing" {
		t.Fatalf("round trip: %v %+v", err, got)
	}
	b, _ := os.ReadFile(l.SpecFile("002-new"))
	if !strings.Contains(string(b), "supersedes:\n  - 001-old\ndepends_on:\n  - 009-missing\n") {
		t.Fatalf("written:\n%s", b)
	}
	specs, _ := l.ListSpecs()
	problems := l.CheckSpecs(specs)
	if len(problems) != 1 || problems[0].Code != ProblemSpecRefMissing || problems[0].Arg != "002-new depends_on 009-missing" {
		t.Fatalf("problems: %+v", problems)
	}
	// A superseded spec nobody supersedes is an orphan.
	succ.Supersedes = nil
	specs = []*Spec{old, succ}
	problems = l.CheckSpecs(specs)
	found := false
	for _, p := range problems {
		if p.Code == ProblemSpecNoSuccessor && p.Arg == "001-old" {
			found = true
		}
	}
	if !found {
		t.Fatalf("orphan not reported: %+v", problems)
	}
	if _, err := l.MoveSpec("002-new", Approved); err != nil {
		t.Fatal(err)
	}
	if s, _ := l.LoadSpec("002-new"); s.Status != Approved {
		t.Fatalf("move not saved: %s", s.Status)
	}
}
