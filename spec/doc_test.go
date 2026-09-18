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

func TestSpecSecurityImpact(t *testing.T) {
	l, _, err := Init(t.TempDir(), InitOptions{Name: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	s := NewSpecDoc("001-auth", "Auth", "en", "")
	s.Status = Approved
	if s.HasSecurityImpact() {
		t.Fatal("empty spec has security impact")
	}
	// Approved with nothing to review: a warning, not a fault.
	problems := l.CheckSpecs([]*Spec{s})
	if len(problems) != 1 || problems[0].Code != ProblemSpecNoSecurity || !problems[0].Warning() || problems[0].Arg != "001-auth" {
		t.Fatalf("problems: %+v", problems)
	}
	if p := (Problem{ProblemSpecRefMissing, "x"}); p.Warning() {
		t.Fatal("a missing reference is not a warning")
	}
	s.Assets = []string{"session cookie", "users table"}
	s.TrustBoundaries = []string{"browser → api"}
	s.Controls = []string{"ASVS V4.1", "ASVS V3.4"}
	s.Evidence = []string{"go test ./internal/auth/...", `sh -c "curl -sf localhost:3000/login | grep -q Sign"`}
	if len(l.CheckSpecs([]*Spec{s})) != 0 {
		t.Fatal("declared impact still warned")
	}
	if err := l.SaveSpec(s); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(l.SpecFile("001-auth"))
	for _, want := range []string{"assets:\n  - session cookie\n  - users table\n", "trust_boundaries:\n  - browser → api\n", "controls:\n  - ASVS V4.1\n", "evidence:\n  - go test ./internal/auth/...\n"} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("written lacks %q:\n%s", want, b)
		}
	}
	got, err := l.LoadSpec("001-auth")
	if err != nil || len(got.Evidence) != 2 || got.Evidence[1] != `sh -c "curl -sf localhost:3000/login | grep -q Sign"` || got.TrustBoundaries[0] != "browser → api" {
		t.Fatalf("round trip: %v %+v", err, got)
	}
	// Dropping a list drops the key: a spec without evidence does not carry `evidence:`.
	got.Evidence = nil
	l.SaveSpec(got)
	if b, _ := os.ReadFile(l.SpecFile("001-auth")); strings.Contains(string(b), "evidence:") {
		t.Fatalf("empty list written:\n%s", b)
	}
	bad := &Spec{ID: "002-b", Title: "B", Status: Draft, Controls: []string{" "}}
	if err := bad.Validate(); err == nil || !strings.Contains(err.Error(), "controls has an empty item") {
		t.Fatalf("empty item accepted: %v", err)
	}
}

func TestMapFields(t *testing.T) {
	src := "---\nname: demo\nlimits:\n  max_cost_per_hour: 5.00\n  max_failure_rate: 0.5\nverify:\n  - go vet ./...\nempty: {}\n---\n"
	d, err := Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	m := d.Fields.GetMap("limits")
	if len(m) != 2 || m["max_cost_per_hour"] != "5.00" || m["max_failure_rate"] != "0.5" {
		t.Fatalf("limits = %v", m)
	}
	if d.Fields.Get("limits") != "" || d.Fields.GetList("limits") != nil {
		t.Fatal("a map is neither a scalar nor a list")
	}
	if got := d.Fields.GetList("verify"); len(got) != 1 {
		t.Fatalf("verify = %v", got)
	}
	if got := d.Fields.GetMap("empty"); got == nil || len(got) != 0 {
		t.Fatalf("empty map = %v", got)
	}
	if string(d.Bytes()) != src {
		t.Fatalf("not stable:\n%s\n---\n%s", src, d.Bytes())
	}
	if got := d.Fields.Map()["limits"].(map[string]string); got["max_failure_rate"] != "0.5" {
		t.Fatalf("Map() = %v", got)
	}
	// A list item under a map is an error; a map entry under a filled list is a new key.
	if _, err := Parse([]byte("---\nlimits:\n  a: 1\n  - x\n---\n")); err == nil {
		t.Fatal("list item under a map accepted")
	}
	d2, err := Parse([]byte("---\nverify:\n  - a\n  b: c\n---\n"))
	if err != nil || d2.Fields.Get("b") != "c" || len(d2.Fields.GetList("verify")) != 1 {
		t.Fatalf("indented key after a filled list: %v %+v", err, d2)
	}
	var f Fields
	f.SetMap("limits", map[string]string{"z": "1", "a": "2"})
	if got := (&Doc{Fields: f}).Bytes(); string(got) != "---\nlimits:\n  a: 2\n  z: 1\n---\n" {
		t.Fatalf("SetMap sorted:\n%s", got)
	}
}

func TestProjectLimitsAndPause(t *testing.T) {
	l, _, err := Init(t.TempDir(), InitOptions{Name: "demo", Description: "a demo"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := l.LoadProject()
	if err != nil || p.Limits != nil || p.Paused {
		t.Fatalf("fresh project: %v %+v", err, p)
	}
	p.Limits = map[string]float64{LimitMaxCostPerHour: 5, LimitMaxRepeatedFailureClass: 3, "custom": 0.25}
	p.Pause("breaker:max_repeated_failure_class")
	if err := l.SaveProject(p); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(l.Project())
	for _, want := range []string{"name: demo\n", "description: a demo\n", "default_agent: coder\n", "verify: []\n", "limits:\n  custom: 0.25\n  max_cost_per_hour: 5\n  max_repeated_failure_class: 3\n", "paused: true\n", "pause_reason: \"breaker:max_repeated_failure_class\"\n", "paused_at: \"20"} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("written lacks %q:\n%s", want, b)
		}
	}
	if !strings.Contains(string(b), "## ") {
		t.Fatalf("body lost:\n%s", b)
	}
	again, err := l.LoadProject()
	if err != nil || !again.Paused || again.PauseReason != "breaker:max_repeated_failure_class" || again.Limits["custom"] != 0.25 || again.Limits[LimitMaxCostPerHour] != 5 {
		t.Fatalf("round trip: %v %+v", err, again)
	}
	again.Resume()
	again.Limits = nil
	l.SaveProject(again)
	b, _ = os.ReadFile(l.Project())
	if strings.Contains(string(b), "paused") || strings.Contains(string(b), "limits") {
		t.Fatalf("resume left state:\n%s", b)
	}
	// What a reader refuses.
	if _, err := ParseProject([]byte("---\nname: x\nlimits:\n  max_cost_per_hour: five\n---\n")); err == nil || !strings.Contains(err.Error(), `limits.max_cost_per_hour "five" is not a number`) {
		t.Fatalf("text limit: %v", err)
	}
	bad := &Project{Name: "x", Limits: map[string]float64{LimitMaxFailureRate: 2, "n": -1}, Paused: true}
	err = bad.Validate()
	for _, want := range []string{"max_failure_rate is a share", "limits.n must not be negative", "paused without paused_at"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("validate lacks %q: %v", want, err)
		}
	}
	// Unknown keys survive a save: the file belongs to more tools than this one.
	os.WriteFile(l.Project(), []byte("---\nname: x\nowner: ops\nlimits:\n  max_cost_per_hour: 1\n---\n\nbody\n"), 0o644)
	p, _ = l.LoadProject()
	p.Pause("manual")
	l.SaveProject(p)
	b, _ = os.ReadFile(l.Project())
	if !strings.Contains(string(b), "owner: ops\n") || !strings.Contains(string(b), "limits:\n  max_cost_per_hour: 1\n") || !strings.HasSuffix(string(b), "\nbody\n") {
		t.Fatalf("save lost fields:\n%s", b)
	}
}

// TestBlockLists covers the two shapes protocol 0.3 added to the grammar: a
// list of blocks (`requirements:`, `milestones:`) and a block that holds a
// list (`review:`).
func TestBlockLists(t *testing.T) {
	src := "---\nid: 009-x\nrequirements:\n  - id: D2-R8\n    source: \"cp-01-2026#anexo-I\"\n    text: informar o cidadao\n  - id: D2-R9\n    text: \"prazo: 24h\"\nreview:\n  quorum: 2\n  roles: [uat, legal]\n---\n"
	// What the writer emits: an inline list becomes a block list, like any
	// other list in this grammar.
	canon := strings.Replace(src, "  roles: [uat, legal]\n", "  roles:\n    - uat\n    - legal\n", 1)
	d, err := Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	items := d.Fields.GetItems("requirements")
	if len(items) != 2 || items[0].Get("id") != "D2-R8" || items[0].Get("source") != "cp-01-2026#anexo-I" {
		t.Fatalf("requirements = %+v", items)
	}
	if items[1].Get("text") != "prazo: 24h" {
		t.Fatalf("quoted scalar in a block = %q", items[1].Get("text"))
	}
	sub, ok := d.Fields.GetFields("review")
	if !ok || sub.Get("quorum") != "2" || strings.Join(sub.GetList("roles"), ",") != "uat,legal" {
		t.Fatalf("review = %+v %v", sub, ok)
	}
	if d.Fields.GetItems("review") != nil || d.Fields.GetList("requirements") != nil {
		t.Fatal("a block is not a list of blocks, and the other way round")
	}
	if string(d.Bytes()) != canon {
		t.Fatalf("not stable:\n%s\n---\n%s", canon, d.Bytes())
	}
	if again, err := Parse(d.Bytes()); err != nil || string(again.Bytes()) != canon {
		t.Fatalf("second round trip: %v\n%s", err, again.Bytes())
	}
	// GetMap still answers the scalars of a block, and skips the list.
	if m := d.Fields.GetMap("review"); len(m) != 1 || m["quorum"] != "2" {
		t.Fatalf("GetMap = %v", m)
	}
	// The JSON shape: a list of blocks is a list of objects.
	if got := d.Fields.Map()["requirements"].([]map[string]any); got[0]["id"] != "D2-R8" {
		t.Fatalf("Map() = %v", got)
	}
	// Scalars and blocks never mix in one list.
	if _, err := Parse([]byte("---\nxs:\n  - a\n  - id: b\n---\n")); err == nil {
		t.Fatal("mixed list accepted")
	}
	// A comma in a quoted item is not a separator, and `- x: y` quoted is a
	// scalar, not a block.
	d2, err := Parse([]byte("---\nxs:\n  - \"a: b\"\n---\n"))
	if err != nil || strings.Join(d2.Fields.GetList("xs"), "|") != "a: b" {
		t.Fatalf("quoted item: %v %+v", err, d2.Fields.GetList("xs"))
	}
	// SetItems writes what Parse reads.
	var f Fields
	var one Fields
	one.Set("id", "M2")
	one.SetList("roles", []string{"uat"})
	f.SetItems("milestones", []Fields{one})
	if got := (&Doc{Fields: f}).Bytes(); string(got) != "---\nmilestones:\n  - id: M2\n    roles:\n      - uat\n---\n" {
		t.Fatalf("SetItems:\n%s", got)
	}
	if again, err := Parse((&Doc{Fields: f}).Bytes()); err != nil || again.Fields.GetItems("milestones")[0].Get("id") != "M2" {
		t.Fatalf("round trip: %v", err)
	}
	// An empty list of blocks is `[]`, like any other empty list.
	var g Fields
	g.SetItems("milestones", nil)
	if got := (&Doc{Fields: g}).Bytes(); string(got) != "---\nmilestones: []\n---\n" {
		t.Fatalf("empty items:\n%s", got)
	}
}
