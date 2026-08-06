package tui

// Reference peek (docs/design-reference-jump.md): `f` on a line with a
// citation opens a read-only excerpt of the target centered on the cited
// location; Enter hands off to $EDITOR at file:line; Esc returns to review.
// References are detected at load into m.refsByLine (rendering stays pure).

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/rcliao/comments/pkg/markdown"
)

// resolvedRef pairs a parsed reference with its on-disk resolution
type resolvedRef struct {
	markdown.Reference
	absPath  string // "" when unresolved
	resolved bool
}

// buildRefMap parses and resolves all references in the document. Called once
// at load; rendering and peek only read the result.
func buildRefMap(content, docPath string) map[int][]resolvedRef {
	docDir := filepath.Dir(docPath)
	if abs, err := filepath.Abs(docDir); err == nil {
		docDir = abs
	}
	byLine := map[int][]resolvedRef{}
	for _, ref := range markdown.ParseReferences(content) {
		abs, ok := markdown.ResolveReference(docDir, ref.Path)
		byLine[ref.LineNum] = append(byLine[ref.LineNum], resolvedRef{
			Reference: ref, absPath: abs, resolved: ok,
		})
	}
	return byLine
}

// openRefPeek enters peek mode for the references on the current cursor line.
// No references -> no-op.
func (m Model) openRefPeek() (tea.Model, tea.Cmd) {
	refs := m.refsByLine[m.selectedLine]
	if len(refs) == 0 {
		return m, nil
	}
	m.refPeekList = refs
	m.refPeekIdx = 0
	m.refPeekReturnMode = m.mode
	m.loadPeekTarget()
	m.mode = ModeRefPeek
	return m, nil
}

// loadPeekTarget reads the current peek target's content (the one file I/O of
// the peek flow, done on open/cycle so the view stays pure).
func (m *Model) loadPeekTarget() {
	m.refPeekContent = nil
	m.refPeekErr = ""
	m.refPeekTargetLine = 0

	ref := m.refPeekList[m.refPeekIdx]
	if !ref.resolved {
		m.refPeekErr = fmt.Sprintf("cannot resolve %s — no such file (searched from the document's directory upward)", ref.Path)
		return
	}
	data, err := os.ReadFile(ref.absPath)
	if err != nil {
		m.refPeekErr = fmt.Sprintf("cannot read %s: %v", ref.absPath, err)
		return
	}
	content := string(data)
	m.refPeekContent = strings.Split(content, "\n")

	switch {
	case ref.Line > 0:
		m.refPeekTargetLine = ref.Line
	case ref.Heading != "":
		m.refPeekTargetLine = markdown.HeadingLine(content, ref.Heading)
	}
}

// editorFinishedMsg reports the $EDITOR handoff ending (TUI resumes)
type editorFinishedMsg struct{ err error }

// editorCommand builds the $EDITOR invocation for a file:line target.
// The +N line prefix is understood by vi/vim/nvim/nano/emacs alike.
func editorCommand(absPath string, line int) *exec.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	args := []string{}
	if line > 0 {
		args = append(args, fmt.Sprintf("+%d", line))
	}
	args = append(args, absPath)
	return exec.Command(editor, args...)
}

// handleRefPeekKeys handles keys in reference-peek mode
func (m Model) handleRefPeekKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.mode = m.refPeekReturnMode
		m.refreshCursorView()
		return m, nil

	case "f", "tab":
		if len(m.refPeekList) > 1 {
			m.refPeekIdx = (m.refPeekIdx + 1) % len(m.refPeekList)
			m.loadPeekTarget()
		}
		return m, nil

	case "enter":
		ref := m.refPeekList[m.refPeekIdx]
		if !ref.resolved {
			return m, nil
		}
		line := m.refPeekTargetLine
		return m, tea.ExecProcess(editorCommand(ref.absPath, line), func(err error) tea.Msg {
			return editorFinishedMsg{err: err}
		})
	}
	return m, nil
}

// viewRefPeek renders the peek overlay: a read-only excerpt of the target
// centered on the cited location, or the resolution error.
func (m Model) viewRefPeek() string {
	ref := m.refPeekList[m.refPeekIdx]

	title := ref.Path
	if ref.Line > 0 {
		title = fmt.Sprintf("%s:%d", ref.Path, ref.Line)
	} else if ref.Heading != "" {
		title = fmt.Sprintf("%s#%s", ref.Path, ref.Heading)
	}
	if len(m.refPeekList) > 1 {
		title = fmt.Sprintf("%s  (%d/%d)", title, m.refPeekIdx+1, len(m.refPeekList))
	}

	var body strings.Builder
	body.WriteString(m.styles.title.Render("🔗 "+title) + "\n\n")

	if m.refPeekErr != "" {
		body.WriteString(m.refPeekErr + "\n")
	} else {
		excerptHeight := min(20, max(m.height-8, 3))
		lineWidth := max(min(m.width-14, 96), 20)

		center := m.refPeekTargetLine
		if center == 0 {
			center = 1 + excerptHeight/2 // no cited line: show the top
		}
		start := max(center-excerptHeight/2, 1)
		end := min(start+excerptHeight-1, len(m.refPeekContent))

		for i := start; i <= end; i++ {
			text := truncate(m.refPeekContent[i-1], lineWidth, "…")
			if i == m.refPeekTargetLine {
				fmt.Fprintf(&body, "%s\n", m.styles.cursor.Render(fmt.Sprintf("►%4d │ %s", i, text)))
			} else {
				fmt.Fprintf(&body, " %s │ %s\n", m.styles.lineNumber.Render(fmt.Sprintf("%4d", i)), text)
			}
		}
	}

	hint := "Enter: open in $EDITOR • Esc: close"
	if m.refPeekErr != "" {
		hint = "Esc: close"
	}
	if len(m.refPeekList) > 1 {
		hint = "f/Tab: next reference • " + hint
	}
	body.WriteString("\n" + m.styles.help.Render(hint))

	return m.styles.modalOverlay.Render(body.String())
}
