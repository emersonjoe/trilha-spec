package task

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/emersonjoe/trilha-spec/spec"
)

// Store reads and writes tasks in a layout. There is no cache: the files are
// the truth, and another process — an agent, a person with an editor — may
// have changed them since the last call.
type Store struct{ Layout spec.Layout }

// Open finds the layout above dir and answers a store on it.
func Open(dir string) (*Store, error) {
	l, err := spec.Find(dir)
	if err != nil {
		return nil, err
	}
	return &Store{Layout: l}, nil
}

// List answers every task, sorted by ID. A file that does not parse fails
// the whole list with its name: a silently skipped task is a task nobody
// will ever run.
func (s *Store) List() ([]*Task, error) {
	entries, err := os.ReadDir(s.Layout.Tasks())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []*Task
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(s.Layout.Tasks(), e.Name()))
		if err != nil {
			return nil, err
		}
		t, err := Parse(b)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		if want := strings.TrimSuffix(e.Name(), ".md"); t.ID != want {
			return nil, fmt.Errorf("%s: file name and id %q disagree", e.Name(), t.ID)
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// Get answers one task by ID.
func (s *Store) Get(id string) (*Task, error) {
	b, err := os.ReadFile(s.Layout.TaskFile(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("task %s: not found", id)
		}
		return nil, err
	}
	return Parse(b)
}

// Save validates and writes one task.
func (s *Store) Save(t *Task) error {
	if err := t.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(s.Layout.Tasks(), 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.Layout.TaskFile(t.ID), t.Bytes(), 0o644)
}

// NextID answers the next free TASK-NNN.
func (s *Store) NextID() (string, error) {
	tasks, err := s.List()
	if err != nil {
		return "", err
	}
	n := 0
	for _, t := range tasks {
		var k int
		if _, err := fmt.Sscanf(t.ID, "TASK-%d", &k); err == nil && k > n {
			n = k
		}
	}
	return fmt.Sprintf("TASK-%03d", n+1), nil
}

// Create writes a new task with the next ID and answers it.
func (s *Store) Create(title string, opts ...func(*Task)) (*Task, error) {
	id, err := s.NextID()
	if err != nil {
		return nil, err
	}
	t := &Task{ID: id, Title: title, Status: Idea, Created: Now()}
	for _, o := range opts {
		o(t)
	}
	if err := s.Save(t); err != nil {
		return nil, err
	}
	return t, nil
}

// Move loads a task, applies a transition and saves it. Moving to Running
// also requires every dependency to be done: the graph is the protocol's
// scheduler, and a task that starts before its inputs exist starts wrong.
func (s *Store) Move(id string, to Status) (*Task, error) {
	t, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if to == Running {
		g, err := s.Graph()
		if err != nil {
			return nil, err
		}
		if open := g.Blockers(id); len(open) > 0 {
			return nil, fmt.Errorf("task %s: cannot run, waiting on %s", id, strings.Join(open, ", "))
		}
	}
	// A task with a review policy does not close on one person's say-so: the
	// signed attestations have to be there, and the refusal says what is
	// missing rather than just saying no.
	if to == Done {
		q, err := QuorumOf(s.Layout, t)
		if err != nil {
			return nil, err
		}
		if !q.Met() {
			return nil, fmt.Errorf("task %s: cannot close, %s", id, q.Reason())
		}
	}
	if err := t.Move(to); err != nil {
		return nil, err
	}
	return t, s.Save(t)
}

// Graph builds the dependency graph of every task in the store.
func (s *Store) Graph() (*Graph, error) {
	tasks, err := s.List()
	if err != nil {
		return nil, err
	}
	return NewGraph(tasks)
}
