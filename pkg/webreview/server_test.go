package webreview

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rcliao/comments/pkg/comment"
)

const testHost = "review.local"

func newTestServer(t *testing.T, markdown string) (*Server, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "review.md")
	if err := os.WriteFile(path, []byte(markdown), 0644); err != nil {
		t.Fatal(err)
	}
	server, err := New(path, "Rae")
	if err != nil {
		t.Fatal(err)
	}
	server.AllowHost(testHost)
	return server, path
}

func authorizedRequest(t *testing.T, server *Server, method, target string, body any) *http.Request {
	t.Helper()
	var input bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&input).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, target, &input)
	req.Host = testHost
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: server.Token()})
	if method != http.MethodGet {
		req.Header.Set("Origin", "http://"+testHost)
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func readState(t *testing.T, server *Server) stateView {
	t.Helper()
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, authorizedRequest(t, server, http.MethodGet, "/api/state", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("state status = %d: %s", recorder.Code, recorder.Body.String())
	}
	var state stateView
	if err := json.Unmarshal(recorder.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	return state
}

func TestBootstrapExchangesTokenForSecureSession(t *testing.T) {
	server, _ := newTestServer(t, "# Review\n")

	unauthorized := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	req.Host = testHost
	server.ServeHTTP(unauthorized, req)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d", unauthorized.Code)
	}

	bootstrap := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/?token="+server.Token(), nil)
	req.Host = testHost
	server.ServeHTTP(bootstrap, req)
	if bootstrap.Code != http.StatusSeeOther {
		t.Fatalf("bootstrap status = %d: %s", bootstrap.Code, bootstrap.Body.String())
	}
	cookies := bootstrap.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatalf("unexpected session cookie: %#v", cookies)
	}
	if got := bootstrap.Header().Get("Content-Security-Policy"); got == "" {
		t.Fatal("missing Content-Security-Policy")
	}
}

func TestStateRendersMarkdownWithoutUnsafeHTML(t *testing.T) {
	server, _ := newTestServer(t, "# Review\n\nHello **world**.\n\n<script>alert('no')</script>\n")
	state := readState(t, server)
	if !strings.Contains(state.RenderedHTML, "<strong>world</strong>") {
		t.Fatalf("Markdown not rendered: %s", state.RenderedHTML)
	}
	if strings.Contains(state.RenderedHTML, "<script>") {
		t.Fatalf("unsafe HTML passed through: %s", state.RenderedHTML)
	}
	for _, sourceRange := range []string{`data-start-line="1" data-end-line="1"`, `data-start-line="3" data-end-line="3"`} {
		if !strings.Contains(state.RenderedHTML, sourceRange) {
			t.Fatalf("rendered block is missing source range %s: %s", sourceRange, state.RenderedHTML)
		}
	}
	if len(state.Sections) != 1 || state.Sections[0].StartLine != 1 {
		t.Fatalf("unexpected sections: %#v", state.Sections)
	}
}

func TestStateHidesFrontmatterWithoutShiftingSourceLines(t *testing.T) {
	server, _ := newTestServer(t, "---\ntype: Design\ncomments:\n  template: design-doc\n---\n\n# Review\n\nBody.\n")
	state := readState(t, server)
	if strings.Contains(state.RenderedHTML, "template: design-doc") || strings.Contains(state.RenderedHTML, "type: Design") {
		t.Fatalf("frontmatter rendered as prose: %s", state.RenderedHTML)
	}
	if !strings.Contains(state.RenderedHTML, `data-start-line="7" data-end-line="7"`) {
		t.Fatalf("heading line shifted after masking frontmatter: %s", state.RenderedHTML)
	}
	if len(state.Sections) != 1 || state.Sections[0].StartLine != 7 {
		t.Fatalf("section lines shifted: %#v", state.Sections)
	}
}

func TestReviewUIIncludesStickyRailAndDocumentTargetSelection(t *testing.T) {
	server, _ := newTestServer(t, "# Review\n\nA line.\n")

	readAsset := func(path string) string {
		t.Helper()
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, authorizedRequest(t, server, http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d: %s", path, recorder.Code, recorder.Body.String())
		}
		return recorder.Body.String()
	}

	css := readAsset("/app.css")
	for _, want := range []string{"position: sticky", "height: calc(100vh - 54px)", ".thread-list", "overflow-y: auto"} {
		if !strings.Contains(css, want) {
			t.Fatalf("review CSS is missing %q", want)
		}
	}

	js := readAsset("/app.js")
	for _, want := range []string{"bindRenderedCommentTargets", "beginTargetSelection", "commentTargetLabel", "Click a passage or section", "selectRelativeThread", "openVerdictKeyboardMode", "actOnSelectedThread", "shortcut-dialog", "beginLineSelection", "moveLineSelection", "keyboard-line-selected", "cycleCommentType", "cycleCommentPriority", "syncCommentTypeFromText", "requestSubmit", "isCommentSubmitShortcut", "NumpadEnter", "keyup", "focusSelectedThread", "closeReplyComposer", "keyboard-focused", "Esc cancels"} {
		if !strings.Contains(js, want) {
			t.Fatalf("review JavaScript is missing %q", want)
		}
	}
	if strings.Contains(js, "rendered-add-comment") {
		t.Fatal("review JavaScript still renders a per-passage add button")
	}

	html := readAsset("/")
	for _, want := range []string{"shortcut-button", "Review shortcuts", "Next / previous thread", "Open verdict mode", "keyboard-mode-hint", "Enter line-select mode", "previous/next thread", "Ctrl+S", "Ctrl+T", "Type prefixes", "comment-compose-status"} {
		if !strings.Contains(html, want) {
			t.Fatalf("review HTML is missing %q", want)
		}
	}
}

func TestActionFlowPersistsThreadReplyAndResolution(t *testing.T) {
	server, path := newTestServer(t, "# Review\n\nA line to discuss.\n")
	state := readState(t, server)

	state = postAction(t, server, http.StatusOK, actionRequest{
		Action: "add", Revision: state.Revision, Author: "Rae", Line: 3, Text: "Clarify this", Type: "Q", Priority: "high", Blocking: true,
	})
	if len(state.Document.Threads) != 1 {
		t.Fatalf("threads after add = %d", len(state.Document.Threads))
	}
	threadID := state.Document.Threads[0].ID
	state = postAction(t, server, http.StatusOK, actionRequest{Action: "reply", Revision: state.Revision, Author: "Sam", ThreadID: threadID, Text: "Done"})
	if len(state.Document.Threads[0].Replies) != 1 {
		t.Fatal("reply was not returned")
	}
	state = postAction(t, server, http.StatusOK, actionRequest{Action: "resolve", Revision: state.Revision, ThreadID: threadID})
	if !state.Document.Threads[0].Resolved || state.Gate.Blocking != 0 {
		t.Fatalf("thread not resolved: %#v gate=%#v", state.Document.Threads[0], state.Gate)
	}

	doc, _, err := comment.LoadFromSidecar(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Threads) != 1 || len(doc.Threads[0].Replies) != 1 || !doc.Threads[0].Resolved {
		t.Fatalf("sidecar did not persist flow: %#v", doc.Threads)
	}
}

func TestActionNormalizesAndLimitsAuthor(t *testing.T) {
	server, _ := newTestServer(t, "# Review\n\nA line.\n")
	state := readState(t, server)
	state = postAction(t, server, http.StatusOK, actionRequest{Action: "add", Revision: state.Revision, Author: "  Rae Reviewer  ", Line: 3, Text: "Comment"})
	if got := state.Document.Threads[0].Author; got != "Rae Reviewer" {
		t.Fatalf("normalized author = %q", got)
	}

	recorder := postRaw(t, server, actionRequest{Action: "add", Revision: state.Revision, Author: strings.Repeat("a", 81), Line: 3, Text: "Too long"})
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "80 characters") {
		t.Fatalf("long author status = %d: %s", recorder.Code, recorder.Body.String())
	}
	if got := len(readState(t, server).Document.Threads); got != 1 {
		t.Fatalf("threads after rejected author = %d", got)
	}
}

func TestActionInfersTypeFromLeadingMarker(t *testing.T) {
	server, _ := newTestServer(t, "# Review\n\nA line.\n")
	state := readState(t, server)
	state = postAction(t, server, http.StatusOK, actionRequest{
		Action: "add", Revision: state.Revision, Author: "Rae", Line: 3, Text: "[S] Tighten this wording",
	})
	if got := state.Document.Threads[0].Type; got != comment.TypeSuggestion {
		t.Fatalf("inferred type = %q, want %q", got, comment.TypeSuggestion)
	}
	if got := state.Document.Threads[0].Text; got != "💡 [S] Tighten this wording" {
		t.Fatalf("typed text = %q", got)
	}
}

func TestStaleActionReturnsConflictWithoutOverwrite(t *testing.T) {
	server, _ := newTestServer(t, "# Review\n\nText.\n")
	state := readState(t, server)
	postAction(t, server, http.StatusOK, actionRequest{Action: "add", Revision: state.Revision, Line: 3, Text: "First"})

	recorder := postRaw(t, server, actionRequest{Action: "add", Revision: state.Revision, Line: 3, Text: "Stale"})
	if recorder.Code != http.StatusConflict {
		t.Fatalf("stale action status = %d: %s", recorder.Code, recorder.Body.String())
	}
	refreshed := readState(t, server)
	if len(refreshed.Document.Threads) != 1 || strings.Contains(refreshed.Document.Threads[0].Text, "Stale") {
		t.Fatalf("stale write was not rejected: %#v", refreshed.Document.Threads)
	}
}

func TestVerdictRecordsReviewAndBaseline(t *testing.T) {
	server, path := newTestServer(t, "# Review\n\nReady.\n")
	state := readState(t, server)
	state = postAction(t, server, http.StatusOK, actionRequest{Action: "verdict", Revision: state.Revision, Author: "Rae", Decision: comment.DecisionApproved, Note: "Ship it"})
	if len(state.Document.Reviews) != 1 || state.Document.Reviews[0].Note != "Ship it" {
		t.Fatalf("review not recorded: %#v", state.Document.Reviews)
	}
	baseline, ok := comment.LoadReviewBaseline(path, "Rae")
	if !ok || baseline != "# Review\n\nReady.\n" {
		t.Fatalf("baseline = %q, ok=%v", baseline, ok)
	}
}

func TestRejectsCrossOriginMutationAndUnknownDocument(t *testing.T) {
	server, _ := newTestServer(t, "# Review\n")
	state := readState(t, server)
	req := authorizedRequest(t, server, http.MethodPost, "/api/action", actionRequest{Action: "add", Revision: state.Revision, Line: 1, Text: "x"})
	req.Header.Set("Origin", "https://attacker.example")
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("cross-origin status = %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	server.ServeHTTP(recorder, authorizedRequest(t, server, http.MethodGet, "/api/state?doc=../secret.md", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("unknown document status = %d", recorder.Code)
	}
}

func postAction(t *testing.T, server *Server, want int, action actionRequest) stateView {
	t.Helper()
	recorder := postRaw(t, server, action)
	if recorder.Code != want {
		t.Fatalf("action status = %d, want %d: %s", recorder.Code, want, recorder.Body.String())
	}
	var state stateView
	if err := json.Unmarshal(recorder.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	return state
}

func postRaw(t *testing.T, server *Server, action actionRequest) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, authorizedRequest(t, server, http.MethodPost, "/api/action", action))
	return recorder
}
