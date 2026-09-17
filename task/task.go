// Package task is the executable unit of the Trilha protocol: what an agent
// picks up, the states it moves through, the graph its dependencies form and
// the evidence it leaves behind. A task is one file, .trilha/tasks/TASK-NNN.md,
// so a human, a diff and an agent all read the same thing.
package task

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/emersonjoe/trilha-spec/execution"
	"github.com/emersonjoe/trilha-spec/spec"
)

// Status is where a task is in its life. The happy path runs top to bottom;
// Blocked and Failed are the two ways out of it, and both lead back to Ready.
type Status string

const (
	Idea    Status = "idea"    // written down, not specified
	Spec    Status = "spec"    // has a specification, criteria not settled
	Ready   Status = "ready"   // criteria settled; can be picked up when deps are done
	Running Status = "running" // an agent owns it, in a worktree
	Verify  Status = "verify"  // the agent finished; checks are running
	Review  Status = "review"  // checks passed; a reviewer decides
	Done    Status = "done"
	Blocked Status = "blocked" // waiting on something outside the graph
	Failed  Status = "failed"  // checks or the agent failed; needs rework
)

// Statuses lists every status in life order.
var Statuses = []Status{Idea, Spec, Ready, Running, Verify, Review, Done, Blocked, Failed}

// Transitions says which moves are legal. The protocol is strict here on
// purpose: a task that jumps from idea to done has no evidence, and the
// whole point is the evidence.
var Transitions = map[Status][]Status{
	Idea:    {Spec, Ready},
	Spec:    {Ready, Idea},
	Ready:   {Running, Blocked, Spec},
	Running: {Verify, Failed, Blocked, Ready},
	Verify:  {Review, Failed},
	Review:  {Done, Ready, Failed},
	Done:    {Ready},
	Blocked: {Ready},
	Failed:  {Ready},
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

// Valid answers whether s is one of the protocol's statuses.
func (s Status) Valid() bool {
	for _, t := range Statuses {
		if t == s {
			return true
		}
	}
	return false
}

// Task is the record. Fields is the front matter as read, so a key the
// protocol does not know survives a round trip.
type Task struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status Status `json:"status"`
	// Spec is the specification this task implements (a spec ID), if any.
	Spec string `json:"spec,omitempty"`
	// Agent is who executes it: an agent name from .trilha/agents/.
	Agent     string   `json:"agent,omitempty"`
	DependsOn []string `json:"depends_on,omitempty"`
	// Acceptance is what must be true to close, in words a reviewer checks.
	Acceptance []string `json:"acceptance,omitempty"`
	// Checks are commands that prove the acceptance; `verify` runs them and
	// records the result as evidence. No shell: a command is a program and
	// its arguments, split on spaces with double quotes respected.
	Checks           []string    `json:"checks,omitempty"`
	ExpectedFiles    []string    `json:"expected_files,omitempty"`
	Routes           []string    `json:"routes,omitempty"`
	Scenarios        []string    `json:"scenarios,omitempty"`
	Accessibility    []string    `json:"accessibility,omitempty"`
	SecurityControls []string    `json:"security_controls,omitempty"`
	MaxAttempts      int         `json:"max_attempts,omitempty"`
	TokenBudget      int         `json:"token_budget,omitempty"`
	Attempt          int         `json:"attempt,omitempty"`
	RetryOf          string      `json:"retry_of,omitempty"`
	FailureClass     string      `json:"failure_class,omitempty"`
	RepairReason     string      `json:"repair_reason,omitempty"`
	Created          string      `json:"created,omitempty"`
	Updated          string      `json:"updated,omitempty"`
	Body             string      `json:"body,omitempty"`
	Fields           spec.Fields `json:"-"`
}

var reID = regexp.MustCompile(`^TASK-[0-9]{3,}$`)

// ValidID answers whether id has the `TASK-NNN` shape.
func ValidID(id string) bool { return reID.MatchString(id) }

// Parse reads a task document.
func Parse(src []byte) (*Task, error) {
	d, err := spec.Parse(src)
	if err != nil {
		return nil, err
	}
	t := &Task{
		ID:               d.Fields.Get("id"),
		Title:            d.Fields.Get("title"),
		Status:           Status(d.Fields.Get("status")),
		Spec:             d.Fields.Get("spec"),
		Agent:            d.Fields.Get("agent"),
		DependsOn:        d.Fields.GetList("depends_on"),
		Acceptance:       d.Fields.GetList("acceptance"),
		Checks:           d.Fields.GetList("checks"),
		ExpectedFiles:    d.Fields.GetList("expected_files"),
		Routes:           d.Fields.GetList("routes"),
		Scenarios:        d.Fields.GetList("scenarios"),
		Accessibility:    d.Fields.GetList("accessibility"),
		SecurityControls: d.Fields.GetList("security_controls"),
		MaxAttempts:      parseInt(d.Fields.Get("max_attempts")),
		TokenBudget:      parseInt(d.Fields.Get("token_budget")),
		Attempt:          parseInt(d.Fields.Get("attempt")),
		RetryOf:          d.Fields.Get("retry_of"),
		FailureClass:     d.Fields.Get("failure_class"),
		RepairReason:     d.Fields.Get("repair_reason"),
		Created:          d.Fields.Get("created"),
		Updated:          d.Fields.Get("updated"),
		Body:             d.Body,
		Fields:           d.Fields,
	}
	if t.Status == "" {
		t.Status = Idea
	}
	return t, t.Validate()
}

// Validate checks what every reader relies on.
func (t *Task) Validate() error {
	var errs []string
	if !ValidID(t.ID) {
		errs = append(errs, fmt.Sprintf("id %q must look like TASK-001", t.ID))
	}
	if strings.TrimSpace(t.Title) == "" {
		errs = append(errs, "title is required")
	}
	if !t.Status.Valid() {
		errs = append(errs, fmt.Sprintf("status %q is not one of %v", t.Status, Statuses))
	}
	for _, d := range t.DependsOn {
		if !ValidID(d) {
			errs = append(errs, fmt.Sprintf("depends_on %q is not a task id", d))
		}
		if d == t.ID {
			errs = append(errs, "a task cannot depend on itself")
		}
	}
	if t.Status == Ready || t.Status == Running {
		if len(t.Acceptance) == 0 {
			errs = append(errs, string(t.Status)+" needs at least one acceptance criterion")
		}
	}
	if t.MaxAttempts < 0 || t.MaxAttempts > 10 {
		errs = append(errs, "max_attempts must be between 1 and 10 when set")
	}
	if t.TokenBudget < 0 {
		errs = append(errs, "token_budget cannot be negative")
	}
	if t.Attempt < 0 {
		errs = append(errs, "attempt cannot be negative")
	}
	if t.RetryOf != "" && !execution.ValidRunID(t.RetryOf) {
		errs = append(errs, "retry_of must be a run id")
	}
	if len(errs) > 0 {
		return errors.New("task " + t.ID + ": " + strings.Join(errs, "; "))
	}
	return nil
}

// Bytes writes the task back, protocol fields first and in a fixed order,
// unknown fields after, in the order they were read.
func (t *Task) Bytes() []byte {
	d := &spec.Doc{Body: t.Body}
	d.Fields.Set("id", t.ID)
	d.Fields.Set("title", t.Title)
	d.Fields.Set("status", string(t.Status))
	setOpt(&d.Fields, "spec", t.Spec)
	setOpt(&d.Fields, "agent", t.Agent)
	d.Fields.SetList("depends_on", t.DependsOn)
	d.Fields.SetList("acceptance", t.Acceptance)
	d.Fields.SetList("checks", t.Checks)
	d.Fields.SetList("expected_files", t.ExpectedFiles)
	d.Fields.SetList("routes", t.Routes)
	d.Fields.SetList("scenarios", t.Scenarios)
	d.Fields.SetList("accessibility", t.Accessibility)
	d.Fields.SetList("security_controls", t.SecurityControls)
	setIntOpt(&d.Fields, "max_attempts", t.MaxAttempts)
	setIntOpt(&d.Fields, "token_budget", t.TokenBudget)
	setIntOpt(&d.Fields, "attempt", t.Attempt)
	setOpt(&d.Fields, "retry_of", t.RetryOf)
	setOpt(&d.Fields, "failure_class", t.FailureClass)
	setOpt(&d.Fields, "repair_reason", t.RepairReason)
	setOpt(&d.Fields, "created", t.Created)
	setOpt(&d.Fields, "updated", t.Updated)
	for _, k := range t.Fields.Keys() {
		if d.Fields.Has(k) || known[k] {
			continue
		}
		if l := t.Fields.GetList(k); len(l) > 0 && t.Fields.Get(k) == "" {
			d.Fields.SetList(k, l)
		} else {
			d.Fields.Set(k, t.Fields.Get(k))
		}
	}
	return d.Bytes()
}

var known = map[string]bool{"id": true, "title": true, "status": true, "spec": true, "agent": true, "depends_on": true, "acceptance": true, "checks": true, "expected_files": true, "routes": true, "scenarios": true, "accessibility": true, "security_controls": true, "max_attempts": true, "token_budget": true, "attempt": true, "retry_of": true, "failure_class": true, "repair_reason": true, "created": true, "updated": true}

func setOpt(f *spec.Fields, k, v string) {
	if v != "" {
		f.Set(k, v)
	}
}

func setIntOpt(f *spec.Fields, key string, value int) {
	if value > 0 {
		f.Set(key, strconv.Itoa(value))
	}
}

func parseInt(value string) int {
	number, _ := strconv.Atoi(strings.TrimSpace(value))
	return number
}

// Now is the timestamp format the protocol writes: RFC 3339 in UTC, to the
// second. A variable so tests can pin it.
var Now = func() string { return time.Now().UTC().Format(time.RFC3339) }

// Move applies a transition, or answers why it cannot.
func (t *Task) Move(to Status) error {
	if !to.Valid() {
		return fmt.Errorf("task %s: %q is not a status", t.ID, to)
	}
	if !t.Status.CanMoveTo(to) {
		return fmt.Errorf("task %s: cannot move from %s to %s (allowed: %v)", t.ID, t.Status, to, Transitions[t.Status])
	}
	t.Status = to
	t.Updated = Now()
	return t.Validate()
}
