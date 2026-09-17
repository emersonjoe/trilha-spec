package task

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/emersonjoe/trilha-spec/spec"
)

// Evidence is one verifiable fact about a task: a check that ran with its
// exit code and the hash of what it printed, a note a reviewer left, an
// artifact an agent produced. It is JSON on disk, one file per record, so it
// can be diffed, signed later and read without this package.
type Evidence struct {
	Task string `json:"task"`
	Seq  int    `json:"seq"`
	// Kind: check | note | artifact | run
	Kind string `json:"kind"`
	At   string `json:"at"`
	// By is who produced it: an agent name, a person, "trilha-spec verify".
	By       string `json:"by"`
	Command  string `json:"command,omitempty"`
	Dir      string `json:"dir,omitempty"`
	ExitCode int    `json:"exit_code"`
	// Output is what the command printed, capped at MaxOutput; OutputSHA256
	// is the hash of the whole of it, so a trimmed record still proves what
	// ran.
	Output       string   `json:"output,omitempty"`
	OutputSHA256 string   `json:"output_sha256,omitempty"`
	Passed       bool     `json:"passed"`
	Note         string   `json:"note,omitempty"`
	Files        []string `json:"files,omitempty"`
	// Cost of a `run`, as the runner observed it — the protocol does not
	// claim it is verified; reconciling it against a provider's invoice is a
	// control plane's job. The same names are on an execution Attempt.
	Provider  string  `json:"provider,omitempty"`
	Model     string  `json:"model,omitempty"`
	TokensIn  int     `json:"tokens_in,omitempty"`
	TokensOut int     `json:"tokens_out,omitempty"`
	Cost      float64 `json:"cost,omitempty"`
	// Currency is ISO 4217 (`USD`); required when Cost is set.
	Currency string            `json:"currency,omitempty"`
	Meta     map[string]string `json:"meta,omitempty"`
}

var reCurrency = regexp.MustCompile(`^[A-Z]{3}$`)

// validateCost checks the cost fields any kind may carry.
func (e Evidence) validateCost() error {
	switch {
	case e.TokensIn < 0 || e.TokensOut < 0:
		return errors.New("evidence: tokens must not be negative")
	case e.Cost < 0:
		return errors.New("evidence: cost must not be negative")
	case e.Cost != 0 && e.Currency == "":
		return errors.New("evidence: cost needs a currency")
	case e.Currency != "" && !reCurrency.MatchString(e.Currency):
		return fmt.Errorf("evidence: currency %q is not an ISO 4217 code", e.Currency)
	}
	return nil
}

// MaxOutput is how much of a command's output an evidence record keeps.
const MaxOutput = 64 << 10

// CheckTimeout bounds one check.
var CheckTimeout = 10 * time.Minute

// Record writes an evidence record with the next sequence number and answers
// the record as written and its path. The file name sorts in time:
// 001-check.json, 002-note.json.
func Record(l spec.Layout, e Evidence) (Evidence, string, error) {
	if !ValidID(e.Task) {
		return e, "", fmt.Errorf("evidence: %q is not a task id", e.Task)
	}
	if e.Kind == "" {
		return e, "", errors.New("evidence: kind is required")
	}
	if err := e.validateCost(); err != nil {
		return e, "", err
	}
	dir := l.EvidenceDir(e.Task)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return e, "", err
	}
	prev, err := ListEvidence(l, e.Task)
	if err != nil {
		return e, "", err
	}
	e.Seq = len(prev) + 1
	if e.At == "" {
		e.At = Now()
	}
	if e.Output != "" && e.OutputSHA256 == "" {
		e.OutputSHA256 = hash(e.Output)
	}
	if len(e.Output) > MaxOutput {
		e.Output = e.Output[:MaxOutput] + "\n[truncated]"
	}
	b, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return e, "", err
	}
	p := filepath.Join(dir, fmt.Sprintf("%03d-%s.json", e.Seq, e.Kind))
	return e, p, os.WriteFile(p, append(b, '\n'), 0o644)
}

// ListEvidence answers a task's records in sequence order.
func ListEvidence(l spec.Layout, id string) ([]Evidence, error) {
	entries, err := os.ReadDir(l.EvidenceDir(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Evidence
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(l.EvidenceDir(id), e.Name()))
		if err != nil {
			return nil, err
		}
		var ev Evidence
		if err := json.Unmarshal(b, &ev); err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		out = append(out, ev)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Seq < out[j].Seq })
	return out, nil
}

// Verification is what RunChecks answers.
type Verification struct {
	Task     string     `json:"task"`
	Passed   bool       `json:"passed"`
	Evidence []Evidence `json:"evidence"`
	Paths    []string   `json:"paths"`
}

// RunChecks runs the task's checks — then the project's — in dir, records one
// evidence per check and answers whether all passed. Nothing is skipped
// after a failure: a reviewer wants to see every result, not the first.
// It does not move the task; the caller decides what the result means.
func RunChecks(ctx context.Context, l spec.Layout, t *Task, dir, by string) (*Verification, error) {
	proj, err := l.LoadProject()
	if err != nil {
		return nil, err
	}
	cmds := append(append([]string(nil), t.Checks...), proj.Verify...)
	v := &Verification{Task: t.ID, Passed: true}
	if len(cmds) == 0 {
		v.Passed = false
		e, p, err := Record(l, Evidence{Task: t.ID, Kind: "note", By: by, Dir: dir, Note: "no checks to run: add `checks:` to the task or `verify:` to project.md"})
		if err != nil {
			return nil, err
		}
		v.Evidence = append(v.Evidence, e)
		v.Paths = append(v.Paths, p)
		return v, nil
	}
	for _, c := range cmds {
		e, p, err := Record(l, RunCheck(ctx, t.ID, c, dir, by))
		if err != nil {
			return nil, err
		}
		v.Evidence = append(v.Evidence, e)
		v.Paths = append(v.Paths, p)
		if !e.Passed {
			v.Passed = false
		}
	}
	return v, nil
}

// RunCheck runs one command without a shell and answers the evidence of it.
func RunCheck(ctx context.Context, taskID, command, dir, by string) Evidence {
	e := Evidence{Task: taskID, Kind: "check", By: by, Command: command, Dir: dir}
	args := SplitCommand(command)
	if len(args) == 0 {
		e.ExitCode = -1
		e.Note = "empty command"
		return e
	}
	ctx, cancel := context.WithTimeout(ctx, CheckTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	e.Output = out.String()
	e.OutputSHA256 = hash(e.Output)
	switch {
	case err == nil:
		e.ExitCode = 0
		e.Passed = true
	case ctx.Err() != nil:
		e.ExitCode = -1
		e.Note = "timed out after " + CheckTimeout.String()
	default:
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			e.ExitCode = ee.ExitCode()
		} else {
			e.ExitCode = -1
			e.Note = err.Error()
		}
	}
	return e
}

// SplitCommand cuts a command line into a program and its arguments: spaces
// separate, quotes (single or double) group. No variables, no pipes, no
// globs — a check that needs a shell says so: `sh -c "go test ./... | tail -1"`.
func SplitCommand(s string) []string {
	var args []string
	var cur strings.Builder
	var inQuote rune
	has := false
	for _, r := range s {
		switch {
		case (r == '"' || r == '\'') && inQuote == 0:
			inQuote = r
			has = true
		case r == inQuote:
			inQuote = 0
		case (r == ' ' || r == '\t') && inQuote == 0:
			if has {
				args = append(args, cur.String())
				cur.Reset()
				has = false
			}
		default:
			cur.WriteRune(r)
			has = true
		}
	}
	if has {
		args = append(args, cur.String())
	}
	return args
}

func hash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
