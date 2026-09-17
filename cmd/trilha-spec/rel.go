package main

import (
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
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
