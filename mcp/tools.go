package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/emersonjoe/trilha-spec/ai"
	"github.com/emersonjoe/trilha-spec/spec"
	"github.com/emersonjoe/trilha-spec/task"
)

// Tools answers the protocol's tool set over one layout. Reading is always
// offered; the tools that change state — a transition, an evidence record —
// only when write is true, and a tool not offered cannot be called.
func Tools(l spec.Layout, write bool) []*Tool {
	st := &task.Store{Layout: l}
	tools := []*Tool{
		{
			Name:        "trilha_list_tasks",
			Description: "List every task with id, title, status, milestone and dependencies. Filter by status with {\"status\": \"ready\"} or by milestone with {\"milestone\": \"M2\"}.",
			Schema:      json.RawMessage(`{"type":"object","properties":{"status":{"type":"string"},"milestone":{"type":"string"}}}`),
			Func: func(ctx context.Context, args json.RawMessage) (string, error) {
				var in struct{ Status, Milestone string }
				json.Unmarshal(args, &in)
				tasks, err := st.List()
				if err != nil {
					return "", err
				}
				byID, err := specsByID(l)
				if err != nil {
					return "", err
				}
				var out []map[string]any
				for _, t := range tasks {
					m := task.MilestoneOf(t, byID)
					if in.Status != "" && string(t.Status) != in.Status {
						continue
					}
					if in.Milestone != "" && m != in.Milestone {
						continue
					}
					out = append(out, map[string]any{"id": t.ID, "title": t.Title, "status": t.Status, "depends_on": t.DependsOn, "agent": t.Agent, "milestone": m})
				}
				return js(out), nil
			},
		},
		{
			Name:        "trilha_get_task",
			Description: "Read one task in full: acceptance criteria, checks, body and evidence.",
			Schema:      json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}`),
			Func: func(ctx context.Context, args json.RawMessage) (string, error) {
				var in struct{ ID string }
				json.Unmarshal(args, &in)
				t, err := st.Get(in.ID)
				if err != nil {
					return "", err
				}
				ev, err := task.ListEvidence(l, in.ID)
				if err != nil {
					return "", err
				}
				return js(map[string]any{"task": t, "evidence": ev}), nil
			},
		},
		{
			Name:        "trilha_next",
			Description: "Answer the tasks that can be picked up now: status ready with every dependency done, nearest milestone due date first, then dependency order. A paused project answers an error with the pause reason.",
			Func: func(ctx context.Context, args json.RawMessage) (string, error) {
				p, err := l.LoadProject()
				if err != nil {
					return "", err
				}
				if p.Paused {
					return "", fmt.Errorf("project is paused: %s (since %s)", p.PauseReason, p.PausedAt)
				}
				g, err := st.Graph()
				if err != nil {
					return "", err
				}
				byID, err := specsByID(l)
				if err != nil {
					return "", err
				}
				return js(task.ByDue(g.Ready(), func(t *task.Task) string { return task.MilestoneOf(t, byID) }, p.Due())), nil
			},
		},
		{
			Name:        "trilha_list_evidence",
			Description: "List the evidence of a task with a verdict per record — unsigned, valid or invalid — checked against the public keys in .trilha/keys.",
			Schema:      json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}`),
			Func: func(ctx context.Context, args json.RawMessage) (string, error) {
				var in struct{ ID string }
				json.Unmarshal(args, &in)
				list, err := task.ListEvidence(l, in.ID)
				if err != nil {
					return "", err
				}
				keys, err := task.ProjectKeys(l)
				if err != nil {
					return "", err
				}
				return js(keys.CheckAll(list)), nil
			},
		},
		{
			Name:        "trilha_context",
			Description: "Build the context pack of a task — project, constitution, spec, task, dependencies, evidence, agent — as Markdown (default) or JSON.",
			Schema:      json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"},"format":{"type":"string","enum":["markdown","json"]}},"required":["id"]}`),
			Func: func(ctx context.Context, args json.RawMessage) (string, error) {
				var in struct{ ID, Format string }
				json.Unmarshal(args, &in)
				p, err := ai.Build(l, in.ID)
				if err != nil {
					return "", err
				}
				if in.Format == "json" {
					return string(p.JSON()), nil
				}
				return p.Markdown(), nil
			},
		},
		{
			Name:        "trilha_list_specs",
			Description: "List every specification with id, title, status, issue and relations. Filter by status with {\"status\": \"approved\"}.",
			Schema:      json.RawMessage(`{"type":"object","properties":{"status":{"type":"string"}}}`),
			Func: func(ctx context.Context, args json.RawMessage) (string, error) {
				var in struct{ Status string }
				json.Unmarshal(args, &in)
				specs, err := l.ListSpecs()
				if err != nil {
					return "", err
				}
				out := []map[string]any{}
				for _, s := range specs {
					if in.Status != "" && string(s.Status) != in.Status {
						continue
					}
					out = append(out, map[string]any{"id": s.ID, "title": s.Title, "status": s.Status, "issue": s.Issue, "supersedes": s.Supersedes, "depends_on": s.DependsOn})
				}
				return js(out), nil
			},
		},
		{
			Name:        "trilha_coverage",
			Description: "The requirement traceability matrix: every external requirement a spec declares, the tasks that cover it, their status and how many evidence records each has. Narrow it to one spec with {\"spec\": \"009-name\"}.",
			Schema:      json.RawMessage(`{"type":"object","properties":{"spec":{"type":"string"}}}`),
			Func: func(ctx context.Context, args json.RawMessage) (string, error) {
				var in struct{ Spec string }
				json.Unmarshal(args, &in)
				rows, err := task.Cover(l, in.Spec)
				if err != nil {
					return "", err
				}
				return js(rows), nil
			},
		},
		{
			Name:        "trilha_graph",
			Description: "The dependency graph as Mermaid.",
			Func: func(ctx context.Context, args json.RawMessage) (string, error) {
				g, err := st.Graph()
				if err != nil {
					return "", err
				}
				return g.Mermaid(), nil
			},
		},
	}
	if !write {
		return tools
	}
	return append(tools,
		&Tool{
			Name:        "trilha_move",
			Description: "Move a task to a status (idea, spec, ready, running, verify, review, done, blocked, failed). Only legal transitions are accepted, and running needs every dependency done.",
			Schema:      json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"},"status":{"type":"string"}},"required":["id","status"]}`),
			Func: func(ctx context.Context, args json.RawMessage) (string, error) {
				var in struct{ ID, Status string }
				json.Unmarshal(args, &in)
				t, err := st.Move(in.ID, task.Status(in.Status))
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("%s is now %s", t.ID, t.Status), nil
			},
		},
		&Tool{
			Name:        "trilha_spec_move",
			Description: "Move a specification to a status (draft, approved, done, rejected, superseded). Only legal transitions are accepted.",
			Schema:      json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"},"status":{"type":"string"}},"required":["id","status"]}`),
			Func: func(ctx context.Context, args json.RawMessage) (string, error) {
				var in struct{ ID, Status string }
				json.Unmarshal(args, &in)
				s, err := l.MoveSpec(in.ID, spec.Status(in.Status))
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("%s is now %s", s.ID, s.Status), nil
			},
		},
		&Tool{
			Name:        "trilha_evidence",
			Description: "Record evidence on a task: a note, or an artifact with the files it produced.",
			Schema:      json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"},"kind":{"type":"string","enum":["note","artifact"]},"note":{"type":"string"},"files":{"type":"array","items":{"type":"string"}},"by":{"type":"string"}},"required":["id","kind"]}`),
			Func: func(ctx context.Context, args json.RawMessage) (string, error) {
				var in struct {
					ID, Kind, Note, By string
					Files              []string
				}
				json.Unmarshal(args, &in)
				if in.By == "" {
					in.By = "mcp"
				}
				if in.Kind != "note" && in.Kind != "artifact" {
					return "", fmt.Errorf("kind must be note or artifact")
				}
				e, p, err := task.Record(l, task.Evidence{Task: in.ID, Kind: in.Kind, Note: in.Note, Files: in.Files, By: in.By, Passed: true})
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("recorded #%d at %s", e.Seq, p), nil
			},
		},
		&Tool{
			Name:        "trilha_verify",
			Description: "Run the checks of a task in the project root, record each as evidence and answer whether all passed. Does not move the task.",
			Schema:      json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"},"by":{"type":"string"}},"required":["id"]}`),
			Func: func(ctx context.Context, args json.RawMessage) (string, error) {
				var in struct{ ID, By string }
				json.Unmarshal(args, &in)
				t, err := st.Get(in.ID)
				if err != nil {
					return "", err
				}
				if in.By == "" {
					in.By = "mcp"
				}
				v, err := task.RunChecks(ctx, l, t, l.Root, in.By)
				if err != nil {
					return "", err
				}
				var b strings.Builder
				fmt.Fprintf(&b, "passed: %v\n", v.Passed)
				for _, e := range v.Evidence {
					fmt.Fprintf(&b, "#%d %s exit %d\n", e.Seq, e.Command, e.ExitCode)
				}
				return b.String(), nil
			},
		},
	)
}

// specsByID indexes the specifications, for the fields a task inherits.
func specsByID(l spec.Layout) (map[string]*spec.Spec, error) {
	specs, err := l.ListSpecs()
	if err != nil {
		return nil, err
	}
	return task.SpecsByID(specs), nil
}

// js renders a tool's answer. No HTML escaping, so a comparator reads as
// `>=` in the host's transcript.
func js(v any) string {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return ""
	}
	return strings.TrimRight(b.String(), "\n")
}
