# RFC: thread citations — `thread:c1abc` as a first-class reference

## Summary

Documents cite code by `file:line`; decisions live in comment threads with no equivalent address.
This RFC makes thread IDs citable: `thread:c1abc` for this document, `thread:path.md#c1abc` across documents — peekable in the TUI (`f`), fetchable verbatim by agents (`comments get 'thread:...' --from <citing-doc>`).
Compounding depends on it: a design that cites its vetoed alternatives by thread makes the reasoning trail navigable instead of archaeological.

## Motivation

Review decisions were being made in threads and then re-argued in later rounds because nothing linked the new text to the old debate.
Chat-made decisions were worse — invisible to the record entirely.
A citation form closes the loop: the doc states the decision, the thread holds the who/why, and both humans and agents traverse the link in one step.

## Guide-level Explanation

Write `thread:c6mv7` anywhere in prose to cite a thread in the same document.
Write `thread:research.md#c6mv7` to cite one in a sibling document; the path resolves relative to the CITING document, never the reader's working directory.
In the TUI, `f` on the citation renders the whole debate — root, replies, resolution — beside the document; `Enter` jumps to the thread's anchor line.
Agents paste the citation verbatim into `comments get`, passing `--from` so relative paths resolve correctly.
Citing a reply resolves to its containing root: you always get the full thought-trail.

## Reference-level Explanation

Parsing: `threadRefRe` in pkg/markdown/refs.go matches the scheme, path optional, ID shaped `c[a-z0-9]{4,12}`; fenced code follows the comment-trail rule shared with file refs.
Resolution: `ResolveThreadCitation` (pkg/comment/storage.go) maps citation + citing-doc to (sidecar, ID); `ReadThread` reads the sidecar RAW — no re-anchor side effects from peeking — and returns the containing root.
The TUI resolves at load into the same refsByLine map file citations use.

## Drawbacks

A deleted sidecar kills its citations — the same exposure file citations already carry, but review history is likelier to be treated as disposable than code.
The explicit scheme adds syntax to learn, though it removes the heading-anchor ambiguity a bare `#id` would have had.

## Rationale and Alternatives

### Bare fragment (`doc.md#c1abc`) — rejected

Collides with markdown heading anchors, forcing precedence rules and sidecar-presence heuristics; the explicit scheme needs neither (decided in review, thread:../plan-compounding-rpi.md#cmvlt).

## Unresolved Questions

- Should resolving a cited thread notify documents that cite it, or is staleness acceptable until the next read?
