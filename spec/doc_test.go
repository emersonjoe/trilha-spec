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
	if problems := l.Doctor(); len(problems) != 1 || !strings.Contains(problems[0], "ignores everything") {
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
	if err := l.SaveSpec(NewSpecDoc(id, "Implement OAuth login!")); err != nil {
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
