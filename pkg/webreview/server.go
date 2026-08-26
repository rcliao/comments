// Package webreview exposes the comments review model as a local, browser-based
// human review surface. It deliberately delegates all document mutations to
// pkg/comment so the TUI, CLI, MCP server, and browser share one sidecar format.
package webreview

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/rcliao/comments/pkg/comment"
	mdstructure "github.com/rcliao/comments/pkg/markdown"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	mdtext "github.com/yuin/goldmark/text"
)

//go:embed static/*
var staticFiles embed.FS

const sessionCookie = "comments_review_token"

type documentTarget struct {
	ID   string
	Name string
	Path string
}

// Server is an http.Handler for one file or one directory review workspace.
type Server struct {
	target string
	author string
	token  string
	docs   []documentTarget

	mu           sync.RWMutex
	docLocks     map[string]*sync.Mutex
	allowedHosts map[string]bool
	static       http.Handler
	markdown     goldmark.Markdown
}

// New creates a review server. A directory includes Markdown documents that
// already have sidecars; an explicit Markdown file is always included.
func New(target, author string) (*Server, error) {
	abs, err := filepath.Abs(target)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	resolvedTarget, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("resolve review target: %w", err)
	}

	paths, err := comment.FindGateTargets(abs)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no Markdown documents with comment sidecars found in %s", abs)
	}
	root := filepath.Dir(resolvedTarget)
	if info.IsDir() {
		root = resolvedTarget
	}

	docs := make([]documentTarget, 0, len(paths))
	for _, path := range paths {
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			return nil, fmt.Errorf("resolve %s: %w", path, err)
		}
		if !withinRoot(root, resolved) {
			return nil, fmt.Errorf("document escapes review root: %s", path)
		}
		rel, err := filepath.Rel(root, resolved)
		if err != nil {
			return nil, err
		}
		docs = append(docs, documentTarget{ID: filepath.ToSlash(rel), Name: filepath.Base(resolved), Path: resolved})
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].ID < docs[j].ID })

	author = strings.TrimSpace(author)
	if author == "" {
		author = "Reviewer"
	}
	if utf8.RuneCountInString(author) > 80 {
		return nil, errors.New("author must be 80 characters or fewer")
	}
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("generate session token: %w", err)
	}
	assets, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return nil, err
	}

	return &Server{
		target:       resolvedTarget,
		author:       author,
		token:        hex.EncodeToString(tokenBytes),
		docs:         docs,
		docLocks:     map[string]*sync.Mutex{},
		allowedHosts: map[string]bool{},
		static:       http.FileServer(http.FS(assets)),
		markdown: goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		),
	}, nil
}

func withinRoot(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// Token returns the unguessable bootstrap token. It is intended to appear only
// in the initial local URL, which exchanges it for an HttpOnly cookie.
func (s *Server) Token() string { return s.token }

// ConfigureListener records the exact Host values accepted by the handler.
// Call it before serving to prevent DNS rebinding against the loopback server.
func (s *Server) ConfigureListener(listener net.Listener) {
	host, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		return
	}
	hosts := []string{net.JoinHostPort(host, port)}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		hosts = append(hosts, "localhost:"+port, "127.0.0.1:"+port, "[::1]:"+port)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, h := range hosts {
		s.allowedHosts[h] = true
	}
}

// AllowHost adds a Host accepted by the handler. It is primarily useful to
// mount the handler under httptest without weakening production checks.
func (s *Server) AllowHost(host string) {
	s.mu.Lock()
	s.allowedHosts[host] = true
	s.mu.Unlock()
}

// BootstrapURL is the local URL the user opens once to establish the session.
func (s *Server) BootstrapURL(listener net.Listener) string {
	_, port, _ := net.SplitHostPort(listener.Addr().String())
	return fmt.Sprintf("http://127.0.0.1:%s/?token=%s", port, s.token)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setSecurityHeaders(w)
	if !s.hostAllowed(r.Host) {
		http.Error(w, "unrecognized host", http.StatusForbidden)
		return
	}

	if token := r.URL.Query().Get("token"); token != "" {
		if !secureEqual(token, s.token) {
			http.Error(w, "invalid review token", http.StatusUnauthorized)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: s.token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
		destination := r.URL.Path
		if destination == "" {
			destination = "/"
		}
		http.Redirect(w, r, destination, http.StatusSeeOther)
		return
	}
	if !s.authenticated(r) {
		http.Error(w, "open the tokenized review URL printed by comments serve", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead && !sameOrigin(r) {
		http.Error(w, "cross-origin mutation rejected", http.StatusForbidden)
		return
	}

	switch r.URL.Path {
	case "/api/state":
		s.handleState(w, r)
	case "/api/action":
		s.handleAction(w, r)
	case "/api/events":
		s.handleEvents(w, r)
	default:
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		s.static.ServeHTTP(w, r)
	}
}

func setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; base-uri 'none'; frame-ancestors 'self'; form-action 'self'")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("Cache-Control", "no-store")
}

func (s *Server) hostAllowed(host string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.allowedHosts[host]
}

func secureEqual(a, b string) bool {
	return len(a) == len(b) && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func (s *Server) authenticated(r *http.Request) bool {
	cookie, err := r.Cookie(sessionCookie)
	return err == nil && secureEqual(cookie.Value, s.token)
}

func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	return origin == "" || origin == "http://"+r.Host || origin == "https://"+r.Host
}

func (s *Server) document(id string) (documentTarget, error) {
	if id == "" && len(s.docs) > 0 {
		return s.docs[0], nil
	}
	for _, doc := range s.docs {
		if doc.ID == id {
			return doc, nil
		}
	}
	return documentTarget{}, errors.New("unknown document")
}

type gateView struct {
	Decision           string `json:"decision"`
	Blocking           int    `json:"blocking"`
	NonBlocking        int    `json:"non_blocking"`
	PendingSuggestions int    `json:"pending_suggestions"`
}

type sectionView struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Path      string `json:"path"`
	Level     int    `json:"level"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

type docSummary struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Decision string `json:"decision"`
	Open     int    `json:"open"`
	Blocking int    `json:"blocking"`
}

type stateView struct {
	DocID        string               `json:"doc_id"`
	Name         string               `json:"name"`
	Author       string               `json:"author"`
	Revision     string               `json:"revision"`
	RenderedHTML string               `json:"rendered_html"`
	Lines        []string             `json:"lines"`
	Sections     []sectionView        `json:"sections"`
	Document     comment.DocumentView `json:"document"`
	Gate         gateView             `json:"gate"`
	Documents    []docSummary         `json:"documents"`
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}
	target, err := s.document(r.URL.Query().Get("doc"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	state, err := s.buildState(target)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (s *Server) buildState(target documentTarget) (stateView, error) {
	doc, _, err := comment.LoadFromSidecar(target.Path)
	if err != nil {
		return stateView{}, err
	}
	// Keep frontmatter's newlines for stable line anchors, but do not show YAML
	// metadata as document prose in the human review surface.
	rendered, err := renderMarkdownBlocks(s.markdown, []byte(mdstructure.MaskFrontmatter(doc.Content)))
	if err != nil {
		return stateView{}, fmt.Errorf("render Markdown: %w", err)
	}
	structure := mdstructure.ParseDocument(doc.Content)
	sections := make([]sectionView, 0, len(structure.SectionsByID))
	for _, section := range structure.SectionsByID {
		sections = append(sections, sectionView{ID: section.ID, Title: section.Title, Path: section.GetFullPath(structure.SectionsByID), Level: section.Level, StartLine: section.StartLine, EndLine: section.EndLine})
	}
	sort.Slice(sections, func(i, j int) bool { return sections[i].StartLine < sections[j].StartLine })
	gate := comment.EvaluateGate(doc, false)
	summaries := make([]docSummary, 0, len(s.docs))
	for _, item := range s.docs {
		other, _, loadErr := comment.LoadFromSidecar(item.Path)
		if loadErr != nil {
			continue
		}
		g := comment.EvaluateGate(other, false)
		summaries = append(summaries, docSummary{ID: item.ID, Name: item.Name, Decision: g.Decision, Open: len(g.Blocking) + len(g.NonBlocking) + len(g.PendingSuggestions), Blocking: len(g.Blocking)})
	}
	rev, err := revision(target.Path)
	if err != nil {
		return stateView{}, err
	}
	return stateView{
		DocID: target.ID, Name: target.Name, Author: s.author, Revision: rev,
		RenderedHTML: rendered, Lines: strings.Split(doc.Content, "\n"), Sections: sections,
		Document:  comment.NewDocumentView(doc),
		Gate:      gateView{Decision: gate.Decision, Blocking: len(gate.Blocking), NonBlocking: len(gate.NonBlocking), PendingSuggestions: len(gate.PendingSuggestions)},
		Documents: summaries,
	}, nil
}

// renderMarkdownBlocks preserves Goldmark's safe GFM rendering while adding
// source ranges to top-level blocks. The browser uses those ranges to place
// comment pins beside the rendered passage that owns each anchored line.
func renderMarkdownBlocks(markdown goldmark.Markdown, source []byte) (string, error) {
	document := markdown.Parser().Parse(mdtext.NewReader(source))
	var rendered strings.Builder
	for node := document.FirstChild(); node != nil; node = node.NextSibling() {
		start, stop, ok := nodeSourceRange(node)
		if !ok {
			continue
		}
		var block bytes.Buffer
		if err := markdown.Renderer().Render(&block, source, node); err != nil {
			return "", err
		}
		startLine := sourceLineAt(source, start)
		endOffset := stop
		if endOffset > start && endOffset <= len(source) && source[endOffset-1] == '\n' {
			endOffset--
		}
		endLine := sourceLineAt(source, endOffset)
		_, _ = fmt.Fprintf(&rendered, `<div class="rendered-block" data-start-line="%d" data-end-line="%d">%s</div>`, startLine, endLine, block.String())
	}
	return rendered.String(), nil
}

func nodeSourceRange(root ast.Node) (int, int, bool) {
	start, stop := 0, 0
	found := false
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || node.Type() != ast.TypeBlock || node.Lines() == nil {
			return ast.WalkContinue, nil
		}
		for i := 0; i < node.Lines().Len(); i++ {
			segment := node.Lines().At(i)
			if !found || segment.Start < start {
				start = segment.Start
			}
			if !found || segment.Stop > stop {
				stop = segment.Stop
			}
			found = true
		}
		return ast.WalkContinue, nil
	})
	return start, stop, found
}

func sourceLineAt(source []byte, offset int) int {
	if offset < 0 {
		offset = 0
	}
	if offset > len(source) {
		offset = len(source)
	}
	return bytes.Count(source[:offset], []byte{'\n'}) + 1
}

type actionRequest struct {
	Action   string `json:"action"`
	Revision string `json:"revision"`
	Author   string `json:"author"`
	ThreadID string `json:"thread_id"`
	Text     string `json:"text"`
	Type     string `json:"type"`
	Priority string `json:"priority"`
	Blocking bool   `json:"blocking"`
	Line     int    `json:"line"`
	Decision string `json:"decision"`
	Note     string `json:"note"`
}

func (s *Server) handleAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	target, err := s.document(r.URL.Query().Get("doc"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	var request actionRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid action: "+err.Error())
		return
	}
	lock := s.lockFor(target.Path)
	lock.Lock()
	defer lock.Unlock()

	current, err := revision(target.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if request.Revision == "" || request.Revision != current {
		state, _ := s.buildState(target)
		writeJSON(w, http.StatusConflict, map[string]any{"error": "document changed; review the refreshed state", "state": state})
		return
	}
	doc, _, err := comment.LoadFromSidecar(target.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	request.Author = strings.TrimSpace(request.Author)
	if request.Author == "" {
		request.Author = s.author
	}
	if utf8.RuneCountInString(request.Author) > 80 {
		writeError(w, http.StatusBadRequest, "author must be 80 characters or fewer")
		return
	}
	if err := applyAction(target.Path, doc, request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := comment.SaveToSidecar(target.Path, doc); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Match the TUI and signoff command: verdicts replace the reviewer's
	// changed-since baseline, while a reply-only pass keeps it accumulating.
	// The review record has already landed, so this local convenience write is
	// deliberately best-effort.
	if request.Action == "verdict" && comment.BaselineUpdatesOn(request.Decision) {
		_ = comment.SaveReviewBaseline(target.Path, request.Author, doc.Content)
	}
	state, err := s.buildState(target)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func applyAction(path string, doc *comment.DocumentWithComments, request actionRequest) error {
	switch request.Action {
	case "add":
		text := strings.TrimSpace(request.Text)
		if text == "" {
			return errors.New("comment text is required")
		}
		// Treat a leading marker as an input shorthand, not just decoration.
		// The browser normally selects the type live, but inferring here keeps
		// stale tabs and non-JavaScript clients from storing "[S] ..." as a
		// General comment when they omit the explicit type field.
		if request.Type == "" {
			if inferred, ok := comment.LeadingType(text); ok {
				request.Type = inferred
			}
		}
		lineCount := len(strings.Split(doc.Content, "\n"))
		if request.Line < 1 || request.Line > lineCount {
			return fmt.Errorf("line must be between 1 and %d", lineCount)
		}
		if request.Priority == "" {
			request.Priority = comment.PriorityMedium
		}
		if request.Priority != comment.PriorityLow && request.Priority != comment.PriorityMedium && request.Priority != comment.PriorityHigh {
			return errors.New("priority must be low, medium, or high")
		}
		if request.Type != "" && request.Type != "Q" && request.Type != "S" && request.Type != "B" && request.Type != "T" && request.Type != "E" {
			return errors.New("type must be Q, S, B, T, or E")
		}
		created := comment.NewCommentWithType(request.Author, request.Line, comment.PrefixType(text, request.Type), request.Type)
		created.Priority = request.Priority
		created.Status = comment.StatusActive
		created.Blocking = request.Blocking
		comment.UpdateCommentSection(created, doc.Content)
		doc.Threads = append(doc.Threads, created)
	case "reply":
		if strings.TrimSpace(request.Text) == "" {
			return errors.New("reply text is required")
		}
		return comment.AddReplyToThread(doc.Threads, request.ThreadID, request.Author, strings.TrimSpace(request.Text))
	case "resolve":
		if err := comment.GuardZoneResolve(doc, path, request.ThreadID, comment.ActorHuman); err != nil {
			return err
		}
		return comment.ResolveThread(doc.Threads, request.ThreadID)
	case "reopen":
		return comment.UnresolveThread(doc.Threads, request.ThreadID)
	case "accept":
		changed, err := comment.ApplyAndAcceptSuggestion(doc, request.ThreadID)
		if err != nil {
			return err
		}
		if changed {
			if err := comment.SaveDocumentContent(path, doc); err != nil {
				return err
			}
		}
	case "reject":
		return comment.RejectSuggestion(doc.Threads, request.ThreadID)
	case "verdict":
		if request.Decision != comment.DecisionApproved && request.Decision != comment.DecisionChangesRequested && request.Decision != comment.DecisionCommented {
			return errors.New("decision must be approved, changes_requested, or commented")
		}
		comment.AddReviewRecord(doc, request.Author, request.Decision, strings.TrimSpace(request.Note), false)
	default:
		return fmt.Errorf("unknown action %q", request.Action)
	}
	return nil
}

func (s *Server) lockFor(path string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.docLocks[path] == nil {
		s.docLocks[path] = &sync.Mutex{}
	}
	return s.docLocks[path]
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}
	target, err := s.document(r.URL.Query().Get("doc"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unavailable")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Connection", "keep-alive")
	last, _ := revision(target.Path)
	if _, err := fmt.Fprint(w, "event: ready\ndata: {}\n\n"); err != nil {
		return
	}
	flusher.Flush()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			next, readErr := revision(target.Path)
			if readErr == nil && next != last {
				last = next
				if _, err := fmt.Fprintf(w, "event: refresh\ndata: {\"revision\":%q}\n\n", next); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	}
}

func revision(path string) (string, error) {
	hash := sha256.New()
	markdown, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	hash.Write([]byte("markdown\x00"))
	hash.Write(markdown)
	hash.Write([]byte("\x00sidecar\x00"))
	sidecar, err := os.ReadFile(comment.GetSidecarPath(path))
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	hash.Write(sidecar)
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
