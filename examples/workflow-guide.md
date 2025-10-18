# Collaborative Document Writing with Comments CLI {>>[@claude-code:c1760732678547858000:c1760732678547858000:1:2025-10-17T13:24:38-07:00:true] [FEATURE] Added document context display in thread view! Now shows 2 lines before/after the commented line with highlighting. <<}

A practical guide to using the comments CLI for human-AI collaboration on document creation. {>>[@rcliao:c1760750402333942000:c1760750402333942000:3:2025-10-17T18:20:02-07:00:false] [S] or more of a note to myself: we can add a recording of the workflow here once we feel more comfortable as key demo <<}

## The Problem

Traditional document collaboration with AI agents has a fundamental issue: when you ask an LLM to improve a document, it often rewrites the entire thing. This makes it hard to:
- Review changes incrementally
- Provide contextual feedback
- Iterate on specific sections
- Maintain authorship and voice

### Comparison: Comments CLI vs Traditional Methods

| Aspect | Traditional (Same-Doc Editing) | Comments CLI |
|--------|-------------------------------|--------------|
| **Workflow** | AI rewrites entire sections | AI adds targeted comments |
| **Review** | Diff entire document | Review comment-by-comment |
| **Authorship** | AI voice overwrites human voice | Human maintains control, AI suggests |
| **Iteration** | Start over each time | Incremental improvements |
| **Context** | Lost in full rewrites | Preserved with inline comments |
| **Collaboration** | Merge conflicts, messy history | Structured threads, clear discussions |
| **Control** | Accept/reject large changes | Accept/reject individual suggestions |
| **Focus** | Scattered across document | Organized by type ([Q], [S], [B], etc.) |

The comments CLI solves this by bringing Google Docs-style commenting to terminal-based markdown editing.

## Core Workflow

```
┌─────────────┐      ┌─────────────┐      ┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│   Phase 1   │      │   Phase 2   │      │   Phase 3   │      │   Phase 4   │      │   Phase 5   │
│   Initial   │ ───> │   Review &  │ ───> │   Act on    │ ───> │   Iterate   │ ───> │   Publish   │
│   Draft     │      │   Comment   │      │   Comments  │      │             │      │             │
└─────────────┘      └─────────────┘      └─────────────┘      └─────────────┘      └─────────────┘
     │                     │                     │                     │                     │
     │                     │                     │                     │                     │
  Write or              Add [Q]              Resolve,              Repeat 2-3            Strip all
  generate           [S] [B] [T] [E]           Reply,               until done           comments
  content             comments              or Dismiss                                  for clean
                                                                                         output
```

### Phase 1: Initial Draft

Start with either:

**Option A: Human-written draft**
```bash
# Create your markdown file
vim document.md
# Write your initial thoughts
```

**Option B: LLM-generated draft**
```bash
# Use your preferred LLM tool (Claude Code, ChatGPT, etc.) to generate content
# Then save it to your markdown file
# The comments CLI stays focused on commenting, not content generation
```

### Phase 2: Review & Comment

This is where the magic happens. Instead of rewriting, add comments at specific locations.

**Human Review (Interactive)**
```bash
# Open TUI for review
comments view document.md

# Navigate with j/k
# Press 'c' to add comment at current line
# Type your feedback
# Press Ctrl+S to save
```

**AI Review (Automated)**
```bash {>>[@reviewer:c1760730896862957000:c1760730896862957000:78:2025-10-17T12:54:56-07:00:true] [S] Consider adding a visual diagram showing the workflow phases <<} {>>[@rcliao:c1760736851100477000:c1760730896862957000:78:2025-10-17T14:34:11-07:00:true] Yes, a visual diagram works so that reader can know what to expect from the program <<} {>>[@claude-code:c1760750630424304000:c1760730896862957000:78:2025-10-17T18:23:50-07:00:true] Added visual ASCII diagram showing all 5 workflow phases with descriptions. Shows the flow from Draft -> Review -> Act -> Iterate -> Publish. <<}
# Use structured comment types for clarity
comments add document.md --line 10 --type Q --text "Is this explanation clear enough?"
comments add document.md --line 15 --type B --text "This section contradicts the intro"
comments add document.md --line 20 --type S --text "Consider adding a code example here"

# Or use your LLM tool (Claude Code, etc.) to read the doc and add comments via CLI
# The tool exports JSON for programmatic access:
comments export document.md --output comments.json
# (Your LLM can parse this and suggest improvements)
```

**Review Checklist**:
- [ ] Does the introduction clearly state the purpose?
- [ ] Are all technical terms defined?
- [ ] Do examples illustrate the concepts?
- [ ] Is the structure logical and easy to follow?
- [ ] Are there any gaps in the explanation?

### Phase 3: Act on Comments

Review comments and decide: {>>[@rcliao:c1760742980787894000:c1760742980787894000:99:2025-10-17T16:16:20-07:00:true] [Q] is there a way that we can make the comments useful to LLM to make edit to the doc as well? <<} {>>[@claude-code:c1760743187763632000:c1760742980787894000:99:2025-10-17T16:19:47-07:00:true] Yes\! Comments could be machine-readable. The CriticMarkup format is already structured. An LLM could parse: 'Show me all [B] blockers' or 'Apply suggested edits from @reviewer'. We'd need a 'comments export --format json' command to make this easy for LLM integrations. <<} {>>[@rcliao:c1760748375557869000:c1760742980787894000:99:2025-10-17T17:46:15-07:00:true] Agreed and liked about the export json option <<} {>>[@rcliao:c1760748552716738000:c1760742980787894000:99:2025-10-17T17:49:12-07:00:true] Let's implement 'comments export --format json' command. Output structure: {"comments": [...], "threads": {...}, "metadata": {...}}. This enables LLM tools (like Claude Code) to parse comments programmatically and apply edits based on suggestions. <<}

**Resolve** - Comment addressed, changes made
```bash
comments resolve document.md --thread c123
```

**Reply** - Need clarification or discussion
```bash
comments reply document.md --thread c123 --text "Good point, I'll expand on this"
```

**Dismiss** - Disagree with suggestion
```bash
comments reply document.md --thread c123 --text "Keeping this as-is because..."
comments resolve document.md --thread c123
```

### Phase 4: Iterate

Repeat phases 2-3 until document quality meets requirements.

**Convergence Indicators**:
- No unresolved blocking comments {>>[@reviewer:c1760730918148284000:c1760730918148284000:122:2025-10-17T12:55:18-07:00:true] [B] The 'strip comments' command doesn't exist yet - this is a critical gap <<} {>>[@rcliao:c1760735792798777000:c1760730918148284000:122:2025-10-17T14:16:32-07:00:true] Instead of the strip command, we should do publish command as the workflow is write -> review -> edit -> publish <<}
- All questions answered
- Suggestions either implemented or dismissed with reasoning
- Document reads smoothly end-to-end

### Phase 5: Publish

Strip comments and publish clean markdown:

```bash
# Output clean markdown without comment markup
comments publish document.md --output final.md

# Or pipe to stdout for further processing
comments publish document.md | pandoc -o document.pdf
```

## Best Practices

### Comment Conventions

Use structured prefixes for clarity: {>>[@rcliao:c1760742942408246000:c1760742942408246000:143:2025-10-17T16:15:42-07:00:true] [Q] shall we consider bringing these as convention to our comment utilities? <<} {>>[@claude-code:c1760743232414863000:c1760742942408246000:143:2025-10-17T16:20:32-07:00:true] Absolutely\! This would be powerful. Ideas: 1) 'comments list --type B' to filter blockers, 2) 'comments add --type Q' to auto-prefix with [Q], 3) TUI could color-code by type, 4) 'comments stats' showing breakdown by type. Makes prioritization trivial\! <<} {>>[@rcliao:c1760748419579568000:c1760742942408246000:143:2025-10-17T17:46:59-07:00:true] I like 1, 2, and 3. 4 may be over killed for the terminal tool for now <<} {>>[@rcliao:c1760748562603412000:c1760742942408246000:143:2025-10-17T17:49:22-07:00:true] Agreed on 1-3\! Implementation plan: 1) 'comments list --type Q' filters by [Q]/[S]/[B]/[T]/[E] prefix, 2) 'comments add --type Q' auto-adds prefix, 3) TUI color-coding: red=[B], yellow=[Q], blue=[S], etc. Skip stats for now - keep it lean and focused. <<}

- `[Q]` - Question: Needs answer before proceeding
- `[S]` - Suggestion: Optional improvement
- `[B]` - Blocker: Must be addressed before publishing
- `[T]` - Technical: Requires technical review
- `[E]` - Editorial: Grammar, style, tone

Example:
```bash
comments add doc.md --line 42 --text "[B] This section contradicts the intro"
comments add doc.md --line 50 --text "[S] Consider adding a diagram here"
comments add doc.md --line 60 --text "[Q] What's the expected audience level?"
```

### Review Strategies

**Top-Down Review**:
1. Structure and flow first
2. Content accuracy second
3. Style and polish last

**Parallel Review**:
- Human focuses on: voice, examples, clarity
- AI focuses on: completeness, consistency, technical accuracy

**Incremental Review**:
- Review and resolve section-by-section
- Don't wait for entire document to be "done"

**Incremental Exploration**:
Instead of asking AI to rewrite entire sections, use comments to explore improvements incrementally:

1. **Start with a Specific Question**: Use `[Q]` comments to ask about specific aspects
   ```bash
   comments add doc.md --line 42 --type Q --text "Could this example be clearer?"
   ```

2. **Get Targeted Suggestions**: AI responds with focused suggestions in comments, not rewrites
   ```bash
   comments reply doc.md --thread c123 --text "Try breaking it into 2 examples: basic + advanced" --author "claude"
   ```

3. **Experiment Section-by-Section**: Make small changes, review, iterate
   - Edit one section based on suggestion
   - Review the change in context
   - If it works, apply similar pattern elsewhere
   - If it doesn't, try different approach

4. **Use Comment Types to Guide Exploration**:
   - `[S]` for optional improvements to try
   - `[Q]` for clarifying questions before changing
   - `[B]` for critical issues that need addressing
   - `[T]` for technical accuracy checks
   - `[E]` for style/tone experiments

This approach preserves your voice while benefiting from AI suggestions, and lets you learn what works through iteration rather than wholesale replacement.

### When to Use Each Command

**`view`** - Interactive review and discussion
- When you want to see full context
- When navigating complex thread conversations
- When doing deep reading/editing

**`add`** - Quick, targeted feedback
- When you know exactly what needs commenting
- When automating review scripts
- When AI is providing feedback

**`reply`** - Building on existing comments
- When discussing a specific point
- When clarifying intent
- When multiple reviewers coordinate

**`resolve`** - Closing the loop
- When change is implemented
- When suggestion is dismissed with reason
- When question is answered

**`export`** - Programmatic access
- When integrating with external tools or LLMs
- When analyzing comment patterns or metrics
- When archiving review history

**`publish`** - Final output
- When creating clean document for distribution
- When converting to other formats (via piping)
- When removing all review markup

## Example Workflows

### Workflow 1: Solo Writing with AI Review

```bash
# 1. Write initial draft
vim guide.md

# 2. Export for AI review (use your preferred AI tool)
comments export guide.md --output review-request.json
# Give this JSON to your AI tool (Claude Code, ChatGPT, etc.)
# AI adds comments via: comments add guide.md --type Q --text "..." --author "ai"

# 3. Open TUI to see all comments
comments view guide.md

# 4. Act on each comment
# (Make edits in your editor)

# 5. Resolve comments as you go
comments resolve guide.md --thread c123
comments resolve guide.md --thread c456

# 6. Filter to check what's left
comments list guide.md --type B  # Check remaining blockers

# 7. Publish
comments publish guide.md --output final.md
```

### Workflow 2: Collaborative Writing (Human + Human)

```bash
# Author writes draft
vim proposal.md
git commit -m "Initial proposal draft"

# Reviewer adds comments
comments view proposal.md
# (Add comments via TUI)
git commit -m "Review comments"

# Author responds
comments view proposal.md
# (Reply to comments via TUI)
git commit -m "Addressed review feedback"

# Back-and-forth continues
# Eventually all comments resolved
comments list proposal.md
# (Should show only resolved comments)
```

### Workflow 3: AI-First Drafting

```bash
# 1. Create empty file with outline
cat > article.md << 'EOF'
# Title

## Introduction
## Main Content
## Conclusion
EOF

# 2. Use your AI tool to expand each section
# (e.g., use Claude Code, ChatGPT web interface, or command-line tools)
# Save the AI-generated content to article.md

# 3. Add review comments on the AI-generated content
comments add article.md --line 3 --type Q --text "Is this tone appropriate?"
comments add article.md --line 5 --type B --text "This needs more examples"
comments add article.md --line 7 --type S --text "Consider adding a call-to-action"

# 4. Review all comments with color-coding
comments view article.md

# 5. Export comments for AI to review
comments export article.md --output feedback.json
# (Share with your AI tool to get revision suggestions)

# 6. Iterate and publish when done
comments publish article.md --output final-article.md
```

## Advanced Techniques

### Comment Threading for Discussion

Build multi-turn discussions in comments:

```bash
# Initial question
comments add doc.md --line 20 --text "[Q] Should we include code examples?"

# Get thread ID from output
# ThreadID: c789

# AI responds
comments reply doc.md --thread c789 --text "Yes, code examples would help. Suggest Python." --author "claude"

# Human follow-up
comments reply doc.md --thread c789 --text "Agreed. Will add Python and JavaScript examples."

# Mark resolved when done
comments resolve doc.md --thread c789
```

### Batch Review with Scripts

Automate common review patterns:

```bash
#!/bin/bash
# review-script.sh

FILE=$1

# Export comments for analysis
comments export $FILE --output /tmp/comments.json

# Filter and list by type
echo "=== Blockers ==="
comments list $FILE --type B

echo "=== Questions ==="
comments list $FILE --type Q

echo "=== Suggestions ==="
comments list $FILE --type S

# Check resolved status
TOTAL=$(cat /tmp/comments.json | jq '.metadata.total_comments')
RESOLVED=$(cat /tmp/comments.json | jq '[.comments[] | select(.Resolved)] | length')
echo "Progress: $RESOLVED/$TOTAL comments resolved"
```

### Progressive Disclosure

Review in layers using comment types:

```bash
# Layer 1: Structure (use Technical type)
comments add doc.md --line 1 --type T --text "Overall structure review needed"
# (Use your AI tool to review structure and add T-type comments)

# Layer 2: Content (use Blocker type for critical issues)
comments list doc.md --type T  # Review structural comments first
comments add doc.md --line 5 --type B --text "Missing key concept explanation"

# Layer 3: Style (use Editorial and Suggestion types)
comments add doc.md --line 10 --type E --text "Inconsistent tone"
comments add doc.md --line 15 --type S --text "Consider simpler wording"

# Filter by priority
comments list doc.md --type B  # Fix blockers first
comments list doc.md --type E  # Polish editorial issues last
```

## Troubleshooting

### Too Many Comments

If you have 20+ unresolved comments:
1. Prioritize: resolve blockers first
2. Batch: group similar comments
3. Defer: move nice-to-have suggestions to later iteration
4. Archive: export comments before resolving for future reference

### Comments Getting Stale

If comments reference outdated content:
1. Review comment context in TUI
2. Update comment text if still relevant
3. Resolve if no longer applicable
4. Add new comment if issue persists in different form

### Lost Context

If you forgot why a comment was added:
1. Check comment thread for discussion
2. Look at surrounding document context
3. Reply asking for clarification
4. If truly orphaned, resolve with note

## Future Enhancements

Ideas for improving the workflow:

1. **✅ Comment Export** - JSON export now available (`comments export`)
2. **✅ Comment Type Filtering** - Filter by type with `--type` flag
3. **✅ Clean Publishing** - Strip comments with `comments publish`
4. **Comment Analytics** - Track comment density, resolution time, author patterns
5. **Template Comments** - Predefined review checklists
6. **Comment Linking** - Reference other comments
7. **Diff Mode** - See document changes alongside comments
8. **Comment Search** - Find comments by author, date, content
9. **Comment Status** - Beyond resolved/unresolved (e.g., "in progress", "blocked")
10. **Import from JSON** - Bulk import comments from external tools

## Conclusion

The comments CLI enables a new way of collaborating on documents:
- **Unix Philosophy**: Does one thing well - manages inline comments
- **Tool Agnostic**: Works with any LLM or text editor
- **Structured Reviews**: Type system ([Q], [S], [B], [T], [E]) organizes feedback
- **Programmatic Access**: JSON export enables automation and tool integration
- **Clean Output**: Publish command strips markup for final distribution

Start with the basic workflow, then adapt it to your needs. The key is treating comments as first-class collaboration objects, not just ephemeral notes.

### Integration Examples

**With Claude Code:**
```bash
# Export for AI review
comments export doc.md --output comments.json
# (Claude Code reads JSON and suggests edits)
```

**With Git Workflow:**
```bash
# Review branch
git checkout feature-branch
comments view proposal.md
# Add comments during review
git commit -am "Review comments added"
git push
```

**With Pandoc:**
```bash
# Convert to PDF without comments
comments publish draft.md | pandoc -o final.pdf
```

---

**Meta Note**: This document itself was written following the workflow it describes. Feel free to add comments and help improve it!
