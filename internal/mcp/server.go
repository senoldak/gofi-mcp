package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"time"
)

const ProtocolVersion = "2025-03-26"
const ServerName = "gofi-mcp"
const ServerVersion = "0.1.0"

const callTimeout = 30 * time.Second

type rpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Server struct {
	registry *Registry
	in       io.Reader
	out      io.Writer
}

func NewServer(r *Registry, in io.Reader, out io.Writer) *Server {
	return &Server{registry: r, in: in, out: out}
}

func (s *Server) Serve(ctx context.Context) error {
	scanner := bufio.NewScanner(s.in)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	enc := json.NewEncoder(s.out)

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var msg rpcMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			log.Printf("gofi-mcp: invalid JSON-RPC line: %v", err)
			continue
		}
		if len(msg.ID) == 0 || string(msg.ID) == "null" {
			continue // notification — never respond
		}
		resp := s.handle(ctx, &msg)
		if err := enc.Encode(resp); err != nil {
			return fmt.Errorf("write response: %w", err)
		}
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read stdin: %w", err)
	}
	return nil
}

func (s *Server) handle(ctx context.Context, msg *rpcMessage) *rpcResponse {
	resp := &rpcResponse{JSONRPC: "2.0", ID: msg.ID}
	switch msg.Method {
	case "initialize":
		resp.Result = map[string]any{
			"protocolVersion": ProtocolVersion,
			"capabilities": map[string]any{
				"tools": map[string]any{"listChanged": false},
			},
			"serverInfo": map[string]any{
				"name":    ServerName,
				"version": ServerVersion,
			},
		}
	case "ping":
		resp.Result = map[string]any{}
	case "tools/list":
		resp.Result = map[string]any{"tools": s.registry.List()}
	case "tools/call":
		result, rpcErr := s.callTool(ctx, msg.Params)
		if rpcErr != nil {
			resp.Error = rpcErr
		} else {
			resp.Result = result
		}
	default:
		resp.Error = &rpcError{Code: -32601, Message: "method not found: " + msg.Method}
	}
	return resp
}

func (s *Server) callTool(ctx context.Context, params json.RawMessage) (any, *rpcError) {
	var p struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.Name == "" {
		return nil, &rpcError{Code: -32602, Message: "invalid params: name required"}
	}
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	out, err := s.registry.Call(ctx, p.Name, p.Arguments)
	if err != nil {
		return map[string]any{
			"content": []map[string]any{{"type": "text", "text": err.Error()}},
			"isError": true,
		}, nil
	}
	text, err := json.Marshal(out)
	if err != nil {
		return nil, &rpcError{Code: -32603, Message: "internal error: " + err.Error()}
	}
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": string(text)}},
	}, nil
}
