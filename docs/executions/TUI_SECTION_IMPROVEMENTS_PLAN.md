# TUI Section Support & Multi-line Improvements Plan

## Overview

This plan achieves full parity between CLI and TUI for section-based commenting, with enhanced multi-line suggestion support and clear visual distinction between section and line comments.

**User Requirements:**
- Full CLI-TUI parity for section operations
- Cursor-based section selection with ability to choose section vs line
- Multi-line suggestions via both range selection and section-based selection
- Visual distinction: section paths displayed + icons (📍 for sections, 💬 for lines)

## Current State Analysis

### What Works (CLI)
✓ Add comments by section path (`--section "Introduction > Overview"`)
✓ Filter/list comments by section (includes nested sub-sections)
✓ Add suggestions by section path
✓ Display section path in comment listings
✓ Helpful error messages showing available sections

### What Works (TUI)
✓ View documents with line-based comments
✓ Add comments to selected lines
✓ Add single-line suggestions
✓ Reply to threads, resolve/unresolve
✓ Accept/reject suggestions
✓ Markdown syntax highlighting

### Critical Gaps (TUI)
✗ NO section-based commenting capability
✗ Section paths stored but NEVER displayed
✗ No section filtering or navigation
✗ No multi-line suggestion support (locked to single-line)
✗ No visual distinction between section/line comments

## Implementation Plan

### Phase 1: Foundation - Section Metadata Integration

**Goal**: Parse and expose section information in TUI Model

**Files to modify:**
- `pkg/tui/model.go`

**Changes:**

1. **Add field to Model struct** (around line 30-50):
```go
type Model struct {
    // ... existing fields ...

    // Section support
    documentSections *markdown.Document  // Parsed section hierarchy
}
```

2. **Parse sections on document load** in `loadDocumentFromPath()` (around line 600-650):
```go
func (m *Model) loadDocumentFromPath(path string) error {
    // ... existing load logic ...

    // Parse sections
    m.documentSections = markdown.ParseDocument(m.doc.Content)

    // ... rest of function ...
}
```

3. **Add helper methods** (add near end of file):
```go
// getSectionAtLine returns the section containing the given line, or nil
func (m *Model) getSectionAtLine(lineNum int) *markdown.Section {
    if m.documentSections == nil {
        return nil
    }
    if section, exists := m.documentSections.SectionsByLine[lineNum]; exists {
        return section
    }
    return nil
}

// isHeadingLine returns true if the line is a markdown heading
func (m *Model) isHeadingLine(lineNum int) bool {
    section := m.getSectionAtLine(lineNum)
    if section == nil {
        return false
    }
    return section.StartLine == lineNum
}

// getSectionPath returns the full hierarchical path for a section
func (m *Model) getSectionPath(section *markdown.Section) string {
    if section == nil {
        return ""
    }

    path := section.Title
    current := section.Parent
    for current != nil {
        path = current.Title + " > " + path
        current = current.Parent
    }
    return path
}
```

**Testing:**
- Load a document with headings and verify `documentSections` is populated
- Test `isHeadingLine()` returns true for heading lines
- Test `getSectionPath()` returns correct hierarchical path

---

### Phase 2: Display Improvements

**Goal**: Show section context in all comment views

**Files to modify:**
- `pkg/tui/rendering.go`
- `pkg/tui/styles.go`

**Changes:**

1. **Update `renderComments()` in `rendering.go`** (lines 228-276):

Replace this block:
```go
commentText = fmt.Sprintf("Line %d • @%s%s\n    Type: %s | Thread ID: %s | Comment ID: %s\n    [%s] %s\n",
    c.Line,
    c.Author,
    timestampStr,
    displayType,
    c.ID,
    c.ID,
    displayType,
    c.Text,
)
```

With:
```go
// Build location string with section context
locationStr := fmt.Sprintf("Line %d", c.Line)
icon := "💬"
if c.SectionPath != "" {
    locationStr = fmt.Sprintf("%s (Line %d)", c.SectionPath, c.Line)
    icon = "📍"
}

commentText = fmt.Sprintf("%s %s • @%s%s\n    Type: %s | Thread ID: %s | Comment ID: %s\n    [%s] %s\n",
    icon,
    locationStr,
    c.Author,
    timestampStr,
    displayType,
    c.ID,
    c.ID,
    displayType,
    c.Text,
)
```

2. **Update `renderThread()` in `rendering.go`** (lines 278-350):

Update the header section (around line 295):
```go
// Build location string with section context
locationStr := fmt.Sprintf("Line %d", rootComment.Line)
icon := "💬"
if rootComment.SectionPath != "" {
    locationStr = fmt.Sprintf("%s (Line %d)", rootComment.SectionPath, rootComment.Line)
    icon = "📍"
}

header := m.styles.threadHeaderStyle.Render(
    fmt.Sprintf("%s %s • Thread %s", icon, locationStr, rootComment.ID),
)
```

3. **Add styles in `styles.go`** (add to Styles struct):
```go
type Styles struct {
    // ... existing styles ...

    sectionIconStyle  lipgloss.Style
    lineIconStyle     lipgloss.Style
}

// In NewStyles():
sectionIconStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("39")),  // Blue
lineIconStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("242")), // Gray
```

**Testing:**
- Load document with mixed section/line comments
- Verify section comments show: `📍 Introduction > Overview (Line 10)`
- Verify line-only comments show: `💬 Line 25`
- Check thread view header shows same format

---

### Phase 3: Section-Based Comment Input

**Goal**: Allow users to add comments to sections by navigating to headings

**Files to modify:**
- `pkg/tui/modes.go`
- `pkg/tui/model.go`

**Changes:**

1. **Add new mode in `modes.go`**:
```go
const (
    // ... existing modes ...
    ModeChooseTarget ViewMode = iota + 8  // Modal for choosing section vs line
)

func (m ViewMode) String() string {
    switch m {
    // ... existing cases ...
    case ModeChooseTarget:
        return "CHOOSE_TARGET"
    default:
        return "UNKNOWN"
    }
}
```

2. **Add state fields to Model struct** in `model.go`:
```go
type Model struct {
    // ... existing fields ...

    // Section input support
    targetIsSection bool  // True if user wants to comment on section, false for line only
}
```

3. **Update `handleLineSelectKeys()` in `model.go`** (around line 400):
```go
case "c":
    if m.isHeadingLine(m.selectedLine) {
        // On a heading - let user choose section vs line
        m.mode = ModeChooseTarget
        return m, nil
    }
    // Regular line - go directly to add comment
    m.mode = ModeAddComment
    m.commentInput.SetValue("")
    m.commentInput.Focus()
    return m, nil
```

4. **Add new handler `handleChooseTargetKeys()` in `model.go`**:
```go
func (m *Model) handleChooseTargetKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "s":
        // Section mode
        m.targetIsSection = true
        m.mode = ModeAddComment
        m.commentInput.SetValue("")
        m.commentInput.Focus()
        return m, nil

    case "l":
        // Line only mode
        m.targetIsSection = false
        m.mode = ModeAddComment
        m.commentInput.SetValue("")
        m.commentInput.Focus()
        return m, nil

    case "esc", "q":
        m.mode = ModeBrowse
        return m, nil
    }

    return m, nil
}
```

5. **Update `handleKeyPress()` switch** to route to new handler:
```go
case ModeChooseTarget:
    return m.handleChooseTargetKeys(msg)
```

6. **Update `handleAddCommentKeys()` to use section info** (around line 500):
```go
case "enter":
    if m.commentInput.Value() == "" {
        return m, nil
    }

    newComment := &comment.Comment{
        ID:        fmt.Sprintf("c%d", time.Now().Unix()),
        Author:    getAuthorName(),
        Timestamp: time.Now(),
        Text:      m.commentInput.Value(),
        Type:      m.selectedCommentType,
        Line:      m.selectedLine,
        Resolved:  false,
        Replies:   []*comment.Comment{},
    }

    // Add section metadata if targeting section
    if m.targetIsSection {
        section := m.getSectionAtLine(m.selectedLine)
        if section != nil {
            comment.UpdateCommentSection(newComment, m.documentSections)
        }
    }

    m.doc.Threads = append(m.doc.Threads, newComment)

    // Save and refresh
    // ... existing save logic ...
```

7. **Add view function `viewChooseTarget()` in `model.go`**:
```go
func (m *Model) viewChooseTarget() string {
    section := m.getSectionAtLine(m.selectedLine)
    sectionPath := m.getSectionPath(section)

    content := fmt.Sprintf("\n  Add comment to:\n\n")
    content += fmt.Sprintf("  [s] Section: %s (covers lines %d-%d)\n",
        sectionPath, section.StartLine, section.EndLine)
    content += fmt.Sprintf("  [l] Line %d only (heading line)\n\n", m.selectedLine)
    content += fmt.Sprintf("  [esc] Cancel\n")

    return m.styles.helpStyle.Render(content)
}
```

8. **Update `View()` switch** to include new mode:
```go
case ModeChooseTarget:
    return m.viewChooseTarget()
```

**Testing:**
- Navigate to heading and press 'c'
- Verify modal shows section vs line choice
- Choose section, add comment, verify SectionPath is set
- Choose line, add comment, verify SectionPath is empty
- Save and reload, verify metadata persists

---

### Phase 4: Multi-line Suggestion Support

**Goal**: Support range selection and section-based suggestions

**Files to modify:**
- `pkg/tui/modes.go`
- `pkg/tui/model.go`
- `pkg/tui/rendering.go`

**Changes:**

1. **Add new modes in `modes.go`**:
```go
const (
    // ... existing modes ...
    ModeSelectSuggestionType ViewMode = iota + 9  // Choose range vs section
    ModeSelectRange          ViewMode = iota + 10 // Visual range selection
)
```

2. **Add state fields to Model** in `model.go`:
```go
type Model struct {
    // ... existing fields ...

    // Multi-line suggestion support
    rangeStartLine int
    rangeEndLine   int
    rangeActive    bool
    suggestionIsSection bool
}
```

3. **Update 's' key handler in `handleLineSelectKeys()`**:
```go
case "s":
    if m.isHeadingLine(m.selectedLine) {
        // On heading - choose range vs section
        m.mode = ModeSelectSuggestionType
        return m, nil
    }
    // Regular line - start range selection
    m.rangeStartLine = m.selectedLine
    m.rangeEndLine = m.selectedLine
    m.rangeActive = true
    m.mode = ModeSelectRange
    return m, nil
```

4. **Add `handleSelectSuggestionTypeKeys()` handler**:
```go
func (m *Model) handleSelectSuggestionTypeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "r":
        // Range selection
        section := m.getSectionAtLine(m.selectedLine)
        m.rangeStartLine = section.StartLine
        m.rangeEndLine = section.EndLine
        m.rangeActive = true
        m.suggestionIsSection = false
        m.mode = ModeSelectRange
        return m, nil

    case "s":
        // Section-based suggestion
        section := m.getSectionAtLine(m.selectedLine)
        m.rangeStartLine = section.StartLine
        m.rangeEndLine = section.EndLine
        m.suggestionIsSection = true
        m.mode = ModeAddSuggestion
        m.commentInput.SetValue("")
        m.commentInput.Focus()
        return m, nil

    case "esc", "q":
        m.mode = ModeBrowse
        return m, nil
    }

    return m, nil
}
```

5. **Add `handleSelectRangeKeys()` handler**:
```go
func (m *Model) handleSelectRangeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "j", "down":
        // Extend range down
        lines := strings.Split(m.doc.Content, "\n")
        if m.rangeEndLine < len(lines) {
            m.rangeEndLine++
        }
        return m, nil

    case "k", "up":
        // Shrink range up
        if m.rangeEndLine > m.rangeStartLine {
            m.rangeEndLine--
        }
        return m, nil

    case "enter":
        // Confirm range
        m.mode = ModeAddSuggestion
        m.commentInput.SetValue("")
        m.commentInput.Focus()
        return m, nil

    case "esc", "q":
        // Cancel range selection
        m.rangeActive = false
        m.mode = ModeBrowse
        return m, nil
    }

    return m, nil
}
```

6. **Update `handleAddSuggestionKeys()` to support multi-line** (modify existing function):
```go
case "enter":
    if m.commentInput.Value() == "" {
        return m, nil
    }

    // Extract original text from range
    lines := strings.Split(m.doc.Content, "\n")
    originalLines := lines[m.rangeStartLine-1 : m.rangeEndLine]
    originalText := strings.Join(originalLines, "\n")

    newSuggestion := &comment.Comment{
        ID:           fmt.Sprintf("c%d", time.Now().Unix()),
        Author:       getAuthorName(),
        Timestamp:    time.Now(),
        Text:         m.commentInput.Value(),
        Line:         m.rangeStartLine,
        IsSuggestion: true,
        StartLine:    m.rangeStartLine,
        EndLine:      m.rangeEndLine,
        OriginalText: originalText,
        ProposedText: "", // TODO: Add proposed text input
        Resolved:     false,
        Replies:      []*comment.Comment{},
    }

    // Add section metadata if section-based
    if m.suggestionIsSection {
        comment.UpdateCommentSection(newSuggestion, m.documentSections)
    }

    m.doc.Threads = append(m.doc.Threads, newSuggestion)

    // Reset state
    m.rangeActive = false
    m.suggestionIsSection = false

    // Save and return to browse
    // ... existing save logic ...
```

7. **Update document rendering to show range selection** in `renderDocument()`:
```go
// In the line rendering loop
for i, line := range lines {
    lineNum := i + 1

    // Highlight range selection
    if m.rangeActive && lineNum >= m.rangeStartLine && lineNum <= m.rangeEndLine {
        line = m.styles.selectedLineStyle.Render(line)
    }

    // ... rest of rendering ...
}
```

8. **Add view functions**:
```go
func (m *Model) viewSelectSuggestionType() string {
    section := m.getSectionAtLine(m.selectedLine)
    sectionPath := m.getSectionPath(section)

    content := fmt.Sprintf("\n  Create suggestion for:\n\n")
    content += fmt.Sprintf("  [r] Line range (manual selection)\n")
    content += fmt.Sprintf("  [s] Section: %s (lines %d-%d)\n\n",
        sectionPath, section.StartLine, section.EndLine)
    content += fmt.Sprintf("  [esc] Cancel\n")

    return m.styles.helpStyle.Render(content)
}

func (m *Model) viewSelectRange() string {
    content := fmt.Sprintf("\n  Range Selection: Lines %d-%d\n\n",
        m.rangeStartLine, m.rangeEndLine)
    content += "  j/k: Adjust end line\n"
    content += "  enter: Confirm range\n"
    content += "  esc: Cancel\n"

    return m.styles.helpStyle.Render(content)
}
```

**Testing:**
- Navigate to heading, press 's', choose section suggestion
- Navigate to regular line, press 's', select range with j/k
- Verify range highlighting shows in document viewport
- Create suggestion, verify StartLine/EndLine are correct
- Verify section-based suggestions have SectionPath set

---

### Phase 5: Keybinding Documentation

**Goal**: Update help text and ensure consistency

**Files to modify:**
- `pkg/tui/model.go` (help text in various view functions)

**Updated Keybindings:**

**ModeBrowse:**
- `c` - Add comment (choose section/line if on heading)
- `s` - Add suggestion (choose type if on heading)
- `j/k` - Navigate comments
- `enter` - Expand thread
- `r` - Reply to thread
- `x` - Resolve thread
- `R` - Toggle show resolved
- `q` - Quit

**ModeLineSelect:**
- `j/k` - Move cursor
- `g/G` - Go to top/bottom
- `ctrl+d/u` - Page down/up
- `c` - Add comment (may show choice modal)
- `s` - Add suggestion (may show choice modal)
- `esc` - Cancel

**ModeSelectRange:**
- `j/k` - Adjust end line
- `enter` - Confirm range
- `esc` - Cancel

**New Modes:**
- **ModeChooseTarget**: `s` (section) / `l` (line) / `esc` (cancel)
- **ModeSelectSuggestionType**: `r` (range) / `s` (section) / `esc` (cancel)

---

### Phase 6: Testing & Documentation

**Manual Testing Checklist:**

1. **Section Display:**
   - [ ] Load document with sections
   - [ ] Verify section paths shown in comment list
   - [ ] Verify icons: 📍 for sections, 💬 for lines
   - [ ] Check thread view shows section context

2. **Section Input:**
   - [ ] Navigate to heading, press 'c'
   - [ ] Verify choice modal appears
   - [ ] Choose section, add comment
   - [ ] Verify SectionPath is set in JSON
   - [ ] Choose line, add comment
   - [ ] Verify SectionPath is empty in JSON

3. **Multi-line Suggestions:**
   - [ ] Select range on regular lines
   - [ ] Verify range highlighting
   - [ ] Create suggestion, verify range correct
   - [ ] Select section suggestion on heading
   - [ ] Verify section range used
   - [ ] Verify SectionPath set for section suggestions

4. **Compatibility:**
   - [ ] Load document with CLI-created section comments
   - [ ] Verify they display correctly in TUI
   - [ ] Create section comment in TUI
   - [ ] Verify `./comments list` shows it correctly

5. **Edge Cases:**
   - [ ] Empty document
   - [ ] Document with no headings
   - [ ] Nested sections (3+ levels)
   - [ ] Very long section paths
   - [ ] Mixed section/line comments in same thread

**Unit Tests to Add:**

Create `pkg/tui/sections_test.go`:
```go
func TestGetSectionAtLine(t *testing.T) { /* ... */ }
func TestIsHeadingLine(t *testing.T) { /* ... */ }
func TestGetSectionPath(t *testing.T) { /* ... */ }
```

**Documentation Updates:**

Update `CLAUDE.md` section "TUI View Command" with:
- New modes: ModeChooseTarget, ModeSelectSuggestionType, ModeSelectRange
- Updated keybindings
- Section-based workflow examples
- Multi-line suggestion workflow

---

## Implementation Order

1. **Phase 1** (Foundation) - Required for all other phases
2. **Phase 2** (Display) - Quick win, immediate UX improvement
3. **Phase 3** (Section Input) - Core feature, moderate complexity
4. **Phase 4** (Multi-line) - Advanced feature, builds on Phase 3
5. **Phase 5** (Keybindings) - Update as phases complete
6. **Phase 6** (Testing) - Continuous throughout

---

## Success Criteria

✓ User can add comment to section by navigating to heading and pressing 'c'
✓ User can choose to comment on heading line vs entire section
✓ Section paths displayed in all comment views (list, thread)
✓ Section comments have 📍 icon, line comments have 💬 icon
✓ User can create multi-line suggestions via range selection
✓ User can create section-based suggestions on headings
✓ TUI-created comments with section metadata match CLI output format
✓ All existing TUI features continue to work (no regressions)

---

## Key Files Reference

**Core TUI Files:**
- `pkg/tui/model.go` - Main state machine (700+ lines)
- `pkg/tui/modes.go` - ViewMode enum and String() method
- `pkg/tui/rendering.go` - Display functions (renderComments, renderDocument, renderThread)
- `pkg/tui/styles.go` - Lipgloss styling definitions

**Supporting Files:**
- `pkg/comment/types.go` - Comment struct with SectionID/SectionPath
- `pkg/comment/sections.go` - Section resolution and helpers
- `pkg/comment/helpers.go` - Thread manipulation (AddReplyToThread, etc.)
- `pkg/markdown/parser.go` - ParseDocument() and Section struct

**Testing:**
- `examples/sample.md` - Sample document for manual testing
- `examples/section-test.md` - Section-focused test document (may need creation)

---

## Notes for Implementation

1. **Section Parsing**: The `markdown.ParseDocument()` function already exists and works well. Use it.

2. **Storage Format**: No changes needed to v2.0 JSON format - it already supports SectionID and SectionPath fields.

3. **Visual Selection Pattern**: The range selection mode is similar to vim's visual mode. Keep it simple - just j/k to adjust, enter to confirm.

4. **Modal Pattern**: ModeChooseTarget and ModeSelectSuggestionType follow the same pattern as existing ModeResolve - show options, handle single keypress, transition.

5. **Backwards Compatibility**: All changes are additive. Documents without sections, or comments without SectionPath, will continue to work with graceful fallbacks.

6. **Performance**: Section parsing is lightweight. Only parse once on document load, store in Model.

7. **Testing Strategy**: Start with manual testing on example documents, then add unit tests for helper functions, finally add integration tests for full workflows.
