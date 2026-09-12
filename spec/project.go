package spec

import (
	"os"
)

// Project is .trilha/project.md: the first thing an agent reads. The front
// matter is what tools consume; the body is prose for the model.
type Project struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	DefaultAgent string `json:"default_agent,omitempty"`
	// Verify lists commands every task runs on top of its own checks.
	Verify []string `json:"verify,omitempty"`
	Body   string   `json:"body,omitempty"`
}

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
	d, err := Parse(b)
	if err != nil {
		return nil, err
	}
	return &Project{
		Name:         d.Fields.Get("name"),
		Description:  d.Fields.Get("description"),
		DefaultAgent: d.Fields.Get("default_agent"),
		Verify:       d.Fields.GetList("verify"),
		Body:         d.Body,
	}, nil
}

// LoadConstitution reads constitution.md as text; "" when absent.
func (l Layout) LoadConstitution() (string, error) {
	b, err := os.ReadFile(l.Constitution())
	if os.IsNotExist(err) {
		return "", nil
	}
	return string(b), err
}
