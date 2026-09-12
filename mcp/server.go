// Package mcp exposes the protocol to any MCP host — Claude Code, Cursor,
// an ai.Agent — over stdio: the same tasks, transitions and evidence the CLI
// offers, as tools a model can call. The JSON-RPC layer is adapted from
// github.com/emersonjoe/trilha/ai/mcp (MIT), trimmed to what a stdio server
// needs so this module keeps zero dependencies.
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
)

// ProtocolVersion is the MCP revision implemented.
const ProtocolVersion = "2025-03-26"

// ToolFunc executes a tool with the model's JSON arguments and answers text
// for the model. An error is reported to the model, not to the host.
type ToolFunc func(ctx context.Context, args json.RawMessage) (string, error)

// Tool is one function the model may call.
type Tool struct {
	Name        string
	Description string
	// Schema is a JSON Schema object for the arguments; nil means none.
	Schema json.RawMessage
	Func   ToolFunc
}

// Server answers initialize, ping, tools/list and tools/call.
type Server struct {
	name, version string
	tools         []*Tool
	byName        map[string]*Tool
}

// NewServer creates a server publishing the given tools.
func NewServer(name, version string, tools ...*Tool) *Server {
	s := &Server{name: name, version: version, byName: map[string]*Tool{}}
	for _, t := range tools {
		s.tools = append(s.tools, t)
		s.byName[t.Name] = t
	}
	return s
}

// Tools answers the names published, in order.
func (s *Server) Tools() []string {
	out := make([]string, 0, len(s.tools))
	for _, t := range s.tools {
		out = append(out, t.Name)
	}
	return out
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const (
	codeParse          = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternal       = -32603
)

// ToolInfo is a tool as listed to the host.
type ToolInfo struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// Content is one block of a call result.
type Content struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// CallResult is what tools/call answers.
type CallResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Handle processes one JSON-RPC message; nil means no reply (notification).
func (s *Server) Handle(ctx context.Context, msg []byte) []byte {
	var req request
	if err := json.Unmarshal(msg, &req); err != nil {
		return mustJSON(response{JSONRPC: "2.0", Error: &rpcError{Code: codeParse, Message: "parse error"}})
	}
	if req.JSONRPC != "2.0" || req.Method == "" {
		return mustJSON(response{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: codeInvalidRequest, Message: "invalid request"}})
	}
	if len(req.ID) == 0 {
		return nil
	}
	reply := func(result any) []byte {
		b, err := json.Marshal(result)
		if err != nil {
			return mustJSON(response{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: codeInternal, Message: err.Error()}})
		}
		return mustJSON(response{JSONRPC: "2.0", ID: req.ID, Result: b})
	}
	fail := func(code int, msg string) []byte {
		return mustJSON(response{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: code, Message: msg}})
	}
	switch req.Method {
	case "initialize":
		return reply(map[string]any{
			"protocolVersion": ProtocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
			"serverInfo":      map[string]string{"name": s.name, "version": s.version},
		})
	case "ping":
		return reply(map[string]any{})
	case "tools/list":
		infos := make([]ToolInfo, 0, len(s.tools))
		for _, t := range s.tools {
			schema := t.Schema
			if len(schema) == 0 {
				schema = json.RawMessage(`{"type":"object","properties":{}}`)
			}
			infos = append(infos, ToolInfo{Name: t.Name, Description: t.Description, InputSchema: schema})
		}
		return reply(map[string]any{"tools": infos})
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return fail(codeInvalidParams, "invalid params")
		}
		t, ok := s.byName[p.Name]
		if !ok {
			return fail(codeInvalidParams, "unknown tool: "+p.Name)
		}
		if len(p.Arguments) == 0 {
			p.Arguments = json.RawMessage("{}")
		}
		out, err := safeCall(ctx, t, p.Arguments)
		if err != nil {
			return reply(CallResult{Content: []Content{{Type: "text", Text: err.Error()}}, IsError: true})
		}
		return reply(CallResult{Content: []Content{{Type: "text", Text: out}}})
	default:
		return fail(codeMethodNotFound, "method not found: "+req.Method)
	}
}

func safeCall(ctx context.Context, t *Tool, args json.RawMessage) (out string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("tool %s panicked: %v", t.Name, r)
		}
	}()
	if t.Func == nil {
		return "", errors.New("tool has no implementation")
	}
	return t.Func(ctx, args)
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

// ServeStdio runs the server over newline-delimited JSON until EOF or ctx
// ends. stdout belongs to the protocol: a server that logs there corrupts
// the stream, so anything else goes to stderr.
func (s *Server) ServeStdio(ctx context.Context, r io.Reader, w io.Writer) error {
	br := bufio.NewReaderSize(r, 1<<20)
	var mu sync.Mutex
	for {
		line, err := br.ReadBytes('\n')
		line = bytes.TrimSpace(line)
		if len(line) > 0 {
			if out := s.Handle(ctx, line); out != nil {
				mu.Lock()
				_, werr := w.Write(append(out, '\n'))
				mu.Unlock()
				if werr != nil {
					return werr
				}
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if ctx.Err() != nil {
			return nil
		}
	}
}
