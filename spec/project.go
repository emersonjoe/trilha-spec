package spec

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Project is .trilha/project.md: the first thing an agent reads. The front
// matter is what tools consume; the body is prose for the model.
type Project struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	DefaultAgent string `json:"default_agent,omitempty"`
	// Verify lists commands every task runs on top of its own checks.
	Verify []string `json:"verify,omitempty"`
	// Milestones are the dated points the programme is paid and reported
	// against; specs and tasks name one in `milestone`.
	Milestones []Milestone `json:"milestones,omitempty"`
	// Repos names the other repositories this one depends on, alias → URL.
	// A task writes `depends_on: [app:TASK-004]` with one of these
	// aliases; the URL says which repository that is, and resolving it is
	// the runner's or the control plane's job.
	Repos map[string]string `json:"repos,omitempty"`
	// Limits are the project's envelope: numeric thresholds a runner reads
	// before it starts a task and a control plane trips on. The protocol
	// carries them; enforcing them is the runner's job, as with agent
	// manifests. Known keys are in Limits' documentation; any other key is
	// kept for the tool that wrote it.
	Limits map[string]float64 `json:"limits,omitempty"`
	// Paused stops the queue without losing it: `next` answers nothing and
	// says why. PauseReason is free text (`breaker:max_repeated_failure_class`
	// when a control plane tripped); PausedAt is RFC 3339 UTC.
	Paused      bool   `json:"paused,omitempty"`
	PauseReason string `json:"pause_reason,omitempty"`
	PausedAt    string `json:"paused_at,omitempty"`
	Body        string `json:"body,omitempty"`
	Fields      Fields `json:"-"`
}

// The limits the protocol names. A reader that does not know a key ignores it.
const (
	LimitMaxCostPerHour          = "max_cost_per_hour"          // in the currency of the evidence
	LimitMaxFailureRate          = "max_failure_rate"           // 0..1, over the last runs
	LimitMaxRepeatedFailureClass = "max_repeated_failure_class" // same failure_class in a row
)

// LoadProject reads project.md. A missing file is not an error: the project
// simply has no description yet.
func (l Layout) LoadProject() (*Project, error) {
	b, err := os.ReadFile(l.Project())
	if os.IsNotExist(err) {
		return &Project{}, nil
	}
	if err != nil {
		return nil, err
	}
	return ParseProject(b)
}

// ParseProject reads a project document.
func ParseProject(src []byte) (*Project, error) {
	d, err := Parse(src)
	if err != nil {
		return nil, err
	}
	p := &Project{
		Name:         d.Fields.Get("name"),
		Description:  d.Fields.Get("description"),
		DefaultAgent: d.Fields.Get("default_agent"),
		Verify:       d.Fields.GetList("verify"),
		Milestones:   milestonesFrom(d.Fields),
		Repos:        d.Fields.GetMap("repos"),
		Paused:       d.Fields.Get("paused") == "true",
		PauseReason:  d.Fields.Get("pause_reason"),
		PausedAt:     d.Fields.Get("paused_at"),
		Body:         d.Body,
		Fields:       d.Fields,
	}
	if raw := d.Fields.GetMap("limits"); len(raw) > 0 {
		p.Limits = map[string]float64{}
		var errs []string
		for k, v := range raw {
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				errs = append(errs, fmt.Sprintf("limits.%s %q is not a number", k, v))
				continue
			}
			p.Limits[k] = f
		}
		if len(errs) > 0 {
			sort.Strings(errs)
			return nil, errors.New("project: " + strings.Join(errs, "; "))
		}
	}
	return p, nil
}

// Validate checks what a reader relies on: limits are numbers, a pause has
// a time, a limit never goes below zero.
func (p *Project) Validate() error {
	var errs []string
	for k, v := range p.Limits {
		if v < 0 {
			errs = append(errs, fmt.Sprintf("limits.%s must not be negative", k))
		}
		if k == LimitMaxFailureRate && v > 1 {
			errs = append(errs, "limits.max_failure_rate is a share, 0..1")
		}
	}
	errs = append(errs, validateMilestones(p.Milestones)...)
	for alias, url := range p.Repos {
		if !ValidAlias(alias) {
			errs = append(errs, fmt.Sprintf("repos %q is not an alias (lowercase words joined by - or .)", alias))
		}
		if strings.TrimSpace(url) == "" {
			errs = append(errs, "repos."+alias+" has no url")
		}
	}
	if p.Paused && p.PausedAt == "" {
		errs = append(errs, "paused without paused_at")
	}
	if p.PausedAt != "" {
		if _, err := time.Parse(time.RFC3339, p.PausedAt); err != nil {
			errs = append(errs, "paused_at is not RFC 3339")
		}
	}
	sort.Strings(errs)
	if len(errs) > 0 {
		return errors.New("project: " + strings.Join(errs, "; "))
	}
	return nil
}

// Pause stops the queue with a reason, timestamped now.
func (p *Project) Pause(reason string) {
	p.Paused = true
	p.PauseReason = reason
	p.PausedAt = time.Now().UTC().Format(time.RFC3339)
}

// Resume clears the pause.
func (p *Project) Resume() {
	p.Paused = false
	p.PauseReason = ""
	p.PausedAt = ""
}

// Bytes writes the project back, keeping every field it does not own.
func (p *Project) Bytes() []byte {
	d := &Doc{Fields: p.Fields, Body: p.Body}
	d.Fields.Set("name", p.Name)
	if p.Description != "" || d.Fields.Has("description") {
		d.Fields.Set("description", p.Description)
	}
	if p.DefaultAgent != "" {
		d.Fields.Set("default_agent", p.DefaultAgent)
	} else {
		d.Fields.Delete("default_agent")
	}
	if p.Verify != nil || d.Fields.Has("verify") {
		d.Fields.SetList("verify", p.Verify)
	}
	if len(p.Milestones) > 0 {
		d.Fields.SetItems("milestones", milestoneFields(p.Milestones))
	} else {
		d.Fields.Delete("milestones")
	}
	if len(p.Repos) > 0 {
		d.Fields.SetMap("repos", p.Repos)
	} else {
		d.Fields.Delete("repos")
	}
	if len(p.Limits) > 0 {
		m := make(map[string]string, len(p.Limits))
		for k, v := range p.Limits {
			m[k] = strconv.FormatFloat(v, 'f', -1, 64)
		}
		d.Fields.SetMap("limits", m)
	} else {
		d.Fields.Delete("limits")
	}
	if p.Paused {
		d.Fields.Set("paused", "true")
		d.Fields.Set("pause_reason", p.PauseReason)
		d.Fields.Set("paused_at", p.PausedAt)
		if p.PauseReason == "" {
			d.Fields.Delete("pause_reason")
		}
	} else {
		for _, k := range []string{"paused", "pause_reason", "paused_at"} {
			d.Fields.Delete(k)
		}
	}
	return d.Bytes()
}

// SaveProject writes project.md.
func (l Layout) SaveProject(p *Project) error {
	if err := p.Validate(); err != nil {
		return err
	}
	return os.WriteFile(l.Project(), p.Bytes(), 0o644)
}

// LoadConstitution reads constitution.md as text; "" when absent.
func (l Layout) LoadConstitution() (string, error) {
	b, err := os.ReadFile(l.Constitution())
	if os.IsNotExist(err) {
		return "", nil
	}
	return string(b), err
}
