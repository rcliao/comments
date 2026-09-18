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

// addOne adds a single root comment (comments_add takes an array; one comment
// is an array of one) and returns the new thread's ID.
func addOne(t *testing.T, session *mcp.ClientSession, doc string, spec map[string]any) string {
	t.Helper()
	result := callTool(t, session, "comments_add", map[string]any{
		"filepath": doc, "comments": []any{spec},
	})
	added, ok := result["added"].([]any)
	if !ok || len(added) != 1 || result["count"] != float64(1) {
		t.Fatalf("expected exactly one added comment, got %v", result)
	}
	return added[0].(map[string]any)["id"].(string)
}

// listThreads is comments_get without a comment_id: every thread root.
func listThreads(t *testing.T, session *mcp.ClientSession, doc string) []any {
	t.Helper()
	listed := callTool(t, session, "comments_get", map[string]any{"filepath": doc})
	return listed["comments"].([]any)
}

// withHumanZoneTemplate prepends design-doc frontmatter to the fixture, which
// makes "Problem" a zone: human section.
func withHumanZoneTemplate(t *testing.T, doc string) {
	t.Helper()
	data, err := os.ReadFile(doc)
	if err != nil {
		t.Fatal(err)
	}
	frontmatter := "---\ncomments:\n  template: design-doc\n---\n\n"
	if err := os.WriteFile(doc, []byte(frontmatter+string(data)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestServerRegistersAllTools(t *testing.T) {
	session := startTestSession(t)
	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	// The exact agent surface. A new tool must be added here on purpose: every
	// extra tool is another path an agent can pick instead of the intended one.
	expected := []string{
		"comments_new", "comments_context",
		"comments_validate", "comments_analyze",
		"comments_add",
		"comments_watch", "comments_inbox", "comments_get",
		"comments_reply", "comments_suggest", "comments_reanchor",
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
		t.Errorf("expected %d tools, got %d: %v", len(expected), len(result.Tools), names)
	}

	// Human decisions (accepting or rejecting a suggestion, the verdict, the
	// gate itself) must never be agent-reachable: they happen in `comments view`
	// / `comments serve`. The rest of this list was folded into the tools above
	// (comments_get_template into comments_context's brief); a name coming back
	// means the surface has forked again.
	removed := []string{
		"comments_accept", "comments_reject", "comments_batch_accept",
		"comments_gate", "comments_request_review", "comments_check_review",
		"comments_resolve", "comments_list", "comments_status",
		"comments_batch_add", "comments_batch_reply", "comments_bundle_index",
		"comments_get_template",
	}
	for _, gone := range removed {
		if names[gone] {
			t.Errorf("tool %s must not be registered over MCP", gone)
		}
	}

	// ToolNames feeds the serve-mcp startup banner; pin it to what is actually
	// registered so the banner can never drift behind again.
	reported := NewServer().ToolNames()
	if len(reported) != len(expected) {
		t.Errorf("ToolNames reports %d tools, want %d: %v", len(reported), len(expected), reported)
	}
	for _, name := range reported {
		if !names[name] {
			t.Errorf("ToolNames reports %s, which is not registered", name)
		}
	}
}

// An MCP caller is an agent by construction, so the suggestion decision has no
// tool at all — not a guarded one. Pinned separately from the catalog test so
// the failure names the contract that broke.
func TestAgentCannotAcceptOrRejectSuggestion(t *testing.T) {
	session := startTestSession(t)
	doc := writeFixture(t)
	suggested := callTool(t, session, "comments_suggest", map[string]any{
		"filepath": doc, "author": "claude", "text": "improve notes",
		"start_line": 9, "end_line": 9,
		"original_text": "Some notes.", "proposed_text": "Better notes.",
	})
	id := suggested["suggestion_id"].(string)

	for _, tool := range []string{"comments_accept", "comments_reject"} {
		result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name: tool, Arguments: map[string]any{"filepath": doc, "suggestion_id": id},
		})
		if err == nil && !result.IsError {
			t.Errorf("%s must fail as an unknown tool, got %v", tool, result.Content)
		}
	}

	// And the attempt left the document and the suggestion untouched
	content, err := os.ReadFile(doc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "Some notes.") {
		t.Errorf("document changed without a human decision:\n%s", content)
	}
	inbox := callTool(t, session, "comments_inbox", map[string]any{"filepath": doc})
	if pending := inbox["pending_suggestions"].([]any); len(pending) != 1 {
		t.Errorf("suggestion should still be pending, got %v", pending)
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
	// comments_get_template is gone: the drafting context is the agent's only
	// route to its writing brief, so a missing brief means drafting blind.
	brief, ok := context["brief"].(map[string]any)
	if !ok || brief["template"] != "plan" {
		t.Fatalf("drafting context must carry the template brief, got %v", context["brief"])
	}
	sections, _ := brief["sections"].([]any)
	if len(sections) == 0 || sections[0].(map[string]any)["heading"] == "" {
		t.Errorf("brief should list the template's sections: %v", brief)
	}
	if brief["marker_prefix"] == "" {
		t.Errorf("brief should name the ambiguity marker: %v", brief)
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
}

func TestAddListRoundTripSnakeCase(t *testing.T) {
	session := startTestSession(t)
	doc := writeFixture(t)

	id := addOne(t, session, doc, map[string]any{
		"author": "eric", "text": "too vague", "line": 5, "blocking": true,
	})

	comments := listThreads(t, session, doc)
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
	// The ID add hands back is the one every later call addresses
	if c["id"] != id {
		t.Errorf("add returned id %q, get lists %v", id, c["id"])
	}
	if len(id) > 8 {
		t.Errorf("expected short ID, got %q", id)
	}
}

// One bad comment must not leave the others behind: an agent that retries the
// whole call after fixing item 2 would otherwise duplicate item 1.
func TestAddIsAtomicAndNamesFailingItem(t *testing.T) {
	session := startTestSession(t)
	doc := writeFixture(t)

	errText := callToolExpectError(t, session, "comments_add", map[string]any{
		"filepath": doc,
		"comments": []any{
			map[string]any{"author": "claude", "text": "fine", "anchor": "It is slow."},
			map[string]any{"author": "claude", "text": "dangling", "anchor": "no such line in the document"},
		},
	})
	if !strings.Contains(errText, "comment 2") {
		t.Errorf("error should name the failing item, got: %s", errText)
	}

	listed := callTool(t, session, "comments_get", map[string]any{"filepath": doc})
	if listed["total"] != float64(0) {
		t.Errorf("rejected add must apply nothing, got %v", listed)
	}
}

// comments_get with no comment_id is the history view: resolved threads are
// included (a lookup most often wants a closed thread) unless unresolved_only.
func TestGetListsResolvedUnlessUnresolvedOnly(t *testing.T) {
	session := startTestSession(t)
	doc := writeFixture(t)

	openID := addOne(t, session, doc, map[string]any{"author": "eric", "text": "open", "line": 5})
	closedID := addOne(t, session, doc, map[string]any{"author": "eric", "text": "closed", "line": 9})
	replied := callTool(t, session, "comments_reply", map[string]any{
		"filepath": doc,
		"replies":  []any{map[string]any{"thread_id": closedID, "author": "claude", "text": "fixed", "resolve": true}},
	})
	if replied["count"] != float64(1) || len(replied["replied"].([]any)) != 1 || replied["resolved"].([]any)[0] != closedID {
		t.Errorf("reply+resolve result = %v", replied)
	}

	all := callTool(t, session, "comments_get", map[string]any{"filepath": doc})
	if all["total"] != float64(2) {
		t.Errorf("resolved threads must be listed by default, got %v", all["total"])
	}
	open := callTool(t, session, "comments_get", map[string]any{"filepath": doc, "unresolved_only": true})
	if open["total"] != float64(1) || open["comments"].([]any)[0].(map[string]any)["id"] != openID {
		t.Errorf("unresolved_only should leave only %s, got %v", openID, open)
	}

	// Replies come back nested under their root, not as flat items
	one := callTool(t, session, "comments_get", map[string]any{"filepath": doc, "comment_id": closedID})
	got := one["comment"].(map[string]any)
	if got["resolved"] != true || got["reply_count"] != float64(1) {
		t.Errorf("single lookup = %v", got)
	}
	if replies := got["replies"].([]any); replies[0].(map[string]any)["text"] != "fixed" {
		t.Errorf("replies not nested: %v", got)
	}
	if _, ok := one["context_lines"]; !ok {
		t.Errorf("single lookup should carry context_lines: %v", one)
	}
}

func TestHumanZoneResolveRefused(t *testing.T) {
	session := startTestSession(t)
	doc := writeFixture(t)
	withHumanZoneTemplate(t, doc)

	problemThread := addOne(t, session, doc, map[string]any{
		"author": "agent", "text": "human decision", "anchor": "It is slow.",
	})

	errText := callToolExpectError(t, session, "comments_reply", map[string]any{
		"filepath": doc,
		"replies":  []any{map[string]any{"thread_id": problemThread, "author": "agent", "text": "done", "resolve": true}},
	})
	if !strings.Contains(errText, "human-decision zone") {
		t.Errorf("expected human-zone refusal, got: %s", errText)
	}

	// Replying without resolving is the path the refusal points the agent to
	callTool(t, session, "comments_reply", map[string]any{
		"filepath": doc,
		"replies":  []any{map[string]any{"thread_id": problemThread, "author": "agent", "text": "my input"}},
	})
	thread := listThreads(t, session, doc)[0].(map[string]any)
	if thread["resolved"] == true || thread["reply_count"] != float64(1) {
		t.Errorf("plain reply in a human zone should land and leave it open: %v", thread)
	}
}

// A refused resolve must not leave half a batch applied: the agent is told to
// change the call and send it again, which would double the reply that landed.
func TestReplyIsAtomicAcrossZoneRefusal(t *testing.T) {
	session := startTestSession(t)
	doc := writeFixture(t)
	withHumanZoneTemplate(t, doc)

	notesThread := addOne(t, session, doc, map[string]any{
		"author": "eric", "text": "tidy this", "anchor": "Some notes.",
	})
	problemThread := addOne(t, session, doc, map[string]any{
		"author": "eric", "text": "human decision", "anchor": "It is slow.",
	})

	errText := callToolExpectError(t, session, "comments_reply", map[string]any{
		"filepath": doc,
		"replies": []any{
			map[string]any{"thread_id": notesThread, "author": "claude", "text": "tidied"},
			map[string]any{"thread_id": problemThread, "author": "claude", "text": "decided", "resolve": true},
		},
	})
	if !strings.Contains(errText, "human-decision zone") {
		t.Errorf("expected human-zone refusal, got: %s", errText)
	}

	for _, raw := range listThreads(t, session, doc) {
		c := raw.(map[string]any)
		if c["reply_count"] != float64(0) || c["resolved"] == true {
			t.Errorf("refused batch must apply nothing, but %v has reply_count %v resolved %v",
				c["id"], c["reply_count"], c["resolved"])
		}
	}
}

// signOffAfter records a review the way every human writer does (TUI, web and
// `comments signoff` all end in AddReviewRecord + SaveToSidecar), after delay.
// The delay matters: Watch's first pass only snapshots, so a signoff written
// before it is the baseline, not an event.
func signOffAfter(doc string, delay time.Duration, author, decision, note string) <-chan error {
	done := make(chan error, 1)
	go func() {
		time.Sleep(delay)
		loaded, _, err := comment.LoadFromSidecar(doc)
		if err != nil {
			done <- err
			return
		}
		comment.AddReviewRecord(loaded, author, decision, note, false)
		done <- comment.SaveToSidecar(doc, loaded)
	}()
	return done
}

func shortenWatchPoll(t *testing.T) {
	t.Helper()
	previous := watchPollInterval
	watchPollInterval = 20 * time.Millisecond
	t.Cleanup(func() { watchPollInterval = previous })
}

// TestWatchReturnsSignoff is the agent's wait: the matched event must carry the
// decision AND the reviewer's note, because the note is often the whole
// instruction ("approved, but rename X") and the agent makes no second read.
func TestWatchReturnsSignoff(t *testing.T) {
	shortenWatchPoll(t)
	session := startTestSession(t)
	doc := writeFixture(t)

	// Watch only follows documents that already have a sidecar
	addOne(t, session, doc, map[string]any{"author": "eric", "text": "tighten this", "line": 5})

	written := signOffAfter(doc, 300*time.Millisecond, "eric", "", "looks good")
	watched := callTool(t, session, "comments_watch", map[string]any{"filepath": doc, "timeout_seconds": 10})
	if err := <-written; err != nil {
		t.Fatalf("recording the signoff failed: %v", err)
	}

	if watched["status"] != "matched" {
		t.Fatalf("expected matched, got %v", watched)
	}
	events := watched["events"].([]any)
	if len(events) == 0 {
		t.Fatalf("matched wait returned no events: %v", watched)
	}
	// until defaults to signoff, and the event that matched is always last
	signoff := events[len(events)-1].(map[string]any)
	if signoff["event"] != "signoff" || signoff["author"] != "eric" || signoff["note"] != "looks good" {
		t.Errorf("signoff event = %v", signoff)
	}
	// Decision derived from the gate: only a non-blocking comment is open
	if signoff["decision"] != comment.DecisionApproved {
		t.Errorf("expected derived decision approved, got %v", signoff["decision"])
	}
}

// A reply-pass ("commented") is not approval; the wait must hand the decision
// through verbatim rather than collapsing it to matched/approved.
func TestWatchCarriesNonApprovalDecision(t *testing.T) {
	shortenWatchPoll(t)
	session := startTestSession(t)
	doc := writeFixture(t)
	addOne(t, session, doc, map[string]any{"author": "eric", "text": "must fix", "line": 5, "blocking": true})

	written := signOffAfter(doc, 300*time.Millisecond, "eric", "", "see the blocking thread")
	watched := callTool(t, session, "comments_watch", map[string]any{
		"filepath": doc, "until": "signoff", "timeout_seconds": 10,
	})
	if err := <-written; err != nil {
		t.Fatalf("recording the signoff failed: %v", err)
	}
	events := watched["events"].([]any)
	if watched["status"] != "matched" || len(events) == 0 {
		t.Fatalf("expected a matched signoff, got %v", watched)
	}
	signoff := events[len(events)-1].(map[string]any)
	if signoff["decision"] != comment.DecisionChangesRequested || signoff["note"] != "see the blocking thread" {
		t.Errorf("signoff event = %v", signoff)
	}
}

// An MCP client has a per-call ceiling, so a quiet wait must come back on its
// own as a normal result (not an error) for the agent to call again.
func TestWatchTimesOut(t *testing.T) {
	shortenWatchPoll(t)
	session := startTestSession(t)
	doc := writeFixture(t)
	addOne(t, session, doc, map[string]any{"author": "eric", "text": "note", "line": 5})

	watched := callTool(t, session, "comments_watch", map[string]any{"filepath": doc, "timeout_seconds": 1})
	if watched["status"] != "timeout" {
		t.Errorf("expected timeout, got %v", watched)
	}
	if events, ok := watched["events"].([]any); !ok || len(events) != 0 {
		t.Errorf("a quiet wait should return an empty events array, got %v", watched["events"])
	}
}

func TestInboxTool(t *testing.T) {
	session := startTestSession(t)
	doc := writeFixture(t)

	// Thread 1: blocking, no replies — always in the inbox
	addOne(t, session, doc, map[string]any{
		"author": "eric", "text": "must fix", "line": 5, "blocking": true,
	})
	// Thread 2: non-blocking with a reply — in the inbox via new_reply
	questionID := addOne(t, session, doc, map[string]any{
		"author": "eric", "text": "question", "line": 9,
	})
	// Thread 3: non-blocking, no replies — listed last, as plain unresolved. The
	// inbox is the agent's only read, so a quiet comment must not be invisible.
	addOne(t, session, doc, map[string]any{
		"author": "eric", "text": "just a note", "line": 9,
	})
	callTool(t, session, "comments_reply", map[string]any{
		"filepath": doc,
		"replies":  []any{map[string]any{"thread_id": questionID, "author": "claude", "text": "answered inline"}},
	})

	inbox := callTool(t, session, "comments_inbox", map[string]any{"filepath": doc})
	if inbox["count"] != float64(3) {
		t.Fatalf("expected 3 inbox items, got %v", inbox)
	}
	if inbox["decision"] != "changes_requested" {
		t.Errorf("inbox must carry the gate decision, got %v", inbox["decision"])
	}
	items := inbox["items"].([]any)
	if first := items[0].(map[string]any)["thread"].(map[string]any)["text"]; first != "must fix" {
		t.Errorf("blocking threads sort first, got %v", first)
	}
	if last := items[2].(map[string]any); last["reasons"].([]any)[0] != "unresolved" {
		t.Errorf("a quiet non-blocking thread lists as unresolved, got %v", last["reasons"])
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

	// since flags news; it never hides a thread. With since in the future the
	// reply is no longer news, so the thread drops from new_reply to unresolved.
	future := time.Now().Add(time.Hour).Format(time.RFC3339)
	inbox = callTool(t, session, "comments_inbox", map[string]any{"filepath": doc, "since": future})
	if inbox["count"] != float64(3) {
		t.Fatalf("since must not hide threads, got %v", inbox)
	}
	for _, raw := range inbox["items"].([]any) {
		item := raw.(map[string]any)
		want := "unresolved"
		if item["thread"].(map[string]any)["text"] == "must fix" {
			want = "blocking"
		}
		if got := item["reasons"].([]any)[0]; got != want {
			t.Errorf("with a future since, %v should be %s, got %v", item["thread"].(map[string]any)["text"], want, got)
		}
	}

	// resolved threads leave the inbox
	callTool(t, session, "comments_reply", map[string]any{
		"filepath": doc,
		"replies":  []any{map[string]any{"thread_id": questionID, "author": "claude", "resolve": true}},
	})
	inbox = callTool(t, session, "comments_inbox", map[string]any{"filepath": doc})
	if inbox["count"] != float64(2) {
		t.Errorf("expected 2 items after resolving the replied thread, got %v", inbox)
	}
}

// A malformed since must fail loudly: silently treating it as "no since" would
// flag every old reply as news and send the agent back over settled threads.
func TestInboxRejectsBadSince(t *testing.T) {
	session := startTestSession(t)
	doc := writeFixture(t)
	addOne(t, session, doc, map[string]any{"author": "eric", "text": "note", "line": 5})
	errText := callToolExpectError(t, session, "comments_inbox", map[string]any{
		"filepath": doc, "since": "not-a-timestamp",
	})
	if !strings.Contains(errText, "RFC3339") {
		t.Errorf("expected RFC3339 error, got: %s", errText)
	}
}

func TestInboxOnDirectory(t *testing.T) {
	session := startTestSession(t)
	doc := writeFixture(t)
	dir := filepath.Dir(doc)

	addOne(t, session, doc, map[string]any{
		"author": "eric", "text": "must fix", "line": 5, "blocking": true,
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

	id := addOne(t, session, doc, map[string]any{
		"author": "eric", "text": "note", "line": 5,
	})

	moved := callTool(t, session, "comments_reanchor", map[string]any{
		"filepath": doc, "moves": []any{map[string]any{"comment_id": id, "line": 9}},
	})
	result := moved["results"].([]any)[0].(map[string]any)
	if result["moved"] != true || result["line"] != float64(9) {
		t.Errorf("reanchor failed: %v", result)
	}
}
