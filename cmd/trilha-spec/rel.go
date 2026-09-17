package main

import (
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"

	"github.com/emersonjoe/trilha-spec/spec"
)

func relPath(root, p string) (string, error) {
	r, err := filepath.Rel(root, p)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(r), nil
}

// parse reads flags anywhere on the line — `task add Title --spec X` and
// `task add --spec X Title` are the same call — and answers the positional
// arguments. flag.FlagSet stops at the first positional; this resumes.
func parse(fs *flag.FlagSet, args []string) ([]string, error) {
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return pos, nil
		}
		pos = append(pos, rest[0])
		args = rest[1:]
	}
}

// body is the pair of flags a document body comes from: --body with the text
// on the line, or --body-file with a path (`-` for stdin).
type body struct{ text, file *string }

func bodyFlags(fs *flag.FlagSet) body {
	return body{text: fs.String("body", "", "Markdown body"), file: fs.String("body-file", "", "file holding the body; - for stdin")}
}

func (b body) read() (string, error) {
	if *b.text != "" && *b.file != "" {
		return "", errors.New(T("--body and --body-file are exclusive"))
	}
	if *b.file == "" {
		return *b.text, nil
	}
	if *b.file == "-" {
		raw, err := io.ReadAll(os.Stdin)
		return string(raw), err
	}
	raw, err := os.ReadFile(*b.file)
	return string(raw), err
}

// security collects the security-impact flags `spec new` and `spec set`
// share. Each is repeatable; on `set`, a flag that was given replaces the
// whole list, one that was not leaves it alone.
type security struct {
	assets, boundaries, controls, evidence multi
}

func securityFlags(fs *flag.FlagSet) *security {
	var s security
	fs.Var(&s.assets, "asset", "asset the change touches (repeatable)")
	fs.Var(&s.boundaries, "boundary", "trust boundary the change crosses (repeatable)")
	fs.Var(&s.controls, "control", "affected control, e.g. \"ASVS V4.1\" (repeatable)")
	fs.Var(&s.evidence, "evidence", "command a reviewer must see run, no shell (repeatable)")
	return &s
}

func (s *security) apply(sp *spec.Spec) {
	if len(s.assets) > 0 {
		sp.Assets = s.assets
	}
	if len(s.boundaries) > 0 {
		sp.TrustBoundaries = s.boundaries
	}
	if len(s.controls) > 0 {
		sp.Controls = s.controls
	}
	if len(s.evidence) > 0 {
		sp.Evidence = s.evidence
	}
}
