// Package spec is the document side of the Trilha protocol: the front matter
// every file under .trilha/ carries, the layout of that directory, the
// specification and project documents. It depends only on the standard
// library, on purpose — a protocol that needs a dependency to be read is a
// protocol fewer tools will read.
package spec

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Doc is a Markdown document with a front matter block: the shape of every
// file the protocol writes. Fields keep the order they were read in, so a
// document written back is byte-for-byte what was read when nothing changed.
type Doc struct {
	Fields Fields
	Body   string
}

// Fields is an ordered set of front matter entries. A value is a scalar, a
// list of scalars or a map of scalars one level deep; that is the whole
// grammar, and it is enough to describe a task without pulling a YAML
// library into every reader.
type Fields struct {
	order []string
	vals  map[string]Value
}

// Value is one front matter entry. Exactly one of the three shapes holds:
// IsMap wins over IsList, IsList over the scalar.
type Value struct {
	Scalar string
	List   []string
	IsList bool
	// Map is a one-level map of scalars (`limits:` in project.md), with its
	// keys in document order.
	Map     map[string]string
	MapKeys []string
	IsMap   bool
}

// ErrNoFrontMatter is a document that does not start with the `---` fence.
var ErrNoFrontMatter = errors.New("spec: document has no front matter")

// Set writes a scalar, keeping the position of an existing key.
func (f *Fields) Set(key, value string) {
	f.put(key, Value{Scalar: value})
}

// SetList writes a list, keeping the position of an existing key.
func (f *Fields) SetList(key string, items []string) {
	f.put(key, Value{List: append([]string(nil), items...), IsList: true})
}

// SetMap writes a map of scalars, keys sorted, keeping the position of an
// existing key.
func (f *Fields) SetMap(key string, m map[string]string) {
	v := Value{IsMap: true, Map: map[string]string{}, MapKeys: SortedKeys(m)}
	for k, x := range m {
		v.Map[k] = x
	}
	f.put(key, v)
}

// GetMap answers the map for key, or nil when absent or not a map.
func (f Fields) GetMap(key string) map[string]string {
	v, ok := f.vals[key]
	if !ok || !v.IsMap {
		return nil
	}
	m := make(map[string]string, len(v.Map))
	for k, x := range v.Map {
		m[k] = x
	}
	return m
}

func (f *Fields) put(key string, v Value) {
	if f.vals == nil {
		f.vals = map[string]Value{}
	}
	if _, ok := f.vals[key]; !ok {
		f.order = append(f.order, key)
	}
	f.vals[key] = v
}

// Get answers the scalar for key, or "" when absent, a list or a map.
func (f Fields) Get(key string) string {
	if v, ok := f.vals[key]; ok && !v.IsList && !v.IsMap {
		return v.Scalar
	}
	return ""
}

// GetList answers the list for key. A scalar answers a one-item list, so a
// task with a single dependency may write `depends_on: TASK-1`.
func (f Fields) GetList(key string) []string {
	v, ok := f.vals[key]
	if !ok || v.IsMap {
		return nil
	}
	if v.IsList {
		return append([]string(nil), v.List...)
	}
	if v.Scalar == "" {
		return nil
	}
	return []string{v.Scalar}
}

// Has answers whether key is present, whatever its value.
func (f Fields) Has(key string) bool { _, ok := f.vals[key]; return ok }

// Delete removes key.
func (f *Fields) Delete(key string) {
	if _, ok := f.vals[key]; !ok {
		return
	}
	delete(f.vals, key)
	for i, k := range f.order {
		if k == key {
			f.order = append(f.order[:i], f.order[i+1:]...)
			break
		}
	}
}

// Keys answers the keys in document order.
func (f Fields) Keys() []string { return append([]string(nil), f.order...) }

// Map answers the fields as a plain map, lists as []string, maps as
// map[string]string and scalars as string: the JSON shape of a document.
func (f Fields) Map() map[string]any {
	m := make(map[string]any, len(f.order))
	for _, k := range f.order {
		v := f.vals[k]
		switch {
		case v.IsMap:
			m[k] = v.Map
		case v.IsList:
			m[k] = v.List
		default:
			m[k] = v.Scalar
		}
	}
	return m
}

// Parse reads a document. The front matter is the block between the first
// line `---` and the next `---`; what follows is the body. Inside the block:
//
//	key: value          a scalar; quotes around the value are removed
//	key: [a, b]         an inline list
//	key:                a list, one `- item` per following line
//	  - item
//	key:                a map of scalars, one indented `sub: value` per line
//	  sub: value
//	key: {}             an empty map
//	# comment           ignored, as is a blank line
//
// A document without front matter fails with ErrNoFrontMatter.
func Parse(src []byte) (*Doc, error) {
	sc := bufio.NewScanner(bytes.NewReader(src))
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	if !sc.Scan() || strings.TrimRight(sc.Text(), " \t\r") != "---" {
		return nil, ErrNoFrontMatter
	}
	d := &Doc{}
	var listKey string
	line := 1
	closed := false
	for sc.Scan() {
		line++
		raw := strings.TrimRight(sc.Text(), " \t\r")
		if raw == "---" {
			closed = true
			break
		}
		t := strings.TrimSpace(raw)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		if strings.HasPrefix(t, "- ") || t == "-" {
			if listKey == "" || d.Fields.vals[listKey].IsMap {
				return nil, fmt.Errorf("spec: line %d: list item without a key", line)
			}
			v := d.Fields.vals[listKey]
			v.List = append(v.List, unquote(strings.TrimSpace(strings.TrimPrefix(t, "-"))))
			d.Fields.vals[listKey] = v
			continue
		}
		key, val, ok := strings.Cut(t, ":")
		if !ok || strings.ContainsAny(key, " \t") {
			return nil, fmt.Errorf("spec: line %d: expected `key: value`, got %q", line, t)
		}
		val = strings.TrimSpace(val)
		// An indented `sub: value` right after `key:` (or after another
		// entry of the same map) is a map entry, not a top-level key.
		if raw != t && listKey != "" {
			v := d.Fields.vals[listKey]
			if v.IsMap || (v.IsList && len(v.List) == 0) {
				if !v.IsMap {
					v = Value{IsMap: true, Map: map[string]string{}}
				}
				if _, dup := v.Map[key]; !dup {
					v.MapKeys = append(v.MapKeys, key)
				}
				v.Map[key] = unquote(val)
				d.Fields.vals[listKey] = v
				continue
			}
		}
		switch {
		case val == "":
			listKey = key
			d.Fields.put(key, Value{IsList: true, List: []string{}})
		case val == "{}":
			listKey = ""
			d.Fields.put(key, Value{IsMap: true, Map: map[string]string{}})
		case strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]"):
			listKey = ""
			var items []string
			for _, it := range strings.Split(strings.TrimSuffix(strings.TrimPrefix(val, "["), "]"), ",") {
				if it = unquote(strings.TrimSpace(it)); it != "" {
					items = append(items, it)
				}
			}
			d.Fields.put(key, Value{IsList: true, List: items})
		default:
			listKey = ""
			d.Fields.put(key, Value{Scalar: unquote(val)})
		}
	}
	if !closed {
		return nil, errors.New("spec: front matter never closed with `---`")
	}
	var body strings.Builder
	for sc.Scan() {
		body.WriteString(sc.Text())
		body.WriteByte('\n')
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	d.Body = strings.TrimLeft(body.String(), "\n")
	return d, nil
}

// unquote removes a matching pair of quotes. Inside double quotes, `\"` and
// `\\` are the two escapes; single quotes carry the text as is.
func unquote(s string) string {
	if len(s) < 2 {
		return s
	}
	switch {
	case s[0] == '\'' && s[len(s)-1] == '\'':
		return s[1 : len(s)-1]
	case s[0] == '"' && s[len(s)-1] == '"':
		var b strings.Builder
		esc := false
		for _, r := range s[1 : len(s)-1] {
			switch {
			case esc:
				b.WriteRune(r)
				esc = false
			case r == '\\':
				esc = true
			default:
				b.WriteRune(r)
			}
		}
		return b.String()
	}
	return s
}

// needsQuotes says whether a scalar would be misread when written bare.
func needsQuotes(s string) bool {
	if s == "" {
		return true
	}
	if strings.ContainsAny(s, ":#[]\"'\n") || strings.HasPrefix(s, "- ") || s == "-" {
		return true
	}
	return s != strings.TrimSpace(s)
}

// Bytes writes the document back: front matter in field order, then a blank
// line and the body. It is the inverse of Parse for what Parse accepts.
func (d *Doc) Bytes() []byte {
	var b bytes.Buffer
	b.WriteString("---\n")
	for _, k := range d.Fields.order {
		v := d.Fields.vals[k]
		if v.IsMap {
			if len(v.MapKeys) == 0 {
				fmt.Fprintf(&b, "%s: {}\n", k)
				continue
			}
			fmt.Fprintf(&b, "%s:\n", k)
			for _, sub := range v.MapKeys {
				fmt.Fprintf(&b, "  %s: %s\n", sub, quote(v.Map[sub]))
			}
			continue
		}
		if v.IsList {
			if len(v.List) == 0 {
				fmt.Fprintf(&b, "%s: []\n", k)
				continue
			}
			fmt.Fprintf(&b, "%s:\n", k)
			for _, it := range v.List {
				fmt.Fprintf(&b, "  - %s\n", quote(it))
			}
			continue
		}
		fmt.Fprintf(&b, "%s: %s\n", k, quote(v.Scalar))
	}
	b.WriteString("---\n")
	if d.Body != "" {
		b.WriteString("\n")
		b.WriteString(d.Body)
		if !strings.HasSuffix(d.Body, "\n") {
			b.WriteByte('\n')
		}
	}
	return b.Bytes()
}

func quote(s string) string {
	if !needsQuotes(s) {
		return s
	}
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

// SortedKeys is a helper for deterministic output elsewhere in the module.
func SortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
