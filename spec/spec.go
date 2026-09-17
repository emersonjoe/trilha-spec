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

// Spec is a specification: what to build and why, before it is cut into
// tasks. Its ID is `NNN-name`, the same numbering spec-kit uses for a
// feature branch, so the two can point at each other.
type Spec struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"` // draft | approved | done
	Issue  string `json:"issue,omitempty"`
	Body   string `json:"body,omitempty"`
	Fields Fields `json:"-"`
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
		ID:     d.Fields.Get("id"),
		Title:  d.Fields.Get("title"),
		Status: d.Fields.Get("status"),
		Issue:  d.Fields.Get("issue"),
		Body:   d.Body,
		Fields: d.Fields,
	}
	if s.Status == "" {
		s.Status = "draft"
	}
	return s, s.Validate()
}

// Validate checks the fields a reader relies on.
func (s *Spec) Validate() error {
	var errs []string
	if !ValidSpecID(s.ID) {
		errs = append(errs, fmt.Sprintf("id %q must look like 001-name", s.ID))
	}
	if strings.TrimSpace(s.Title) == "" {
		errs = append(errs, "title is required")
	}
	switch s.Status {
	case "draft", "approved", "done":
	default:
		errs = append(errs, fmt.Sprintf("status %q must be draft, approved or done", s.Status))
	}
	if len(errs) > 0 {
		return errors.New("spec " + s.ID + ": " + strings.Join(errs, "; "))
	}
	return nil
}

// Bytes writes the specification back.
func (s *Spec) Bytes() []byte {
	d := &Doc{Fields: s.Fields, Body: s.Body}
	d.Fields.Set("id", s.ID)
	d.Fields.Set("title", s.Title)
	d.Fields.Set("status", s.Status)
	if s.Issue != "" {
		d.Fields.Set("issue", s.Issue)
	} else {
		d.Fields.Delete("issue")
	}
	return d.Bytes()
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
	s := &Spec{ID: id, Title: title, Status: "draft"}
	if body == "" {
		body = fmt.Sprintf(Templates(lang).SpecBody, title)
	}
	s.Body = body
	return s
}
