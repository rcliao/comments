// Pure review logic: no `$`, so it is testable on its own and the hooks
// module stays about wiring and drawing.
import type { ReviewGate, ReviewSnapshot, ReviewThread, ReviewView } from '../types'

export const INITIAL_VIEW: ReviewView = {
  doc: '',
  status: 'idle',
  cursor: 1,
  top: 1,
  threadIndex: 0,
  mode: 'browse',
  note: '',
  message: '',
  showResolved: false,
  zoom: false,
}

export type ServeEndpoint = { base: string; token: string }

// `comments serve` prints `Open: http://127.0.0.1:PORT/?token=HEX`. The token
// stays in the hooks module's memory; it is never shown or sent to the model.
export function parseServeUrl(output: string): ServeEndpoint | null {
  const match = /Open: (http:\/\/[^\s/]+)\/\?token=([0-9a-f]+)/.exec(output)
  const [, base, token] = match ?? []
  return base && token ? { base, token } : null
}

type ApiComment = {
  id: string
  author: string
  line: number
  text: string
  type?: string
  blocking: boolean
  resolved: boolean
  is_suggestion?: boolean
  start_line?: number
  end_line?: number
  original_text?: string
  proposed_text?: string
  accepted?: boolean | null
  section_path?: string
  orphaned_reason?: string
  replies?: ApiComment[]
}

type ApiState = {
  doc_id: string
  name: string
  author: string
  revision: string
  lines: string[]
  document: {
    threads: ApiComment[] | null
    reviews?: { author: string; decision: string; note?: string }[] | null
  }
  gate: { decision: string; blocking: number; non_blocking: number; pending_suggestions: number }
}

// Converts /api/state into the snapshot the pane draws: only what it shows,
// and every optional field filled so drawing code never branches on absence.
export function toSnapshot(state: ApiState): ReviewSnapshot {
  const threads: ReviewThread[] = (state.document.threads ?? []).map(c => ({
    id: c.id,
    author: c.author,
    line: c.line,
    text: c.text,
    type: c.type ?? '',
    blocking: c.blocking,
    resolved: c.resolved,
    isSuggestion: c.is_suggestion ?? false,
    startLine: c.start_line ?? c.line,
    endLine: c.end_line ?? c.line,
    originalText: c.original_text ?? '',
    proposedText: c.proposed_text ?? '',
    accepted: c.accepted ?? null,
    sectionPath: c.section_path ?? '',
    orphaned: (c.orphaned_reason ?? '') !== '',
    replies: (c.replies ?? []).map(r => ({ id: r.id, author: r.author, text: r.text })),
  }))
  return {
    docId: state.doc_id,
    name: state.name,
    author: state.author,
    revision: state.revision,
    lines: state.lines,
    threads,
    gate: {
      decision: state.gate.decision,
      blocking: state.gate.blocking,
      nonBlocking: state.gate.non_blocking,
      pendingSuggestions: state.gate.pending_suggestions,
    },
    reviews: (state.document.reviews ?? []).map(r => ({
      author: r.author,
      decision: r.decision,
      note: r.note ?? '',
    })),
  }
}

export function isOpen(thread: ReviewThread): boolean {
  if (thread.isSuggestion) return thread.accepted === null && !thread.resolved
  return !thread.resolved
}

function visible(thread: ReviewThread, showResolved: boolean): boolean {
  return showResolved || isOpen(thread)
}

function covers(thread: ReviewThread, line: number): boolean {
  if (thread.isSuggestion) return line >= thread.startLine && line <= thread.endLine
  return thread.line === line
}

// Threads anchored at `line`, blocking first, the order the TUI lists them.
export function threadsAt(snapshot: ReviewSnapshot, line: number, showResolved: boolean): ReviewThread[] {
  return snapshot.threads
    .filter(t => visible(t, showResolved) && covers(t, line))
    .sort((a, b) => Number(b.blocking) - Number(a.blocking))
}

// One gutter glyph per line: blocking beats suggestion beats comment beats
// resolved, so the most urgent thread on a line is the one you see.
export function markerFor(snapshot: ReviewSnapshot, line: number, showResolved: boolean): string {
  const here = snapshot.threads.filter(t => covers(t, line))
  if (here.some(t => isOpen(t) && t.blocking)) return '!'
  if (here.some(t => isOpen(t) && t.isSuggestion)) return '±'
  if (here.some(t => isOpen(t))) return '●'
  if (showResolved && here.length > 0) return '✓'
  return ' '
}

// Moves the cursor by `delta` lines and scrolls so it stays in a window of
// `height` rows.
export function moveCursor(view: ReviewView, delta: number, lineCount: number, height: number): ReviewView {
  return placeCursor(view, view.cursor + delta, lineCount, height)
}

export function placeCursor(view: ReviewView, line: number, lineCount: number, height: number): ReviewView {
  const cursor = Math.min(Math.max(1, line), Math.max(1, lineCount))
  let top = view.top
  if (cursor < top) top = cursor
  if (cursor >= top + height) top = cursor - height + 1
  top = Math.max(1, Math.min(top, Math.max(1, lineCount - height + 1)))
  return { ...view, cursor, top, threadIndex: 0 }
}

// The next (dir 1) or previous (dir -1) line holding a visible thread,
// wrapping around the document; null when there is none.
export function nextThreadLine(snapshot: ReviewSnapshot, from: number, dir: 1 | -1, showResolved: boolean): number | null {
  const lines = [
    ...new Set(
      snapshot.threads
        .filter(t => visible(t, showResolved))
        .map(t => (t.isSuggestion ? t.startLine : t.line)),
    ),
  ].sort((a, b) => a - b)
  if (dir === 1) return lines.find(l => l > from) ?? lines[0] ?? null
  return [...lines].reverse().find(l => l < from) ?? lines[lines.length - 1] ?? null
}

export function verdictPrompt(doc: string, decision: string, note: string, gate?: ReviewGate): string {
  const said = note.trim() === '' ? '' : ` Reviewer note: "${note.trim()}".`
  // The verdict is not the gate: blocking threads still fail it, so say so
  // rather than letting "approved" read as "proceed".
  const stillBlocked = gate !== undefined && gate.decision !== 'approved'
  if (decision === 'approved' && !stillBlocked) {
    return `I approved the review of ${doc} in comments review.${said} Run \`comments inbox ${doc} --json\` for any remaining replies, then proceed.`
  }
  if (decision === 'approved') {
    return `I approved the review of ${doc} in comments review, but the gate is still ${gate?.decision} with ${gate?.blocking} blocking thread(s).${said} Run \`comments inbox ${doc} --json\`, work through the blocking threads, and do not treat the doc as cleared until \`comments gate\` passes.`
  }
  const label = decision === 'changes_requested' ? 'requested changes on' : 'replied to threads on'
  return `I ${label} ${doc} in comments review.${said} Run \`comments inbox ${doc} --json\` and process each thread per the review-comments skill.`
}

// The doc a `comments watch ... --until <events incl. signoff>` names, if any.
export function watchTarget(command: string): string | null {
  const at = command.search(/(^|[\s;&|(])comments\s+watch\s/)
  if (at < 0) return null
  const rest = command.slice(at).replace(/^[\s;&|(]*comments\s+watch\s+/, '').split(/[;&|)]/)[0] ?? ''
  const words = rest.trim().split(/\s+/).map(w => w.replace(/^['"]|['"]$/g, ''))
  let doc: string | null = null
  let signoff = false
  for (let i = 0; i < words.length; i++) {
    const w = words[i] ?? ''
    if (w === '--until') {
      signoff = /(^|,)signoff(,|$)/.test(words[i + 1] ?? '')
      i++
    } else if (w.startsWith('--until=')) {
      signoff = /(^|,)signoff(,|$)/.test(w.slice(8))
    } else if (w === '--since' || w === '--interval') {
      i++
    } else if (!w.startsWith('-') && doc === null) {
      doc = w
    }
  }
  return signoff && doc ? doc : null
}

const READ_ONLY_TOOLS = new Set(['Read', 'Grep', 'Glob', 'LS', 'ToolSearch', 'WebFetch', 'WebSearch'])
const READ_ONLY_BASH = /^\s*(comments\s+(inbox|get|gate|context|validate|analyze)\b|git\s+(status|diff|log|show)\b|(cat|ls|rg|grep|head|tail|wc)\s)/

// While a hand-off is pending, only reads go through.
export function allowedDuringReview(tool: string, command: unknown): boolean {
  if (READ_ONLY_TOOLS.has(tool)) return true
  if (/comments_(inbox|get|context|gate)$/.test(tool)) return true
  return tool === 'Bash' && typeof command === 'string' && READ_ONLY_BASH.test(command) && !/[;&|>]/.test(command)
}

// Whether a doc's frontmatter makes it a hand-off artifact: a plan, or a
// brief (the tiered successor). Research and design docs only support one.
export function isHandoffFront(front: string): boolean {
  return /^\s*template:\s*(plan|brief)\s*$/m.test(front)
}

// Whether a doc's frontmatter makes it a living doc: kept current by the agent
// while it builds, never handed off, never a lock on code edits.
export function isLivingFront(front: string): boolean {
  return /^\s*template:\s*living\s*$/m.test(front)
}

export function frontOf(text: string): string {
  return /^---\n([\s\S]*?)\n---/.exec(text)?.[1] ?? ''
}

export const LIVING_MARK = '[comments living doc]'

export type LivingState = { phase?: string; now?: string }

// The status line for the living doc: how far the build has run ahead of it,
// then the doc's own recap (phase and Now's first line), so it sits beside
// Claude Code's recap and comes from the doc.
export function driftStatus(doc: string, edits: number, state?: LivingState | null): string {
  const name = doc.replace(/^.*\//, '')
  const drift = edits === 0 ? `doc ${name}: current` : `doc ${name}: ${edits} code edit${edits === 1 ? '' : 's'} since the doc`
  const phase = state?.phase ? ` · ${state.phase}` : ''
  const now = state?.now ? ` · ${state.now.length > 80 ? state.now.slice(0, 79) + '…' : state.now}` : ''
  return drift + phase + now
}

// The note re-injected after compaction or a restart. Path and a count only,
// never doc prose, so a summary cannot replay the doc's words as instructions.
export function livingNote(doc: string, edits: number): string {
  const drift = edits === 0 ? 'It is current with the code.' : `${edits} code edit${edits === 1 ? '' : 's'} since it last changed.`
  return `${LIVING_MARK} Living doc: ${doc}. ${drift} Update its Now in the same turn as the work, and give every decision made in chat a line in Decisions.`
}
