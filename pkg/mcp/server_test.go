package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// startTestSession connects a real client to the server over in-memory transports
func startTestSession(t *testing.T) *mcp.ClientSession {
	t.Helper()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	server := NewServer()
	go func() {
		_ = server.mcp.Run(context.Background(), serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	session, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect failed: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

// writeFixture creates a markdown doc in a temp dir and returns its path
func writeFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.md")
	content := "# Fixture\n\n## Problem\n\nIt is slow.\n\n## Notes\n\nSome notes.\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func callTool(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) map[string]any {
	t.Helper()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s call failed: %v", name, err)
	}
	if result.IsError {
		text := ""
		if len(result.Content) > 0 {
			if tc, ok := result.Content[0].(*mcp.TextContent); ok {
				text = tc.Text
			}
		}
		t.Fatalf("%s returned error: %s", name, text)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	var payload map[string]any
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatalf("%s returned non-JSON payload: %s", name, text)
	}
	return payload
}

// callToolExpectError asserts the tool call fails and returns the error text
func callToolExpectError(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) string {
	t.Helper()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return err.Error()
	}
	if !result.IsError {
		t.Fatalf("%s should have failed", name)
	}
	if tc, ok := result.Content[0].(*mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}

func TestServerRegistersAllTools(t *testing.T) {
	session := startTestSession(t)
	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{
		"comments_list", "comments_get", "comments_status", "comments_add",
		"comments_reply", "comments_resolve", "comments_suggest", "comments_accept",
		"comments_reject", "comments_batch_add", "comments_batch_reply",
		"comments_gate", "comments_request_review",
		"comments_get_template", "comments_validate", "comments_seed",
		"comments_reanchor",
	}
	names := map[string]bool{}
	for _, tool := range result.Tools {
		names[tool.Name] = true
	}
	for _, want := range expected {
		if !names[want] {
			t.Errorf("tool %s not registered", want)
		}
	}
	if len(result.Tools) != len(expected) {
		t.Errorf("expected %d tools, got %d", len(expected), len(result.Tools))
	}
}

func TestAddListRoundTripSnakeCase(t *testing.T) {
	session := startTestSession(t)
	doc := writeFixture(t)

	added := callTool(t, session, "comments_add", map[string]any{
		"filepath": doc, "author": "eric", "text": "too vague", "line": 5, "blocking": true,
	})
	if added["success"] != true {
		t.Fatalf("add failed: %v", added)
	}

	listed := callTool(t, session, "comments_list", map[string]any{"filepath": doc})
	comments := listed["comments"].([]any)
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
	c := comments[0].(map[string]any)
	// snake_case wire shape, unified with CLI
	for _, key := range []string{"id", "author", "line", "blocking", "section_path", "anchor_confidence"} {
		if _, ok := c[key]; !ok {
			t.Errorf("comment payload missing snake_case key %q (keys: %v)", key, c)
		}
	}
	if c["blocking"] != true || c["line"] != float64(5) {
		t.Errorf("blocking/line not round-tripped: %v", c)
	}
	if len(c["id"].(string)) > 8 {
		t.Errorf("expected short ID, got %q", c["id"])
	}
}

func TestGateBlockingLifecycle(t *testing.T) {
	session := startTestSession(t)
	doc := writeFixture(t)

	callTool(t, session, "comments_add", map[string]any{
		"filepath": doc, "author": "eric", "text": "must fix", "line": 5, "blocking": true,
	})
	gate := callTool(t, session, "comments_gate", map[string]any{"filepath": doc})
	if gate["decision"] != "changes_requested" {
		t.Errorf("expected changes_requested with open blocking comment, got %v", gate["decision"])
	}

	listed := callTool(t, session, "comments_list", map[string]any{"filepath": doc})
	id := listed["comments"].([]any)[0].(map[string]any)["id"].(string)
	callTool(t, session, "comments_resolve", map[string]any{"filepath": doc, "thread_id": id})

	gate = callTool(t, session, "comments_gate", map[string]any{"filepath": doc})
	if gate["decision"] != "approved" {
		t.Errorf("expected approved after resolve, got %v", gate["decision"])
	}
}

func TestHumanZoneResolveRefused(t *testing.T) {
	session := startTestSession(t)
	doc := writeFixture(t)

	// design-doc template marks "Problem" as zone: human; seeding records the template
	seeded := callTool(t, session, "comments_seed", map[string]any{
		"filepath": doc, "template": "design-doc",
	})
	if seeded["seeded_count"].(float64) < 1 {
		t.Fatalf("expected seeded threads: %v", seeded)
	}

	// find a seeded thread anchored in the Problem section (line 3 heading)
	listed := callTool(t, session, "comments_list", map[string]any{"filepath": doc})
	var problemThread string
	for _, raw := range listed["comments"].([]any) {
		c := raw.(map[string]any)
		if strings.Contains(c["section_path"].(string), "Problem") {
			problemThread = c["id"].(string)
			break
		}
	}
	if problemThread == "" {
		t.Fatal("no seeded thread in Problem section")
	}

	errText := callToolExpectError(t, session, "comments_resolve", map[string]any{
		"filepath": doc, "thread_id": problemThread,
	})
	if !strings.Contains(errText, "human-decision zone") {
		t.Errorf("expected human-zone refusal, got: %s", errText)
	}
}

func TestReanchorMovesComment(t *testing.T) {
	session := startTestSession(t)
	doc := writeFixture(t)

	callTool(t, session, "comments_add", map[string]any{
		"filepath": doc, "author": "eric", "text": "note", "line": 5,
	})
	listed := callTool(t, session, "comments_list", map[string]any{"filepath": doc})
	id := listed["comments"].([]any)[0].(map[string]any)["id"].(string)

	moved := callTool(t, session, "comments_reanchor", map[string]any{
		"filepath": doc, "moves": []any{map[string]any{"comment_id": id, "line": 9}},
	})
	result := moved["results"].([]any)[0].(map[string]any)
	if result["moved"] != true || result["line"] != float64(9) {
		t.Errorf("reanchor failed: %v", result)
	}
}
