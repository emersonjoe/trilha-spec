package spec

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Status is where a specification is in its life. Unlike a task, a spec has
// no execution: it is written, judged, delivered, and it may be replaced.
type Status string

const (
	Draft      Status = "draft"      // being written
	Approved   Status = "approved"   // agreed; tasks may be cut from it
	Done       Status = "done"       // every task delivered
	Rejected   Status = "rejected"   // judged and refused; may return to draft
	Superseded Status = "superseded" // replaced by a spec that names it in `supersedes`
)

// Statuses lists every spec status in life order.
var Statuses = []Status{Draft, Approved, Done, Rejected, Superseded}

// Transitions says which spec moves are legal. Superseded is final: the
// successor carries the work on.
var Transitions = map[Status][]Status{
	Draft:      {Approved, Rejected, Superseded},
	Approved:   {Done, Rejected, Superseded, Draft},
	Done:       {Superseded},
	Rejected:   {Draft},
	Superseded: {},
}

// Valid answers whether s is one of the protocol's spec statuses.
func (s Status) Valid() bool {
	for _, t := range Statuses {
		if t == s {
			return true
		}
	}
	return false
}

// CanMoveTo answers whether the transition s → to is legal.
func (s Status) CanMoveTo(to Status) bool {
	for _, t := range Transitions[s] {
		if t == to {
			return true
		}
	}
	return false
}

// Spec is a specification: what to build and why, before it is cut into
// tasks. Its ID is `NNN-name`, the same numbering spec-kit uses for a
// feature branch, so the two can point at each other.
type Spec struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status Status `json:"status"`
	Issue  string `json:"issue,omitempty"`
	// Supersedes names the specs this one replaces; each of them is
	// `superseded`, and a `superseded` spec is named by exactly this field of
	// its successor.
	Supersedes []string `json:"supersedes,omitempty"`
	// DependsOn names the specs this one builds on.
	DependsOn []string `json:"depends_on,omitempty"`
	// Security impact, for the reviewer: what the change touches and which
	// controls it affects. Free identifiers ("session cookie", "ASVS V4.1");
	// the protocol judges nothing about them, it only carries them.
	Assets          []string `json:"assets,omitempty"`
	TrustBoundaries []string `json:"trust_boundaries,omitempty"`
	Controls        []string `json:"controls,omitempty"`
	// Evidence lists the commands a reviewer must see run before the spec is
	// done: program and arguments, never a shell, exactly like task checks.
	Evidence []string `json:"evidence,omitempty"`
	Body     string   `json:"body,omitempty"`
	Fields   Fields   `json:"-"`
}

// HasSecurityImpact answers whether the spec declares any of the security
// fields. An approved spec without them is a doctor warning: the reviewer
// has nothing to look at.
func (s *Spec) HasSecurityImpact() bool {
	return len(s.Assets)+len(s.TrustBoundaries)+len(s.Controls)+len(s.Evidence) > 0
}

var reSpecID = regexp.MustCompile(`^[0-9]{3}-[a-z0-9]+(-[a-z0-9]+)*$`)

// ValidSpecID answers whether id has the `NNN-name` shape.
func ValidSpecID(id string) bool { return reSpecID.MatchString(id) }

// ParseSpec reads a specification document.
func ParseSpec(src []byte) (*Spec, error) {
	d, err := Parse(src)
	if err != nil {
		return nil, err
	}
	s := &Spec{
		ID:              d.Fields.Get("id"),
		Title:           d.Fields.Get("title"),
		Status:          Status(d.Fields.Get("status")),
		Issue:           d.Fields.Get("issue"),
		Supersedes:      d.Fields.GetList("supersedes"),
		DependsOn:       d.Fields.GetList("depends_on"),
		Assets:          d.Fields.GetList("assets"),
		TrustBoundaries: d.Fields.GetList("trust_boundaries"),
		Controls:        d.Fields.GetList("controls"),
		Evidence:        d.Fields.GetList("evidence"),
		Body:            d.Body,
		Fields:          d.Fields,
	}
	if s.Status == "" {
		s.Status = Draft
	}
	return s, s.Validate()
}

// Validate checks the fields a reader relies on. References to other specs
// are checked for shape here and for existence by Layout.CheckSpecs.
func (s *Spec) Validate() error {
	var errs []string
	if !ValidSpecID(s.ID) {
		errs = append(errs, fmt.Sprintf("id %q must look like 001-name", s.ID))
	}
	if strings.TrimSpace(s.Title) == "" {
		errs = append(errs, "title is required")
	}
	if !s.Status.Valid() {
		errs = append(errs, fmt.Sprintf("status %q is not one of %v", s.Status, Statuses))
	}
	for field, refs := range map[string][]string{"supersedes": s.Supersedes, "depends_on": s.DependsOn} {
		for _, r := range refs {
			if !ValidSpecID(r) {
				errs = append(errs, fmt.Sprintf("%s %q is not a spec id", field, r))
			}
			if r == s.ID {
				errs = append(errs, "a spec cannot reference itself in "+field)
			}
		}
	}
	for field, items := range map[string][]string{"assets": s.Assets, "trust_boundaries": s.TrustBoundaries, "controls": s.Controls, "evidence": s.Evidence} {
		for _, it := range items {
			if strings.TrimSpace(it) == "" {
				errs = append(errs, field+" has an empty item")
			}
		}
	}
	sort.Strings(errs)
	if len(errs) > 0 {
		return errors.New("spec " + s.ID + ": " + strings.Join(errs, "; "))
	}
	return nil
}

// Move applies a transition, or answers why it cannot.
func (s *Spec) Move(to Status) error {
	if !to.Valid() {
		return fmt.Errorf("spec %s: %q is not a status", s.ID, to)
	}
	if !s.Status.CanMoveTo(to) {
		return fmt.Errorf("spec %s: cannot move from %s to %s (allowed: %v)", s.ID, s.Status, to, Transitions[s.Status])
	}
	s.Status = to
	return s.Validate()
}

// Bytes writes the specification back.
func (s *Spec) Bytes() []byte {
	d := &Doc{Fields: s.Fields, Body: s.Body}
	d.Fields.Set("id", s.ID)
	d.Fields.Set("title", s.Title)
	d.Fields.Set("status", string(s.Status))
	if s.Issue != "" {
		d.Fields.Set("issue", s.Issue)
	} else {
		d.Fields.Delete("issue")
	}
	// Fixed order, so a spec written twice is the same bytes.
	for _, kv := range []struct {
		k string
		v []string
	}{{"supersedes", s.Supersedes}, {"depends_on", s.DependsOn}, {"assets", s.Assets}, {"trust_boundaries", s.TrustBoundaries}, {"controls", s.Controls}, {"evidence", s.Evidence}} {
		if len(kv.v) > 0 {
			d.Fields.SetList(kv.k, kv.v)
		} else {
			d.Fields.Delete(kv.k)
		}
	}
	return d.Bytes()
}

// MoveSpec loads a spec, applies a transition and saves it.
func (l Layout) MoveSpec(id string, to Status) (*Spec, error) {
	s, err := l.LoadSpec(id)
	if err != nil {
		return nil, err
	}
	if err := s.Move(to); err != nil {
		return nil, err
	}
	return s, l.SaveSpec(s)
}

// Spec problem codes, for Doctor.
const (
	// ProblemSpecRefMissing: Arg is "ID field REF".
	ProblemSpecRefMissing = "spec-ref-missing"
	// ProblemSpecNoSuccessor: a superseded spec no other spec supersedes.
	ProblemSpecNoSuccessor = "spec-no-successor"
	// ProblemSpecNoSecurity: an approved spec with no assets, trust
	// boundaries, controls or evidence. A warning: the spec is still valid.
	ProblemSpecNoSecurity = "spec-no-security"
)

// warnings are the problem codes doctor reports without failing.
var warnings = map[string]bool{ProblemSpecNoSecurity: true}

// Warning answers whether the problem is advice rather than a fault.
func (p Problem) Warning() bool { return warnings[p.Code] }

// CheckSpecs answers what is wrong across specs: a reference to a spec that
// does not exist, a superseded spec that no successor names, and — as a
// warning — an approved spec that declares no security impact.
func (l Layout) CheckSpecs(specs []*Spec) []Problem {
	byID := map[string]*Spec{}
	for _, s := range specs {
		byID[s.ID] = s
	}
	successor := map[string]bool{}
	var problems []Problem
	for _, s := range specs {
		for _, field := range []string{"supersedes", "depends_on"} {
			refs := s.DependsOn
			if field == "supersedes" {
				refs = s.Supersedes
			}
			for _, r := range refs {
				if _, ok := byID[r]; !ok {
					problems = append(problems, Problem{ProblemSpecRefMissing, s.ID + " " + field + " " + r})
				}
				if field == "supersedes" {
					successor[r] = true
				}
			}
		}
	}
	for _, s := range specs {
		if s.Status == Superseded && !successor[s.ID] {
			problems = append(problems, Problem{ProblemSpecNoSuccessor, s.ID})
		}
		if s.Status == Approved && !s.HasSecurityImpact() {
			problems = append(problems, Problem{ProblemSpecNoSecurity, s.ID})
		}
	}
	return problems
}

// LoadSpec reads one specification by ID.
func (l Layout) LoadSpec(id string) (*Spec, error) {
	b, err := os.ReadFile(l.SpecFile(id))
	if err != nil {
		return nil, err
	}
	return ParseSpec(b)
}

// SaveSpec writes one specification.
func (l Layout) SaveSpec(s *Spec) error {
	if err := s.Validate(); err != nil {
		return err
	}
	return os.WriteFile(l.SpecFile(s.ID), s.Bytes(), 0o644)
}

// ListSpecs answers every specification, sorted by ID.
func (l Layout) ListSpecs() ([]*Spec, error) {
	entries, err := os.ReadDir(l.Specs())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []*Spec
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(l.Specs(), e.Name()))
		if err != nil {
			return nil, err
		}
		s, err := ParseSpec(b)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// NextSpecID answers the next free `NNN` prefix joined with name.
func (l Layout) NextSpecID(name string) (string, error) {
	specs, err := l.ListSpecs()
	if err != nil {
		return "", err
	}
	n := 0
	for _, s := range specs {
		var k int
		if _, err := fmt.Sscanf(s.ID, "%03d-", &k); err == nil && k > n {
			n = k
		}
	}
	id := fmt.Sprintf("%03d-%s", n+1, Slug(name))
	if !ValidSpecID(id) {
		return "", fmt.Errorf("spec: cannot make an id out of %q", name)
	}
	return id, nil
}

// Slug lowers a title into the `a-b-c` shape IDs and branches use. Latin
// accents are folded to their base letter so a Portuguese title makes a
// readable ID ("Fundação" → "fundacao"); anything else is a dash.
func Slug(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		if base, ok := accents[r]; ok {
			r = base
		}
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		default:
			if !dash && b.Len() > 0 {
				b.WriteByte('-')
				dash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

var accents = map[rune]rune{
	'á': 'a', 'à': 'a', 'â': 'a', 'ã': 'a', 'ä': 'a', 'é': 'e', 'è': 'e', 'ê': 'e', 'ë': 'e',
	'í': 'i', 'ì': 'i', 'î': 'i', 'ï': 'i', 'ó': 'o', 'ò': 'o', 'ô': 'o', 'õ': 'o', 'ö': 'o',
	'ú': 'u', 'ù': 'u', 'û': 'u', 'ü': 'u', 'ç': 'c', 'ñ': 'n',
}

// NewSpecDoc is what `spec new` writes: the spec-kit short form, in Markdown,
// in the language of Templates(lang). A body, when given, replaces the
// template.
func NewSpecDoc(id, title, lang, body string) *Spec {
	s := &Spec{ID: id, Title: title, Status: Draft}
	if body == "" {
		body = fmt.Sprintf(Templates(lang).SpecBody, title)
	}
	s.Body = body
	return s
}
