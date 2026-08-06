package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func runServer(t *testing.T, input string) (string, error) {
	t.Helper()
	in := strings.NewReader(input)
	var out bytes.Buffer
	r := NewRegistry(&fakeFetcher{body: []byte(`{"price":100}`)})
	s := NewServer(r, in, &out)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := s.Serve(ctx)
	return out.String(), err
}

func TestInitializeReturnsServerInfo(t *testing.T) {
	out, err := runServer(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`+"\n")
	if err != nil {
		t.Fatalf("Serve error: %v", err)
	}
	var resp struct {
		Result struct {
			ProtocolVersion string `json:"protocolVersion"`
			ServerInfo      struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"serverInfo"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("bad response JSON: %v\n%s", err, out)
	}
	if resp.Result.ProtocolVersion != ProtocolVersion {
		t.Fatalf("protocolVersion = %q", resp.Result.ProtocolVersion)
	}
	if resp.Result.ServerInfo.Name != ServerName {
		t.Fatalf("server name = %q", resp.Result.ServerInfo.Name)
	}
}

func TestPingReturnsEmptyResult(t *testing.T) {
	out, err := runServer(t, `{"jsonrpc":"2.0","id":2,"method":"ping"}`+"\n")
	if err != nil {
		t.Fatalf("Serve error: %v", err)
	}
	if !strings.Contains(out, `"result":{}`) {
		t.Fatalf("expected empty result, got: %s", out)
	}
}

func TestToolsListReturnsFifteen(t *testing.T) {
	out, err := runServer(t, `{"jsonrpc":"2.0","id":3,"method":"tools/list"}`+"\n")
	if err != nil {
		t.Fatalf("Serve error: %v", err)
	}
	var resp struct {
		Result struct {
			Tools []Tool `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("bad response JSON: %v\n%s", err, out)
	}
	if len(resp.Result.Tools) != 15 {
		t.Fatalf("expected 15 tools, got %d", len(resp.Result.Tools))
	}
}

func TestToolsCallReturnsTextContent(t *testing.T) {
	out, err := runServer(t, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"get_quote","arguments":{"ticker":"AAPL:NASDAQ"}}}`+"\n")
	if err != nil {
		t.Fatalf("Serve error: %v", err)
	}
	var resp struct {
		Result struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("bad response JSON: %v\n%s", err, out)
	}
	if len(resp.Result.Content) != 1 || resp.Result.Content[0].Type != "text" {
		t.Fatalf("unexpected content: %+v", resp.Result.Content)
	}
	if !strings.Contains(resp.Result.Content[0].Text, `"price"`) {
		t.Fatalf("text content missing price: %s", resp.Result.Content[0].Text)
	}
}

func TestToolsCallErrorSetsIsError(t *testing.T) {
	out, err := runServer(t, `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"get_quote","arguments":{}}}`+"\n")
	if err != nil {
		t.Fatalf("Serve error: %v", err)
	}
	var resp struct {
		Result struct {
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("bad response JSON: %v\n%s", err, out)
	}
	if !resp.Result.IsError {
		t.Fatal("expected isError=true for missing ticker")
	}
}

func TestUnknownMethodReturnsError(t *testing.T) {
	out, err := runServer(t, `{"jsonrpc":"2.0","id":6,"method":"bogus"}`+"\n")
	if err != nil {
		t.Fatalf("Serve error: %v", err)
	}
	var resp struct {
		Error struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("bad response JSON: %v\n%s", err, out)
	}
	if resp.Error.Code != -32601 {
		t.Fatalf("expected -32601, got %d", resp.Error.Code)
	}
}

func TestNotificationGetsNoResponse(t *testing.T) {
	out, err := runServer(t, `{"jsonrpc":"2.0","method":"notifications/initialized"}`+"\n")
	if err != nil {
		t.Fatalf("Serve error: %v", err)
	}
	if strings.TrimSpace(out) != "" {
		t.Fatalf("notification should not produce a response, got: %s", out)
	}
}

func TestServeReturnsOnEOF(t *testing.T) {
	out, err := runServer(t, "")
	if err != nil {
		t.Fatalf("Serve error: %v", err)
	}
	if out != "" {
		t.Fatalf("expected no output, got: %s", out)
	}
}
