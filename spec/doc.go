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
	"regexp"
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
// list of scalars, a block of scalars and lists, or a list of such blocks;
// that is the whole grammar, and it is enough to describe a task without
// pulling a YAML library into every reader.
type Fields struct {
	order []string
	vals  map[string]Value
}

// Value is one front matter entry. Exactly one shape holds, in this order of
// precedence: IsItems, IsMap, IsList, then the scalar.
type Value struct {
	Scalar string
	List   []string
	IsList bool
	// Sub is a nested block: scalars and lists under one key (`limits:`,
	// `review:`), with its keys in document order.
	Sub   Fields
	IsMap bool
	// Items is a list of nested blocks (`requirements:`, `milestones:`),
	// each one read and written like Sub.
	Items   []Fields
	IsItems bool
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

// SetMap writes a block of scalars, keys sorted, keeping the position of an
// existing key.
func (f *Fields) SetMap(key string, m map[string]string) {
	var sub Fields
	for _, k := range SortedKeys(m) {
		sub.Set(k, m[k])
	}
	f.put(key, Value{IsMap: true, Sub: sub})
}

// SetFields writes a block that may hold lists as well as scalars, in the
// order the block itself carries.
func (f *Fields) SetFields(key string, sub Fields) {
	f.put(key, Value{IsMap: true, Sub: sub})
}

// SetItems writes a list of blocks (`requirements:`, `milestones:`).
func (f *Fields) SetItems(key string, items []Fields) {
	f.put(key, Value{IsItems: true, Items: append([]Fields(nil), items...)})
}

// GetMap answers the scalars of the block at key, or nil when absent or not
// a block. A list inside the block is not a scalar and is left out; use
// GetFields to see the whole of it.
func (f Fields) GetMap(key string) map[string]string {
	v, ok := f.vals[key]
	if !ok || !v.IsMap {
		return nil
	}
	m := make(map[string]string, len(v.Sub.order))
	for _, k := range v.Sub.order {
		if x := v.Sub.vals[k]; !x.IsList && !x.IsMap && !x.IsItems {
			m[k] = x.Scalar
		}
	}
	return m
}

// GetFields answers the block at key and whether there was one.
func (f Fields) GetFields(key string) (Fields, bool) {
	v, ok := f.vals[key]
	if !ok || !v.IsMap {
		return Fields{}, false
	}
	return v.Sub, true
}

// GetItems answers the list of blocks at key, or nil when absent or not a
// list of blocks.
func (f Fields) GetItems(key string) []Fields {
	v, ok := f.vals[key]
	if !ok || !v.IsItems {
		return nil
	}
	return append([]Fields(nil), v.Items...)
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

// Get answers the scalar for key, or "" when absent or not a scalar.
func (f Fields) Get(key string) string {
	if v, ok := f.vals[key]; ok && !v.IsList && !v.IsMap && !v.IsItems {
		return v.Scalar
	}
	return ""
}

// GetList answers the list for key. A scalar answers a one-item list, so a
// task with a single dependency may write `depends_on: TASK-1`.
func (f Fields) GetList(key string) []string {
	v, ok := f.vals[key]
	if !ok || v.IsMap || v.IsItems {
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

// Map answers the fields as a plain map, lists as []string, blocks of
// scalars as map[string]string, a block holding a list as map[string]any and
// a list of blocks as []map[string]any: the JSON shape of a document.
func (f Fields) Map() map[string]any {
	m := make(map[string]any, len(f.order))
	for _, k := range f.order {
		m[k] = f.vals[k].any()
	}
	return m
}

func (v Value) any() any {
	switch {
	case v.IsItems:
		out := make([]map[string]any, 0, len(v.Items))
		for _, it := range v.Items {
			out = append(out, it.Map())
		}
		return out
	case v.IsMap:
		flat := map[string]string{}
		for _, k := range v.Sub.order {
			x := v.Sub.vals[k]
			if x.IsList || x.IsMap || x.IsItems {
				return v.Sub.Map()
			}
			flat[k] = x.Scalar
		}
		return flat
	case v.IsList:
		return v.List
	}
	return v.Scalar
}

// line is one front matter line kept with the indentation it was read at:
// the grammar below is indentation-sensitive once blocks nest.
type line struct {
	n      int // 1-based, for an error a person can find
	indent int
	text   string // trimmed of leading and trailing space
}

// Parse reads a document. The front matter is the block between the first
// line `---` and the next `---`; what follows is the body. Inside the block:
//
//	key: value          a scalar; quotes around the value are removed
//	key: [a, b]         an inline list
//	key:                a list, one `- item` per following line
//	  - item
//	key:                a block of scalars and lists, one per indented line
//	  sub: value
//	  sub: [a, b]
//	key:                a list of blocks, each one opened by `- `
//	  - sub: value
//	    sub: value
//	  - sub: value
//	key: {}             an empty block
//	# comment           ignored, as is a blank line
//
// A `- ` item whose text is an unquoted `sub: value` opens a block; a scalar
// item that holds a colon is quoted, which is how the writer emits it.
//
// A document without front matter fails with ErrNoFrontMatter.
func Parse(src []byte) (*Doc, error) {
	sc := bufio.NewScanner(bytes.NewReader(src))
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	if !sc.Scan() || strings.TrimRight(sc.Text(), " \t\r") != "---" {
		return nil, ErrNoFrontMatter
	}
	var lines []line
	n := 1
	closed := false
	for sc.Scan() {
		n++
		raw := strings.TrimRight(sc.Text(), " \t\r")
		if raw == "---" {
			closed = true
			break
		}
		t := strings.TrimSpace(raw)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		lines = append(lines, line{n: n, indent: len(raw) - len(strings.TrimLeft(raw, " \t")), text: t})
	}
	if !closed {
		return nil, errors.New("spec: front matter never closed with `---`")
	}
	d := &Doc{}
	i := 0
	fields, err := parseFields(lines, &i, 0)
	if err != nil {
		return nil, err
	}
	d.Fields = fields
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

func isItem(t string) bool { return t == "-" || strings.HasPrefix(t, "- ") }

// reBlockItem matches a `- ` item that opens a block rather than carrying a
// scalar: an unquoted key followed by a colon.
var reBlockItem = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]*:( |$)`)

// parseFields reads the entries at one level. It stops at the first line
// shallower than indent; a line deeper than indent that no block claimed
// belongs to this level all the same, which keeps a hand-indented document
// readable rather than fatal.
func parseFields(lines []line, i *int, indent int) (Fields, error) {
	var f Fields
	for *i < len(lines) {
		ln := lines[*i]
		if ln.indent < indent {
			return f, nil
		}
		if isItem(ln.text) {
			return f, fmt.Errorf("spec: line %d: list item without a key", ln.n)
		}
		key, val, ok := strings.Cut(ln.text, ":")
		if !ok || strings.ContainsAny(key, " \t") {
			return f, fmt.Errorf("spec: line %d: expected `key: value`, got %q", ln.n, ln.text)
		}
		val = strings.TrimSpace(val)
		*i++
		switch {
		case val == "{}":
			f.put(key, Value{IsMap: true})
		case strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]"):
			f.put(key, Value{IsList: true, List: inlineList(val)})
		case val != "":
			f.put(key, Value{Scalar: unquote(val)})
		default:
			v, err := parseBlock(lines, i, ln.indent)
			if err != nil {
				return f, err
			}
			f.put(key, v)
		}
	}
	return f, nil
}

// parseBlock reads what follows a bare `key:` — nothing, a list of scalars,
// a list of blocks or a block — given the indentation of the key itself.
func parseBlock(lines []line, i *int, keyIndent int) (Value, error) {
	if *i >= len(lines) || lines[*i].indent <= keyIndent {
		return Value{IsList: true, List: []string{}}, nil
	}
	sub := lines[*i].indent
	if !isItem(lines[*i].text) {
		f, err := parseFields(lines, i, sub)
		return Value{IsMap: true, Sub: f}, err
	}
	return parseList(lines, i, sub)
}

// parseList reads the `- ` items at indentation sub. Every item is a scalar
// or every item opens a block; the first item decides, and a mixture is a
// mistake worth naming.
func parseList(lines []line, i *int, sub int) (Value, error) {
	v := Value{IsList: true, List: []string{}}
	blocks := false
	for *i < len(lines) && lines[*i].indent == sub && isItem(lines[*i].text) {
		ln := lines[*i]
		rest := strings.TrimSpace(strings.TrimPrefix(ln.text, "-"))
		*i++
		if len(v.List) == 0 && len(v.Items) == 0 {
			blocks = reBlockItem.MatchString(rest)
			if blocks {
				v = Value{IsItems: true}
			}
		}
		if blocks != reBlockItem.MatchString(rest) {
			return v, fmt.Errorf("spec: line %d: a list holds scalars or blocks, not both", ln.n)
		}
		if !blocks {
			v.List = append(v.List, unquote(rest))
			continue
		}
		// The item's first entry rides on the `- ` line; the rest of the
		// block is whatever follows it, indented past the dash.
		head := []line{{n: ln.n, indent: sub + 2, text: rest}}
		j := 0
		item, err := parseFields(head, &j, sub+2)
		if err != nil {
			return v, err
		}
		tail, err := parseFields(lines, i, sub+1)
		if err != nil {
			return v, err
		}
		for _, k := range tail.Keys() {
			item.put(k, tail.vals[k])
		}
		v.Items = append(v.Items, item)
	}
	return v, nil
}

func inlineList(val string) []string {
	var items []string
	for _, it := range strings.Split(strings.TrimSuffix(strings.TrimPrefix(val, "["), "]"), ",") {
		if it = unquote(strings.TrimSpace(it)); it != "" {
			items = append(items, it)
		}
	}
	return items
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
	writeFields(&b, d.Fields, 0)
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

// writeFields renders one level, two spaces deeper per level.
func writeFields(b *bytes.Buffer, f Fields, indent int) {
	pad := strings.Repeat(" ", indent)
	for _, k := range f.order {
		v := f.vals[k]
		switch {
		case v.IsItems:
			// An item with no entry carries nothing; writing it would read
			// back as a scalar, so it is dropped.
			items := make([]Fields, 0, len(v.Items))
			for _, it := range v.Items {
				if len(it.order) > 0 {
					items = append(items, it)
				}
			}
			if len(items) == 0 {
				fmt.Fprintf(b, "%s%s: []\n", pad, k)
				continue
			}
			fmt.Fprintf(b, "%s%s:\n", pad, k)
			for _, it := range items {
				writeItem(b, it, indent+2)
			}
		case v.IsMap:
			if len(v.Sub.order) == 0 {
				fmt.Fprintf(b, "%s%s: {}\n", pad, k)
				continue
			}
			fmt.Fprintf(b, "%s%s:\n", pad, k)
			writeFields(b, v.Sub, indent+2)
		case v.IsList:
			if len(v.List) == 0 {
				fmt.Fprintf(b, "%s%s: []\n", pad, k)
				continue
			}
			fmt.Fprintf(b, "%s%s:\n", pad, k)
			for _, it := range v.List {
				fmt.Fprintf(b, "%s  - %s\n", pad, quote(it))
			}
		default:
			fmt.Fprintf(b, "%s%s: %s\n", pad, k, quote(v.Scalar))
		}
	}
}

// writeItem renders one block of a list: the first line carries the dash,
// the rest lines up under it.
func writeItem(b *bytes.Buffer, f Fields, indent int) {
	var inner bytes.Buffer
	writeFields(&inner, f, indent+2)
	s := inner.String()
	b.WriteString(strings.Repeat(" ", indent) + "- " + strings.TrimPrefix(s, strings.Repeat(" ", indent+2)))
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
