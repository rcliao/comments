# Testing Guide: Comments & Suggestions System

This guide walks you through testing all features of the comments and suggestions system.

## Test Document

We've created `test-workflow.md` with pre-populated comments and suggestions for testing.

## What's Already Been Set Up (via CLI)

The following was added programmatically to demonstrate batch operations:

### 1. Batch Comments Added
```bash
./comments batch-add examples/test-workflow.md --json initial-comments.json
```
- 4 comments added by different users (reviewer, editor, tech-lead, designer)
- Comment types: Q (Question), S (Suggestion), T (TODO), E (Enhancement)

### 2. Batch Suggestions Added
```bash
./comments batch-add examples/test-workflow.md --json suggestions.json
```
- 3 edit suggestions created:
  - Line 5: Simplify intro (line replacement)
  - Lines 8-11: Improve list formatting (multi-line replacement)
  - Line 29: Add specific example (line replacement)

### 3. Single Command Examples
```bash
# Single reply
./comments reply examples/test-workflow.md --thread <id> --author "author" --text "Response"

# Single comment
./comments add examples/test-workflow.md --line 45 --author "qa-engineer" --text "Add test coverage" --type "T"
```

### 4. Batch Replies
```bash
./comments batch-reply examples/test-workflow.md --json replies.json
```
- 2 replies added to existing comment threads

## Your Testing Tasks (Human/TUI)

Now you'll use the interactive TUI to test all features:

### Test 1: Browse Mode - View Comments
```bash
./comments view examples/test-workflow.md
```

**What to test:**
- [ ] Document displays on the left (60% width)
- [ ] Comments display on the right (40% width)
- [ ] See 8 total comments/suggestions
- [ ] Comments show type indicators: [Q], [S], [T], [E]
- [ ] Suggestions show [📝 SUGGESTION] indicator
- [ ] Navigate with `j/k` or arrow keys
- [ ] Press `R` to toggle showing resolved comments

**Expected:** Clean split view with syntax highlighting for markdown headers

---

### Test 2: Thread View - Read Discussions
```bash
# In browse mode, press Enter on a comment
```

**What to test:**
- [ ] Press `Enter` on comment #1 (reviewer's question)
- [ ] See thread view with:
  - Document context (2 lines before/after)
  - Root comment in bordered box
  - Replies section showing 1 reply from author
- [ ] Navigate with scroll keys
- [ ] Press `Esc` to return to browse mode

**Expected:** Full thread view with context

---

### Test 3: Reply to Comments
```bash
# In thread view, press 'r'
```

**What to test:**
- [ ] Open any comment thread
- [ ] Press `r` to reply
- [ ] Type your reply in the textarea
- [ ] Press `Ctrl+S` to submit
- [ ] See your reply appear in thread
- [ ] File is saved (check timestamp)

**Expected:** Reply added and visible immediately

---

### Test 4: Add New Comment (via Line Select)
```bash
# In browse mode, press 'c'
```

**What to test:**
- [ ] Press `c` to enter line selection mode
- [ ] Mode changes to "LINE_SELECT" in title
- [ ] Help text shows: "j/k: move • ... • c: comment • s: suggestion"
- [ ] Navigate to line 35 with `j/k`
- [ ] See cursor indicator (►) on selected line
- [ ] Press `c` or `Enter` to add comment
- [ ] Type comment text
- [ ] Press `Ctrl+S` to submit
- [ ] Return to browse mode and see new comment in list

**Expected:** New comment appears at line 35

---

### Test 5: Create Suggestion (NEW FEATURE!)
```bash
# In line select mode, press 's'
```

**What to test:**
- [ ] Press `c` to enter line selection mode
- [ ] Navigate to line 40
- [ ] Press **`s`** to create suggestion
- [ ] See "Add Suggestion" modal
- [ ] Original text is shown in a box
- [ ] Textarea is pre-filled with original text
- [ ] Edit the text to your proposed version
- [ ] Press `Ctrl+S` or `Ctrl+D` to submit
- [ ] See suggestion appear in comment list with [📝 SUGGESTION]

**Expected:** Suggestion created and visible in list

---

### Test 6: Review Suggestion - Accept
```bash
# Navigate to a pending suggestion and press Enter
```

**What to test:**
- [ ] Find suggestion #2 (copywriter's line 7 suggestion - "Simplify the intro sentence")
- [ ] Press `Enter` to view thread
- [ ] See suggestion details box showing:
  - Suggestion Type: line
  - Lines: 7-7
  - Original text
  - Proposed text
  - "Press 'a' to accept or 'x' to reject"
- [ ] Press `a` to accept
- [ ] See preview dialog showing full document with changes
- [ ] Press `y` or `Enter` to confirm
- [ ] See success message
- [ ] Document content is updated
- [ ] Suggestion now shows [✓ ACCEPTED]

**Expected:** Suggestion applied, document modified, positions recalculated

---

### Test 7: Review Suggestion - Reject
```bash
# View another suggestion thread
```

**What to test:**
- [ ] Find suggestion #4 (copywriter's lines 13-16 suggestion - "Make the list more scannable")
- [ ] Press `Enter` to view thread
- [ ] Press `x` to reject
- [ ] Suggestion marked as [✗ REJECTED] immediately
- [ ] Document content NOT changed
- [ ] Can still see the suggestion in list

**Expected:** Suggestion rejected but preserved in history

---

### Test 8: Resolve Thread
```bash
# In thread view, press 'x' for non-suggestion
```

**What to test:**
- [ ] Open a regular comment thread (not a suggestion)
- [ ] Press `x` to resolve
- [ ] See confirmation dialog
- [ ] Press `y` to confirm
- [ ] Thread disappears from unresolved list
- [ ] Press `R` to toggle resolved view
- [ ] See resolved thread with ✓ indicator

**Expected:** Thread marked resolved

---

### Test 9: Toggle Resolved Comments
```bash
# In browse mode, press 'R'
```

**What to test:**
- [ ] Press `R` to show all comments (resolved + unresolved)
- [ ] Comment count changes in header
- [ ] Resolved threads show ✓ prefix
- [ ] Press `R` again to hide resolved
- [ ] Only unresolved comments visible

**Expected:** Toggle works smoothly

---

### Test 10: Keyboard Navigation
**What to test:**
- [ ] `j/k` to navigate comments
- [ ] `Ctrl+D` for page down
- [ ] `Ctrl+U` for page up
- [ ] `g` to go to first line (in line select)
- [ ] `G` to go to last line (in line select)
- [ ] `Esc` to cancel/go back
- [ ] `q` to quit application

**Expected:** All keyboard shortcuts work as documented

---

## CLI Testing (Already Completed)

These were used to set up the test document:

### List Comments
```bash
./comments list examples/test-workflow.md
./comments list examples/test-workflow.md --format json
./comments list examples/test-workflow.md --author copywriter
./comments list examples/test-workflow.md --type S
```

### Accept Suggestion via CLI
```bash
# Preview first
./comments accept examples/test-workflow.md --suggestion <id> --preview

# Then accept
./comments accept examples/test-workflow.md --suggestion <id>
```

### Batch Accept
```bash
# Accept all suggestions from copywriter
./comments batch-accept examples/test-workflow.md --author copywriter
```

### Reject Suggestion via CLI
```bash
./comments reject examples/test-workflow.md --suggestion <id>
```

---

## Verification Checklist

After testing, verify:

- [ ] All TUI modes work (Browse, LineSelect, AddComment, AddSuggestion, ThreadView, Reply, Resolve, ReviewSuggestion)
- [ ] Comments and suggestions visually distinguished
- [ ] Acceptance states clear (pending/accepted/rejected)
- [ ] Document updates correctly when suggestions accepted
- [ ] Position tracking works (comments stay on correct lines)
- [ ] Sidecar JSON file is valid (`cat examples/test-workflow.md.comments.json | python -m json.tool`)
- [ ] No crashes or errors during normal usage
- [ ] Help text is accurate and helpful

---

## Expected File State

After all testing:
- `examples/test-workflow.md` - Modified markdown content (if suggestions accepted)
- `examples/test-workflow.md.comments.json` - Contains ~10-15 comments/suggestions with various states

---

## Tips

- The TUI uses Vim-style keybindings (`j/k` for navigation)
- All modals can be cancelled with `Esc`
- `Ctrl+S` submits in input modes
- Suggestions show metadata in thread view
- Press `R` (capital R) to toggle resolved comments
- Line numbers in document match comment positions

---

## Troubleshooting

**Document doesn't load:**
- Check file path is correct
- Ensure sidecar JSON is valid

**Suggestions don't show indicators:**
- Check comment has `SuggestionType` field in JSON
- Verify `AcceptanceState` is set

**Accept fails:**
- Original text may have changed (validation error)
- Check error message for details

---

## Success Criteria

✅ All 10 test cases pass
✅ No crashes or hangs
✅ Data persists correctly
✅ UI is responsive and clear
✅ Help text guides users effectively

Happy testing! 🎉
