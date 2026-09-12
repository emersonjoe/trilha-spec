package main

import (
	"flag"
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
