package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/emersonjoe/trilha-spec/spec"
	"github.com/emersonjoe/trilha-spec/task"
)

func TestStdioRoundTrip(t *testing.T) {
	l, _, err := spec.Init(t.TempDir(), spec.InitOptions{Name: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	st := &task.Store{Layout: l}
	if err := l.SaveSpec(spec.NewSpecDoc("001-one", "One", "en", "")); err != nil {
		t.Fatal(err)
	}
	tk, _ := st.Create("One", func(x *task.Task) { x.Acceptance = []string{"ok"} })
	st.Move(tk.ID, task.Ready)

	ro := NewServer("trilha-spec", "test", Tools(l, false)...)
	if n := len(ro.Tools()); n != 6 {
		t.Fatalf("read-only tools = %d", n)
	}
	rw := NewServer("trilha-spec", "test", Tools(l, true)...)
	in := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"trilha_next"}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"trilha_move","arguments":{"id":"TASK-001","status":"running"}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"trilha_move","arguments":{"id":"TASK-001","status":"done"}}}`,
		`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"trilha_context","arguments":{"id":"TASK-001"}}}`,
		`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"nope"}}`,
		`not json`,
		`{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"trilha_list_specs","arguments":{"status":"draft"}}}`,
		`{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"trilha_spec_move","arguments":{"id":"001-one","status":"approved"}}}`,
	}, "\n") + "\n"
	var out bytes.Buffer
	if err := rw.ServeStdio(context.Background(), strings.NewReader(in), &out); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 10 {
		t.Fatalf("%d replies:\n%s", len(lines), out.String())
	}
	var resp struct {
		ID     int             `json:"id"`
		Result json.RawMessage `json:"result"`
		Error  *rpcError       `json:"error"`
	}
	json.Unmarshal([]byte(lines[1]), &resp)
	if !strings.Contains(string(resp.Result), `"trilha_move"`) {
		t.Fatalf("tools/list: %s", lines[1])
	}
	json.Unmarshal([]byte(lines[2]), &resp)
	if !strings.Contains(string(resp.Result), `TASK-001`) {
		t.Fatalf("next: %s", lines[2])
	}
	json.Unmarshal([]byte(lines[3]), &resp)
	if !strings.Contains(string(resp.Result), `is now running`) {
		t.Fatalf("move: %s", lines[3])
	}
	json.Unmarshal([]byte(lines[4]), &resp)
	if !strings.Contains(string(resp.Result), `"isError":true`) || !strings.Contains(string(resp.Result), "cannot move") {
		t.Fatalf("illegal move: %s", lines[4])
	}
	json.Unmarshal([]byte(lines[5]), &resp)
	if !strings.Contains(string(resp.Result), `# Task TASK-001`) {
		t.Fatalf("context: %s", lines[5])
	}
	json.Unmarshal([]byte(lines[6]), &resp)
	if resp.Error == nil || resp.Error.Code != codeInvalidParams {
		t.Fatalf("unknown tool: %s", lines[6])
	}
	json.Unmarshal([]byte(lines[7]), &resp)
	if resp.Error == nil || resp.Error.Code != codeParse {
		t.Fatalf("parse error: %s", lines[7])
	}
	json.Unmarshal([]byte(lines[8]), &resp)
	if !strings.Contains(string(resp.Result), `001-one`) {
		t.Fatalf("list specs: %s", lines[8])
	}
	json.Unmarshal([]byte(lines[9]), &resp)
	if !strings.Contains(string(resp.Result), `001-one is now approved`) {
		t.Fatalf("spec move: %s", lines[9])
	}
	// A paused project: next reports the reason instead of a list.
	p, _ := l.LoadProject()
	p.Pause("breaker:max_cost_per_hour")
	if err := l.SaveProject(p); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	rw.ServeStdio(context.Background(), strings.NewReader(`{"jsonrpc":"2.0","id":10,"method":"tools/call","params":{"name":"trilha_next"}}`+"\n"), &out)
	json.Unmarshal([]byte(strings.TrimSpace(out.String())), &resp)
	if !strings.Contains(string(resp.Result), `"isError":true`) || !strings.Contains(string(resp.Result), "project is paused: breaker:max_cost_per_hour") {
		t.Fatalf("paused next: %s", out.String())
	}
}
