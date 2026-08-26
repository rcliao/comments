package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rcliao/comments/pkg/comment"
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
		"comments_analyze",
		"comments_reply", "comments_resolve", "comments_suggest", "comments_accept",
		"comments_batch_accept",
		"comments_reject", "comments_batch_add", "comments_batch_reply",
		"comments_gate", "comments_request_review", "comments_check_review",
		"comments_inbox",
		"comments_get_template", "comments_validate",
		"comments_new", "comments_context", "comments_bundle_index",
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

	// ToolNames feeds the serve-mcp startup banner; pin it to what is actually
	// registered so the banner can never drift behind again.
	reported := NewServer().ToolNames()
	if len(reported) != len(result.Tools) {
		t.Errorf("ToolNames reports %d tools, server registered %d", len(reported), len(result.Tools))
	}
	for _, name := range reported {
		if !names[name] {
			t.Errorf("ToolNames reports %s, which is not registered", name)
		}
	}
}

func TestKnowledgeBundleToolsRoundTrip(t *testing.T) {
	session := startTestSession(t)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".comments"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := "bundle: MCP Knowledge\nversion: 1\nokf_version: \"0.2\"\nroot: docs\ncollections:\n  plans:\n    path: plans\n    type: Plan\n    templates: [plan]\n"
	if err := os.WriteFile(filepath.Join(root, ".comments", "bundle.yaml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	created := callTool(t, session, "comments_new", map[string]any{
		"name": "cache-policy", "template": "plan", "bundle_path": root,
	})
	path := created["path"].(string)
	if !strings.HasSuffix(filepath.ToSlash(path), "/docs/plans/cache-policy.md") {
		t.Fatalf("unexpected created path: %v", created)
	}
	context := callTool(t, session, "comments_context", map[string]any{"filepath": path, "for": "drafting"})
	document := context["document"].(map[string]any)
	if document["template"] != "plan" || document["type"] != "Plan" {
		t.Fatalf("unexpected context document: %v", document)
	}
	implementation := callTool(t, session, "comments_context", map[string]any{"filepath": path, "for": "implementation"})
	direct, err := comment.BuildDocumentContext(path, comment.ContextOptions{For: "implementation"})
	if err != nil {
		t.Fatal(err)
	}
	implementationView := implementation["implementation"].(map[string]any)
	if implementationView["overall_status"] != direct.Implementation.OverallStatus {
		t.Fatalf("MCP context drifted from shared implementation context: %v vs %#v", implementationView, direct.Implementation)
	}
	indexed := callTool(t, session, "comments_bundle_index", map[string]any{"path": root})
	if indexed["indexed"] != true {
		t.Fatalf("bundle index failed: %v", indexed)
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

	data, err := os.ReadFile(doc)
	if err != nil {
		t.Fatal(err)
	}
	frontmatter := "---\ncomments:\n  template: design-doc\n---\n\n"
	if err := os.WriteFile(doc, []byte(frontmatter+string(data)), 0o644); err != nil {
		t.Fatal(err)
	}
	added := callTool(t, session, "comments_add", map[string]any{
		"filepath": doc, "author": "agent", "text": "human decision", "anchor": "It is slow.",
	})
	problemThread := added["comment_id"].(string)

	listed := callTool(t, session, "comments_list", map[string]any{"filepath": doc})
	if len(listed["comments"].([]any)) != 1 {
		t.Fatalf("expected explicit agent annotation: %v", listed)
	}

	errText := callToolExpectError(t, session, "comments_resolve", map[string]any{
		"filepath": doc, "thread_id": problemThread,
	})
	if !strings.Contains(errText, "human-decision zone") {
		t.Errorf("expected human-zone refusal, got: %s", errText)
	}
}

// TestNonBlockingReviewFlow exercises the durable review-handle loop:
// request (non-blocking) -> check pending -> human signoff via CLI-style
// sidecar write -> check completed.
func TestNonBlockingReviewFlow(t *testing.T) {
	session := startTestSession(t)
	doc := writeFixture(t)

	// A sidecar must exist for the gate evaluation after signoff
	callTool(t, session, "comments_add", map[string]any{
		"filepath": doc, "author": "eric", "text": "tighten this", "line": 5,
	})

	requested := callTool(t, session, "comments_request_review", map[string]any{
		"filepath": doc, "blocking": false,
	})
	if requested["status"] != "requested" {
		t.Fatalf("expected status requested, got %v", requested)
	}
	since, ok := requested["since"].(string)
	if !ok || since == "" {
		t.Fatalf("expected a since handle, got %v", requested)
	}
	if _, err := time.Parse(time.RFC3339, since); err != nil {
		t.Fatalf("since is not RFC3339: %q (%v)", since, err)
	}

	// No signoff yet: pending
	check := callTool(t, session, "comments_check_review", map[string]any{
		"filepath": doc, "since": since,
	})
	if check["status"] != "pending" {
		t.Fatalf("expected pending before signoff, got %v", check)
	}

	// Human signs off the CLI way: AddReviewRecord + SaveToSidecar
	loaded, _, err := comment.LoadFromSidecar(doc)
	if err != nil {
		t.Fatal(err)
	}
	comment.AddReviewRecord(loaded, "eric", "", "looks good", false)
	if err := comment.SaveToSidecar(doc, loaded); err != nil {
		t.Fatal(err)
	}

	check = callTool(t, session, "comments_check_review", map[string]any{
		"filepath": doc, "since": since,
	})
	if check["status"] != "review_completed" {
		t.Fatalf("expected review_completed after signoff, got %v", check)
	}
	review, ok := check["review"].(map[string]any)
	if !ok || review["author"] != "eric" {
		t.Errorf("expected review record by eric, got %v", check["review"])
	}
	if check["gate_decision"] != "approved" {
		t.Errorf("expected gate approved (only non-blocking comment), got %v", check["gate_decision"])
	}
	if _, ok := check["files"].([]any); !ok {
		t.Errorf("expected files array in completed payload, got %v", check["files"])
	}
}

func TestCheckReviewRejectsBadSince(t *testing.T) {
	session := startTestSession(t)
	doc := writeFixture(t)
	callTool(t, session, "comments_add", map[string]any{
		"filepath": doc, "author": "eric", "text": "note", "line": 5,
	})
	errText := callToolExpectError(t, session, "comments_check_review", map[string]any{
		"filepath": doc, "since": "not-a-timestamp",
	})
	if !strings.Contains(errText, "RFC3339") {
		t.Errorf("expected RFC3339 error, got: %s", errText)
	}
}

func TestInboxTool(t *testing.T) {
	session := startTestSession(t)
	doc := writeFixture(t)

	// Thread 1: blocking, no replies — always in the inbox
	callTool(t, session, "comments_add", map[string]any{
		"filepath": doc, "author": "eric", "text": "must fix", "line": 5, "blocking": true,
	})
	// Thread 2: non-blocking with a reply — in the inbox via new_reply
	callTool(t, session, "comments_add", map[string]any{
		"filepath": doc, "author": "eric", "text": "question", "line": 9,
	})
	// Thread 3: non-blocking, no replies — not in the inbox
	callTool(t, session, "comments_add", map[string]any{
		"filepath": doc, "author": "eric", "text": "just a note", "line": 9,
	})

	listed := callTool(t, session, "comments_list", map[string]any{"filepath": doc})
	var questionID string
	for _, raw := range listed["comments"].([]any) {
		c := raw.(map[string]any)
		if c["text"] == "question" {
			questionID = c["id"].(string)
		}
	}
	if questionID == "" {
		t.Fatal("question thread not found")
	}
	callTool(t, session, "comments_reply", map[string]any{
		"filepath": doc, "thread_id": questionID, "author": "claude", "text": "answered inline",
	})

	inbox := callTool(t, session, "comments_inbox", map[string]any{"filepath": doc})
	if inbox["count"] != float64(2) {
		t.Fatalf("expected 2 inbox items, got %v", inbox)
	}
	byText := map[string]map[string]any{}
	for _, raw := range inbox["items"].([]any) {
		item := raw.(map[string]any)
		thread := item["thread"].(map[string]any)
		byText[thread["text"].(string)] = item
	}
	blockingItem, ok := byText["must fix"]
	if !ok {
		t.Fatalf("blocking thread missing from inbox: %v", byText)
	}
	if reasons := blockingItem["reasons"].([]any); len(reasons) != 1 || reasons[0] != "blocking" {
		t.Errorf("expected [blocking] reasons, got %v", reasons)
	}
	replyItem, ok := byText["question"]
	if !ok {
		t.Fatalf("replied thread missing from inbox: %v", byText)
	}
	if reasons := replyItem["reasons"].([]any); len(reasons) != 1 || reasons[0] != "new_reply" {
		t.Errorf("expected [new_reply] reasons, got %v", reasons)
	}
	last := replyItem["last_reply"].(map[string]any)
	if last["author"] != "claude" || last["text"] != "answered inline" {
		t.Errorf("last_reply not surfaced: %v", last)
	}
	// snake_case thread payload (shared commentJSON shape)
	thread := replyItem["thread"].(map[string]any)
	if _, ok := thread["section_path"]; !ok {
		t.Errorf("thread payload missing snake_case keys: %v", thread)
	}

	// since in the future filters the reply-driven item but keeps blocking
	future := time.Now().Add(time.Hour).Format(time.RFC3339)
	inbox = callTool(t, session, "comments_inbox", map[string]any{"filepath": doc, "since": future})
	if inbox["count"] != float64(1) {
		t.Fatalf("expected only the blocking item with future since, got %v", inbox)
	}
	item := inbox["items"].([]any)[0].(map[string]any)
	if item["thread"].(map[string]any)["text"] != "must fix" {
		t.Errorf("expected blocking thread to survive since filter, got %v", item)
	}

	// resolved threads leave the inbox
	callTool(t, session, "comments_resolve", map[string]any{"filepath": doc, "thread_id": questionID})
	inbox = callTool(t, session, "comments_inbox", map[string]any{"filepath": doc})
	if inbox["count"] != float64(1) {
		t.Errorf("expected 1 item after resolving replied thread, got %v", inbox)
	}
}

func TestInboxOnDirectory(t *testing.T) {
	session := startTestSession(t)
	doc := writeFixture(t)
	dir := filepath.Dir(doc)

	callTool(t, session, "comments_add", map[string]any{
		"filepath": doc, "author": "eric", "text": "must fix", "line": 5, "blocking": true,
	})
	inbox := callTool(t, session, "comments_inbox", map[string]any{"filepath": dir})
	if inbox["count"] != float64(1) {
		t.Fatalf("expected 1 item for directory inbox, got %v", inbox)
	}
	if inbox["items"].([]any)[0].(map[string]any)["file"] == "" {
		t.Error("inbox item missing file path")
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
