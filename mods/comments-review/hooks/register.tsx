// comments-review: the `comments view` review loop in a Claude Code pane.
//
// The pane is a client of `comments serve`, the existing browser review
// surface: it spawns the server, keeps the bootstrap token in this module's
// memory (never in the transcript, never in $.state), and sends the same
// /api/action requests the browser does. Every mutation, the verdict
// included, therefore runs through the code paths that already guard the
// human surfaces (zone: human resolve guard, revision conflicts, RecordVerdict
// semantics); this mod adds no new way to write a sidecar.
import { atom, read, update } from 'claude-code'
import type { EngineInterface, Register } from 'claude-code'

import type { ReviewGate, ReviewMode, ReviewSnapshot, ReviewThread, ReviewView } from '../types'
import {
  INITIAL_VIEW,
  isOpen,
  markerFor,
  moveCursor,
  nextThreadLine,
  parseServeUrl,
  placeCursor,
  threadsAt,
  toSnapshot,
  verdictPrompt,
  watchTarget,
  allowedDuringReview,
  type ServeEndpoint,
  isHandoffFront,
  driftStatus,
  frontOf,
  isLivingFront,
  LIVING_MARK,
  livingNote,
} from './review'

const PANE = 'comments-review'
const view = atom({ plugin: 'comments-review', key: 'view' } as const, INITIAL_VIEW)
const snapshot = atom({ plugin: 'comments-review', key: 'snapshot' } as const, null as ReviewSnapshot | null)
// The plan whose approval gates code edits, and a plan the user unlocked by
// hand. Kept in $.state so a reload of the mod cannot silently open the gate.
const activePlan = atom({ plugin: 'comments-review', key: 'activePlan' } as const, null as string | null)
const unlockedPlan = atom({ plugin: 'comments-review', key: 'unlockedPlan' } as const, null as string | null)
// Whether this session was already reminded of the contract. session.start
// fires again on every hot reload of the mod; $.state outlives a reload but
// not the process, so this keeps the reminder to once per session.
// The session's living doc and the code edits made since it last changed.
const livingDoc = atom({ plugin: 'comments-review', key: 'livingDoc' } as const, null as string | null)
const drift = atom({ plugin: 'comments-review', key: 'drift' } as const, 0)
const reminded = atom({ plugin: 'comments-review', key: 'reminded' } as const, false)

// The tallest the thread panel grows: header, body and a few replies.
const THREAD_ROWS = 6

type Server = { endpoint: ServeEndpoint | null; stop: () => void; doc: string }

let binary = 'comments'
let server: Server | null = null
let docRows = 20

function setView($: EngineInterface, fn: (v: ReviewView) => ReviewView) {
  return update($, view, fn)
}
function say($: EngineInterface, message: string) {
  return setView($, v => ({ ...v, message }))
}

async function api($: EngineInterface, path: string, body?: object): Promise<{ status: number; json: unknown }> {
  if (!server?.endpoint) throw new Error('review server is not running')
  const { base, token } = server.endpoint
  const doc = encodeURIComponent((await read($, snapshot))?.docId ?? '')
  const res = await $.http.fetch(`${base}${path}?doc=${doc}`, {
    method: body ? 'POST' : 'GET',
    headers: { Cookie: `comments_review_token=${token}`, 'Content-Type': 'application/json' },
    body: body ? JSON.stringify(body) : undefined,
  })
  let json: unknown = null
  try {
    json = JSON.parse(res.text)
  } catch {
    json = { error: res.text.trim() }
  }
  return { status: res.status, json }
}

async function refresh($: EngineInterface) {
  const { status, json } = await api($, '/api/state')
  if (status !== 200) return say($, `refresh failed: ${errorOf(json)}`)
  const next = toSnapshot(json as Parameters<typeof toSnapshot>[0])
  const prev = await read($, snapshot)
  if (prev?.revision === next.revision) return
  await update($, snapshot, () => next)
  await setView($, v => placeCursor(v, v.cursor, next.lines.length, docRows))
}

// Sends one review action; a 409 means the agent edited the doc meanwhile,
// so the refreshed state replaces ours and the person retries knowingly.
async function act($: EngineInterface, body: Record<string, unknown>, done: string): Promise<boolean> {
  const snap = await read($, snapshot)
  if (!snap) return false
  try {
    const { status, json } = await api($, '/api/action', { ...body, revision: snap.revision })
    if (status === 409) {
      const state = (json as { state?: Parameters<typeof toSnapshot>[0] }).state
      if (state) await update($, snapshot, () => toSnapshot(state))
      await say($, 'The document changed underneath you; refreshed. Check the line and try again.')
      return false
    }
    if (status !== 200) {
      await say($, `${String(body.action)} failed: ${errorOf(json)}`)
      return false
    }
    await update($, snapshot, () => toSnapshot(json as Parameters<typeof toSnapshot>[0]))
    await say($, done)
    return true
  } catch (err) {
    await say($, `${String(body.action)} failed: ${String(err)}`)
    return false
  }
}

// What the supervisor should serve; /review-doc sets it, closing the pane clears it.
let wanted: string | null = null
let supervising = false
let wake: () => void = () => undefined
let waiters: ((ok: boolean) => void)[] = []

function settle(ok: boolean) {
  const pending = waiters
  waiters = []
  for (const resolve of pending) resolve(ok)
}

function request(doc: string | null) {
  wanted = doc
  server?.stop()
  wake()
}

// Owns `comments serve` for the session. It is started from session.start
// because a child spawned inside a command's dispatch belongs to that
// dispatch: /review-doc would never finish, and finishing would kill the server.
async function supervise($: EngineInterface) {
  supervising = true
  for (;;) {
    const doc = wanted
    if (doc === null) {
      await new Promise<void>(resolve => (wake = resolve))
      continue
    }
    await serveOnce($, doc)
  }
}

async function serveOnce($: EngineInterface, doc: string) {
  const stream = $.process.spawn({ argv: [binary, 'serve', doc] })
  const mine: Server = { endpoint: null, doc, stop: () => void stream.return?.(undefined as never) }
  server = mine
  let out = ''
  let err = ''
  try {
    for await (const chunk of stream) {
      if (chunk.stream === 'stderr') err += chunk.text
      else out += chunk.text
      if (!mine.endpoint) {
        mine.endpoint = parseServeUrl(out)
        if (mine.endpoint) settle(true)
      }
    }
  } catch (e) {
    err += String(e)
  }
  if (server === mine) server = null
  // Exited on its own rather than replaced or closed: report it and idle.
  if (wanted === doc) {
    wanted = null
    settle(false)
    // Best effort: the module may already be unloading (reload, session end).
    await setView($, v => ({ ...v, status: 'error', message: `comments serve exited: ${(err || out).trim().slice(0, 300)}` })).catch(
      () => undefined,
    )
  }
}

// Live refresh: the agent may reply or edit while the pane is open. Paused
// while the person composes, so a redraw never fights their typing.
async function poll($: EngineInterface) {
  const v = await read($, view)
  if (server?.endpoint && v.status === 'ready' && (v.mode === 'browse' || v.mode === 'verdict')) {
    await refresh($).catch(() => undefined)
  }
}

async function openReview($: EngineInterface, doc: string) {
  await update($, snapshot, () => null)
  await setView($, () => ({ ...INITIAL_VIEW, doc, status: 'starting', message: `Starting comments serve for ${doc}…` }))
  await focusPane($, doc)
  if (!supervising) {
    await setView($, v => ({ ...v, status: 'error', message: 'The review supervisor is not running; restart the session.' }))
    return
  }
  const ok = await new Promise<boolean>(resolve => {
    waiters.push(resolve)
    request(doc)
  })
  if (!ok) return
  await refresh($)
  await setView($, v => ({ ...v, status: 'ready', message: 'Esc hands the keys back to Claude Code (its rule, not ours); click the pane or run /review-doc to return.' }))
}

// The person asked for the pane, so it takes the keyboard (granted only over
// an empty composer) and opens tall enough inline to read a passage. Another
// plugin's band above the prompt would otherwise catch ctrl+x tab first.
async function focusPane($: EngineInterface, doc: string) {
  await $.ui.open({ id: PANE, title: `Review · ${doc}`, focus: true, rows: 34, columns: 100 })
}

// --- herdr: the real `comments view` beside Claude --------------------------
//
// Inside a herdr pane, /review-doc opens `comments view` in a sibling pane:
// the full TUI at full height, Esc behaving as it does there, and the verdict
// recorded by the human surface that already owns it (no token, no server).
// A `comments watch --until signoff` loop, owned by session.start for the same
// reason the server is, wakes Claude when the reviewer exits with a verdict.
// While a review is open the doc is locked against Claude's Edit/Write, and a
// review pane that closes without a verdict ends the wait instead of hanging it.

let surface: 'auto' | 'herdr' | 'pane' = 'auto'
let sessionCwd = '.'

type WatchJob = { doc: string; since: string; pane: string | null; plan: boolean; stop: () => void; closed: boolean }
let watched: WatchJob | null = null
let watchWake: () => void = () => undefined

async function herdrPane($: EngineInterface): Promise<string | null> {
  if (surface === 'pane') return null
  if ((await $.env.get('HERDR_ENV')) !== '1') return null
  return (await $.env.get('HERDR_PANE_ID')) ?? null
}

function shellQuote(s: string): string {
  return `'${s.replace(/'/g, `'\\''`)}'`
}

function absolute(path: string): string {
  return path.startsWith('/') ? path : `${sessionCwd.replace(/\/$/, '')}/${path.replace(/^\.\//, '')}`
}

// Splits Claude's herdr pane to the right and runs the TUI there; the pane
// closes itself when the reviewer quits. Never guesses a pane id: a split
// that returns none is an error, not a `pane list` diff (revdiff's rule).
async function openInHerdr($: EngineInterface, caller: string, doc: string): Promise<string> {
  const split = await $.process.run(['herdr', 'pane', 'split', '--pane', caller, '--direction', 'right', '--cwd', sessionCwd, '--focus'])
  if (split.exitCode !== 0) throw new Error((split.stderr || split.stdout).trim() || 'herdr pane split failed')
  let id: string | undefined
  try {
    id = (JSON.parse(split.stdout) as { result?: { pane?: { pane_id?: string } } }).result?.pane?.pane_id
  } catch {
    id = undefined
  }
  if (!id || id === caller) throw new Error('herdr pane split returned no usable pane id')
  const ran = await $.process.run(['herdr', 'pane', 'run', id, `${shellQuote(binary)} view ${shellQuote(doc)}; exit`])
  if (ran.exitCode !== 0) throw new Error((ran.stderr || ran.stdout).trim() || 'herdr pane run failed')
  return id
}

async function superviseWatch($: EngineInterface) {
  for (;;) {
    const job = watched
    if (job === null) {
      await new Promise<void>(resolve => (watchWake = resolve))
      continue
    }
    await watchOnce($, job)
  }
}

type Signoff = { event: string; author?: string; decision?: string; note?: string }

async function watchOnce($: EngineInterface, job: WatchJob) {
  const stream = $.process.spawn({ argv: [binary, 'watch', job.doc, '--until', 'signoff', '--since', job.since] })
  job.stop = () => void stream.return?.(undefined as never)
  let buffer = ''
  let signoff: Signoff | null = null
  try {
    for await (const chunk of stream) {
      if (chunk.stream !== 'stdout') continue
      buffer += chunk.text
      const lines = buffer.split('\n')
      buffer = lines.pop() ?? ''
      for (const line of lines) {
        try {
          const event = JSON.parse(line) as Signoff
          if (event.event === 'signoff') signoff = event
        } catch {
          // not an event line
        }
      }
    }
  } catch {
    // the watcher could not start or was stopped; handled below
  }
  if (watched !== job) return
  watched = null
  if (handoffPending === job.doc) handoffPending = null
  $.ui.status(undefined)
  if (!signoff) {
    $.ui.toast(
      job.closed
        ? `The review of ${job.doc} closed without a verdict; run /review-doc ${job.doc} to reopen it.`
        : `comments watch stopped before a verdict on ${job.doc}`,
    )
    return
  }
  const gate = await gateOf($, job.doc)
  const decision = signoff.decision ?? 'commented'
  const note = signoff.note ?? ''
  if (job.plan) return deliverPlanVerdict($, job.doc, decision, note, gate)
  await $.prompt.submit({ text: verdictPrompt(job.doc, decision, note, gate) })
}

// The pane closing is the reviewer leaving: a `comments view` quit records
// its verdict before it exits, so give the watcher a moment to see it, then
// stop waiting rather than hold the session on a review nobody can finish.
async function checkReviewPane($: EngineInterface) {
  const job = watched
  if (!job?.pane || job.closed) return
  const got = await $.process.run(['herdr', 'pane', 'get', job.pane]).catch(() => null)
  if (!got || got.exitCode === 0 || !/pane_not_found/.test(got.stdout + got.stderr)) return
  job.closed = true
  await $.clock.sleep(2500)
  if (watched === job) job.stop()
}

async function gateOf($: EngineInterface, doc: string): Promise<ReviewGate | undefined> {
  const res = await $.process.run([binary, 'gate', doc, '--json'])
  try {
    const g = JSON.parse(res.stdout) as { decision: string; summary: { blocking: number; non_blocking: number; pending_suggestions: number } }
    return { decision: g.decision, blocking: g.summary.blocking, nonBlocking: g.summary.non_blocking, pendingSuggestions: g.summary.pending_suggestions }
  } catch {
    return undefined
  }
}

async function startWatch($: EngineInterface, doc: string, pane: string | null, plan: boolean) {
  const since = new Date(await $.clock.now()).toISOString()
  watched?.stop()
  watched = { doc, since, pane, plan, stop: () => undefined, closed: false }
  watchWake()
}

async function reviewInHerdr($: EngineInterface, caller: string, doc: string, plan = false): Promise<string> {
  const pane = await openInHerdr($, caller, doc)
  await startWatch($, doc, pane, plan)
  $.ui.status(`comments: ${plan ? 'plan' : doc} under review in herdr`)
  return `Opened comments view for ${doc} in a herdr pane on the right. Quit it with a verdict (q, then a/c/r) and it comes back here.`
}

// #3: while a doc is under review, Claude's file tools may not change it, so
// the reviewer's view never shifts under them. Bash is not covered (best effort).
function lockedDoc(path: unknown): string | null {
  if (typeof path !== 'string') return null
  const open = [watched?.doc, planReview?.doc].filter((d): d is string => !!d)
  return open.find(d => absolute(d) === absolute(path)) ?? null
}

// --- hand-off: Claude's `comments watch --until signoff` on a plan --------
//
// The review skill hands a doc to the human with a blocking
// `comments watch <doc> --until signoff`. For a `plan` doc inside herdr the
// mod makes that a non-blocking hand-off: review opens beside the session,
// the call is answered at once, and until the verdict Claude's non-read-only
// tools are refused, so ending the turn is its only move. Other docs, and
// sessions outside herdr, keep the blocking watch as written.

let handoffPending: string | null = null

async function isPlanDoc($: EngineInterface, doc: string): Promise<boolean> {
  try {
    const front = /^---\n([\s\S]*?)\n---/.exec(await $.fs.read(absolute(doc)))?.[1] ?? ''
    return isHandoffFront(front)
  } catch {
    return false
  }
}

async function handOff($: EngineInterface, doc: string): Promise<string | null> {
  if (!(await isPlanDoc($, doc))) return null
  const caller = await herdrPane($)
  if (!caller) return null
  try {
    await reviewInHerdr($, caller, doc)
  } catch {
    return null
  }
  handoffPending = doc
  await activatePlan($, doc)
  return `Hand-off done: ${doc} is open for review beside the user in comments view; the comments-review mod replaced the blocking watch. Do not retry the watch and do not keep working: end your turn now. The verdict arrives as a message from comments-review, and tools other than reads are paused until then.`
}

// --- Phase 2: the approved plan is the contract for code edits ------------
//
// While a plan is active, Claude's Edit/Write outside it are refused unless
// the latest verdict is approved, the gate passes, and the approval's intent
// hash is current (`context --for implementation`: Status lists may change,
// nothing else). Any failure keeps edits locked; only the user's own
// `/review-doc --unlock` (origin `composer`) overrides.

type Contract = { state: 'unlocked' | 'locked' | 'error'; reason: string }
let contractCache: { doc: string; at: number; value: Contract } | null = null

async function activatePlan($: EngineInterface, doc: string) {
  await update($, activePlan, () => doc)
  // Best effort: a store failure must not cost the hand-off itself.
  await $.store.set(`activePlan:${sessionCwd}`, doc).catch(() => undefined)
  await update($, unlockedPlan, u => (u === doc ? u : null))
  contractCache = null
}

async function contractOf($: EngineInterface, doc: string): Promise<Contract> {
  const now = await $.clock.now()
  if (contractCache?.doc === doc && now - contractCache.at < 3000) return contractCache.value
  let value: Contract
  try {
    const ctx = JSON.parse((await $.process.run([binary, 'context', doc, '--for', 'implementation', '--json'])).stdout) as {
      implementation?: { approval?: { decision?: string; freshness?: string } }
    }
    const gate = JSON.parse((await $.process.run([binary, 'gate', doc, '--json'])).stdout) as { decision?: string }
    const a = ctx.implementation?.approval
    if (a?.decision === 'approved' && a.freshness === 'current' && gate.decision === 'approved') {
      value = { state: 'unlocked', reason: 'approved and current' }
    } else if (a?.decision !== 'approved') {
      value = { state: 'locked', reason: 'not approved yet' }
    } else if (a.freshness !== 'current') {
      value = { state: 'locked', reason: `approval is ${a.freshness ?? 'unknown'}: the plan changed since` }
    } else {
      value = { state: 'locked', reason: `gate is ${gate.decision ?? 'unknown'}` }
    }
  } catch (err) {
    value = { state: 'error', reason: `contract check failed: ${String(err).slice(0, 120)}` }
  }
  contractCache = { doc, at: now, value }
  return value
}

async function showContract($: EngineInterface) {
  const doc = await read($, activePlan)
  if (!doc && !watched) {
    const living = await read($, livingDoc)
    if (living) $.ui.status(driftStatus(living, await read($, drift)))
    return
  }
  if (!doc || watched) return
  const name = doc.replace(/^.*\//, '')
  if ((await read($, unlockedPlan)) === doc) return $.ui.status(`plan ${name}: unlocked by you`)
  const c = await contractOf($, doc)
  $.ui.status(c.state === 'unlocked' ? `plan ${name}: approved, edits open` : `plan ${name}: edits locked (${c.reason})`)
}

async function contractDeny($: EngineInterface, path: unknown, content?: unknown): Promise<string | null> {
  const doc = await read($, activePlan)
  if (!doc) return null
  if (typeof path === 'string' && absolute(path) === absolute(doc)) return null
  // A living doc is markdown the agent keeps current; it is never code.
  if (typeof path === 'string' && (await livingTarget($, absolute(path), content))) return null
  if ((await read($, unlockedPlan)) === doc) return null
  const c = await contractOf($, doc)
  if (c.state === 'unlocked') return null
  return `Code edits wait for the plan contract: ${doc} (${c.reason}). Revise the plan and hand it off with \`comments watch ${doc} --until signoff\`, then end your turn. Only the user can override with /review-doc --unlock.`
}

// --- Phase 3: the contract survives compaction and resume ------------------
//
// The note carries only gate-derived state (plan path, gate, lock, blocking
// count), never thread text or plan prose, so a summary cannot re-inject a
// reviewer's words as instructions (adversarial-review-loop's rule).

const CONTRACT_MARK = '[comments contract]'

async function contractNote($: EngineInterface): Promise<string | null> {
  const doc = await read($, activePlan)
  if (!doc) return null
  const c = (await read($, unlockedPlan)) === doc ? { state: 'unlocked', reason: 'unlocked by the user' } : await contractOf($, doc)
  const gate = await gateOf($, doc)
  const edits = c.state === 'unlocked' ? `open (${c.reason})` : `locked (${c.reason})`
  const blocking = gate ? `${gate.decision}, ${gate.blocking} blocking` : 'unknown'
  const picks = await openPicks($, doc).catch(() => null)
  const pickLine = picks === null ? '' : ` ${picks} open pick${picks === 1 ? '' : 's'} for the end review.`
  return `${CONTRACT_MARK} Active plan: ${doc}. Gate: ${blocking}. Code edits: ${edits}.${pickLine} Implement only what the plan says; decide alone where you can and file a pick (\`comments add --pick\`) instead of asking; run \`comments context ${doc} --for implementation\` for phases and status.`
}

// With no plan active, the living doc's note stands in for the contract's.
async function sessionNote($: EngineInterface): Promise<string | null> {
  const note = await contractNote($).catch(() => null)
  if (note) return note
  const living = await read($, livingDoc)
  return living ? livingNote(living, await read($, drift)) : null
}

// --- living docs: the doc and the build move together ----------------------
//
// Writing a doc whose frontmatter says `template: living` makes it the
// session's living doc; every other file edit counts as drift until the doc
// changes again. The count is a nudge in the status line and the session
// note, never a lock.

async function livingTarget($: EngineInterface, path: string, content: unknown): Promise<boolean> {
  if (!/\.md$/.test(path)) return false
  if (typeof content === 'string') return isLivingFront(frontOf(content))
  try {
    return isLivingFront(frontOf(await $.fs.read(path)))
  } catch {
    return false
  }
}

async function trackEdit($: EngineInterface, path: unknown, content: unknown) {
  if (typeof path !== 'string') return
  const abs = absolute(path)
  if (await livingTarget($, abs, content)) {
    await update($, livingDoc, () => abs)
    await update($, drift, () => 0)
    await $.store.set(`livingDoc:${sessionCwd}`, abs).catch(() => undefined)
  } else if (await read($, livingDoc)) {
    await update($, drift, n => n + 1)
  } else {
    return
  }
  // The count outlives a restart, so a resumed session never reads "current"
  // for a doc the build has run ahead of.
  await $.store.set(`drift:${sessionCwd}`, String(await read($, drift))).catch(() => undefined)
  await showContract($).catch(() => undefined)
}

async function restoreLivingDoc($: EngineInterface) {
  if (await read($, livingDoc)) return
  const saved = await $.store.get(`livingDoc:${sessionCwd}`)
  if (typeof saved !== 'string' || saved === '') return
  await update($, livingDoc, () => saved)
  const count = Number(await $.store.get(`drift:${sessionCwd}`))
  await update($, drift, () => (Number.isFinite(count) && count > 0 ? count : 0))
}

async function clearLivingDoc($: EngineInterface): Promise<string | null> {
  const doc = await read($, livingDoc)
  await update($, livingDoc, () => null)
  await update($, drift, () => 0)
  await $.store.set(`livingDoc:${sessionCwd}`, '').catch(() => undefined)
  await $.store.set(`drift:${sessionCwd}`, '0').catch(() => undefined)
  $.ui.status('')
  return doc
}

// Picks the agent filed and the human has not settled yet.
async function openPicks($: EngineInterface, doc: string): Promise<number> {
  const inbox = JSON.parse((await $.process.run([binary, 'inbox', doc, '--json'])).stdout) as { items?: { thread?: { pick?: string } }[] }
  return (inbox.items ?? []).filter(i => !!i.thread?.pick).length
}

// True the first time it is asked in a session, false after, reloads included.
async function claimReminder($: EngineInterface): Promise<boolean> {
  if (await read($, reminded)) return false
  await update($, reminded, () => true)
  return true
}

async function restoreActivePlan($: EngineInterface) {
  if (await read($, activePlan)) return
  const saved = await $.store.get(`activePlan:${sessionCwd}`)
  if (typeof saved === 'string' && saved !== '') await update($, activePlan, () => saved)
}

// --- plan mode → comments review (Plannotator's loop, comments as surface) --
//
// Claude's ExitPlanMode is denied while the plan is saved as a comments doc
// and opened for review; the verdict arrives as a plugin turn. A revised plan
// (ExitPlanMode again) rewrites the same doc, so threads re-anchor and carry
// over between rounds. After approval with a passing gate, Claude's next
// ExitPlanMode is allowed with the reviewed doc as the plan, accepted
// suggestions included. Subagents keep Claude Code's own flow.

type PlanReview = { doc: string; version: number; status: 'reviewing' | 'approved'; passing: string | null }
let planReview: PlanReview | null = null

function planSlug(plan: string): string {
  const heading = /^#\s+(.+)$/m.exec(plan)?.[1] ?? 'plan'
  return heading.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '').slice(0, 48) || 'plan'
}

async function planText($: EngineInterface, e: { plan?: unknown; planFilePath?: unknown }): Promise<string> {
  const inline = typeof e.plan === 'string' ? e.plan : ''
  const path = e.planFilePath
  if (typeof path === 'string' && path.startsWith('/') && /\.md$/i.test(path)) {
    try {
      return (await $.fs.read(path)) || inline
    } catch {
      return inline
    }
  }
  return inline
}

async function writePlanDoc($: EngineInterface, plan: string, doc?: string): Promise<string> {
  const target = doc ?? `docs/artifacts/plans/${new Date(await $.clock.now()).toISOString().slice(0, 10)}-${planSlug(plan)}.md`
  await $.process.run(['mkdir', '-p', absolute(target).replace(/\/[^/]+$/, '')])
  await $.fs.write(absolute(target), plan.endsWith('\n') ? plan : `${plan}\n`)
  return target
}

async function onPlanCall($: EngineInterface, toolUseId: string, input: { plan?: unknown; planFilePath?: unknown }): Promise<string | null> {
  if (planReview?.status === 'approved') {
    planReview.passing = toolUseId
    return null
  }
  const plan = await planText($, input)
  if (plan.trim() === '') return null
  const caller = await herdrPane($)
  if (planReview) {
    planReview.version += 1
    await writePlanDoc($, plan, planReview.doc)
    // `comments view` reloads the file only before a write, so an open pane
    // would show the old text: close it and reopen on the rewritten doc.
    // Submitted comments are already saved; only an unsent draft is lost.
    const open = watched?.doc === planReview.doc ? watched : null
    if (open?.pane) {
      open.closed = true
      open.stop()
      await $.process.run(['herdr', 'pane', 'close', open.pane]).catch(() => null)
    }
    const reopened = await reopenPlanReview($, caller, planReview.doc)
    return `Plan v${planReview.version} rewrote ${planReview.doc}; its review threads carried over.${reopened} It is NOT approved. Stay in plan mode and end your turn; the verdict arrives as a message from comments-review.`
  }
  const doc = await writePlanDoc($, plan)
  await activatePlan($, doc)
  planReview = { doc, version: 1, status: 'reviewing', passing: null }
  const opened = await reopenPlanReview($, caller, doc)
  return `Plan v1 saved to ${doc}.${opened} It is NOT approved. Stay in plan mode and do not implement; end your turn. The verdict arrives as a message from comments-review; to revise, call ExitPlanMode again.`
}

async function reopenPlanReview($: EngineInterface, caller: string | null, doc: string): Promise<string> {
  if (caller) {
    try {
      await reviewInHerdr($, caller, doc, true)
      return ' It is open for review in comments view beside you.'
    } catch (err) {
      await startWatch($, doc, null, true)
      return ` The herdr pane did not open (${String(err)}); the user reviews it with \`comments view ${doc}\`.`
    }
  }
  await startWatch($, doc, null, true)
  $.ui.status('comments: plan waiting for review')
  return ` The user reviews it with \`comments view ${doc}\` (or /review-doc ${doc}).`
}

async function deliverPlanVerdict($: EngineInterface, doc: string, decision: string, note: string, gate: ReviewGate | undefined) {
  const said = note.trim() === '' ? '' : ` Reviewer note: "${note.trim()}".`
  if (planReview?.doc === doc && decision === 'approved' && gate?.decision === 'approved') {
    planReview.status = 'approved'
    await $.prompt.submit({
      text: `The plan in ${doc} was approved in comments review and its gate passes.${said} Run \`comments inbox ${doc} --json\` for any last replies, then call ExitPlanMode again; the reviewed doc (accepted suggestions included) becomes the plan you implement.`,
    })
    return
  }
  const gateLine = gate ? ` The gate is ${gate.decision} with ${gate.blocking} blocking thread(s) and ${gate.pendingSuggestions} pending suggestion(s).` : ''
  await $.prompt.submit({
    text: `Plan review of ${doc}: ${decision.replace('_', ' ')}.${gateLine}${said} Stay in plan mode. Run \`comments inbox ${doc} --json\`, reply on each thread with \`comments reply\` (do not resolve the human's threads), then call ExitPlanMode with the revised plan; it rewrites ${doc} and the threads carry over.`,
  })
}


export const register: Register = (on, options) => {
  binary = typeof options.binary === 'string' && options.binary !== '' ? options.binary : 'comments'
  surface = options.surface === 'herdr' || options.surface === 'pane' ? options.surface : 'auto'

  on('session.start', async ($, e, next) => {
    await $.command.register({
      name: 'review-doc',
      description: 'Review a comments doc in a pane (comment, reply, resolve, suggestions, verdict)',
      argumentHint: '<doc.md> | --unlock | --park | --done',
    })
    sessionCwd = e.cwd
    await restoreActivePlan($).catch(() => undefined)
    await restoreLivingDoc($).catch(() => undefined)
    void supervise($)
    void superviseWatch($)
    // Live refresh: the agent may reply or edit while the pane is open.
    $.clock.every(2000, () => void poll($))
    $.clock.every(3000, () => void checkReviewPane($))
    $.clock.every(5000, () => void showContract($).catch(() => undefined))
    // A reload drops the old module's child; reopen against the doc on record.
    const v = await read($, view)
    if (v.doc !== '' && v.status !== 'idle') void openReview($, v.doc)
    const started = await next(e)
    // On a resumed or restarted session, remind Claude of the contract once.
    const note = await sessionNote($).catch(() => null)
    if (note && (await claimReminder($).catch(() => false))) await $.session.append({ message: { type: 'user', content: [{ type: 'text', text: note }] } }).catch(() => undefined)
    return started
  })

  // #1: plan mode hands its plan to comments review instead of the dialog.
  on('tool.call', { tool: 'ExitPlanMode' }, async ($, e, next) => {
    const call = e as unknown as { tool_use_id: string; agentId?: string; plan?: unknown; planFilePath?: unknown }
    if (call.agentId) return next(e)
    const deny = await onPlanCall($, call.tool_use_id, call)
    return deny === null ? next(e) : { deny }
  }).catch(($, e, next) => next(e)) // fail open: Claude Code's own plan dialog

  // The approved plan's second ExitPlanMode: allow it with the reviewed doc as
  // the plan. Answered without next(), so no other review opens for it.
  on('classic.PermissionRequest', async ($, e, next) => {
    const req = e as unknown as { tool_name?: string; agent_id?: string; tool_use_id?: string; tool_input?: Record<string, unknown> }
    const review = planReview
    if (req.tool_name !== 'ExitPlanMode' || req.agent_id || review?.status !== 'approved' || review.passing !== req.tool_use_id) return next(e)
    const plan = await $.fs.read(absolute(review.doc))
    planReview = null
    if (watched?.doc === review.doc) watched.stop()
    $.ui.status(undefined)
    return { decision: { behavior: 'allow', updatedInput: { ...(req.tool_input ?? {}), plan } } } as never
  }).catch(($, e, next) => next(e))

  // Phase 1: a plan's blocking hand-off becomes a non-blocking review, and
  // until the verdict only reads go through. Subagents are left alone.
  on('tool.call', async ($, e, next) => {
    const call = e as unknown as { tool: string; command?: unknown; agentId?: string }
    if (call.agentId) return next(e)
    if (call.tool === 'Bash' && typeof call.command === 'string') {
      const doc = watchTarget(call.command)
      if (doc) {
        const handed = await handOff($, doc)
        if (handed) return { deny: handed }
      }
    }
    if (handoffPending && !allowedDuringReview(call.tool, call.command)) {
      return { deny: `${handoffPending} is waiting for the user's review, so only reads run until the verdict. End your turn; the verdict arrives as a message from comments-review.` }
    }
    return next(e)
  }).catch(($, e, next) => next(e))

  // Phase 2: code edits outside the active plan wait for its approval. Fails
  // closed: a broken check keeps edits locked.
  on('tool.call', { tool: ['Edit', 'Write', 'NotebookEdit'] }, async ($, e, next) => {
    const args = e as unknown as { file_path?: unknown; notebook_path?: unknown; content?: unknown }
    const deny = await contractDeny($, args.file_path ?? args.notebook_path, args.content)
    return deny === null ? next(e) : { deny }
  }).catch(() => ({ deny: 'The plan contract check failed, so code edits stay locked. Only the user can override with /review-doc --unlock.' }))

  // Living docs: track which file changed after the edit lands. Main session only.
  on('tool.call', { tool: ['Edit', 'Write', 'NotebookEdit'] }, async ($, e, next) => {
    const args = e as unknown as { file_path?: unknown; notebook_path?: unknown; content?: unknown; agentId?: string }
    const result = await next(e)
    if (!args.agentId && !JSON.stringify(result).includes('"deny"')) await trackEdit($, args.file_path ?? args.notebook_path, args.content).catch(() => undefined)
    return result
  }).catch(($, e, next) => next(e))

  // #3: no Edit/Write to a doc while it is under review.
  on('tool.call', { tool: ['Edit', 'Write', 'NotebookEdit'] }, async ($, e, next) => {
    const args = e as unknown as { file_path?: unknown; notebook_path?: unknown }
    const doc = lockedDoc(args.file_path ?? args.notebook_path)
    if (!doc) return next(e)
    return {
      deny: `${doc} is under human review in comments view, so it is locked until the verdict arrives. Reply on threads with \`comments reply\`${planReview?.doc === doc ? ', or revise the plan by calling ExitPlanMode again' : ''}.`,
    }
  }).catch(($, e, next) => next(e))

  // Phase 3: compaction keeps exactly one fresh contract note.
  on('session.compact', async ($, e, next) => {
    if ((e as unknown as { agentId?: string }).agentId !== undefined) return next(e)
    const note = await sessionNote($).catch(() => null)
    if (!note) return next(e)
    const result = await next(e)
    if (result.skip !== undefined) return result
    const kept = result.messages.filter(m => !(m.text ?? '').startsWith(CONTRACT_MARK) && !(m.text ?? '').startsWith(LIVING_MARK))
    return { ...result, messages: [...kept, { role: 'user' as const, text: note, toolUses: [] }] }
  }).catch(($, e, next) => next(e))

  on('session.end', async ($, e, next) => {
    request(null)
    return next(e)
  })

  on('command.run', { command: 'review-doc' }, async ($, e) => {
    // `--unlock` opens the active plan's gate by hand; only the person's own
    // Enter (an engine-stamped `composer` origin) may run it.
    if (/(^|\s)--unlock(\s|$)/.test(e.args)) {
      if (e.origin?.kind !== 'composer') return { text: '/review-doc --unlock runs only from your own Enter at the prompt.' }
      const doc = await read($, activePlan)
      if (!doc) return { text: 'No plan is active, so there is nothing to unlock.' }
      await update($, unlockedPlan, () => doc)
      await showContract($)
      return { text: `Unlocked code edits for ${doc} by hand. The contract check is bypassed until another plan is handed off.` }
    }
    // `--park` sets the active plan aside: no lock, no contract note, until a
    // plan is handed off again. The person's own Enter only, like --unlock.
    if (/(^|\s)--park(\s|$)/.test(e.args)) {
      if (e.origin?.kind !== 'composer') return { text: '/review-doc --park runs only from your own Enter at the prompt.' }
      const doc = await read($, activePlan)
      if (!doc) return { text: 'No plan is active, so there is nothing to park.' }
      await update($, activePlan, () => null)
      await update($, unlockedPlan, () => null)
      await $.store.set(`activePlan:${sessionCwd}`, '').catch(() => undefined)
      contractCache = null
      await showContract($)
      return { text: `Parked ${doc}: code edits are no longer gated on it. Handing a plan off makes it active again.` }
    }
    // `--done` ends the living doc for this repo: no note, no drift count,
    // until a living doc is written again.
    if (/(^|\s)--done(\s|$)/.test(e.args)) {
      const doc = await clearLivingDoc($)
      return { text: doc ? `Done with ${doc}: no living doc is tracked now.` : 'No living doc is tracked.' }
    }
    // `--pane` forces the in-Claude pane even inside herdr.
    const forcePane = /(^|\s)--pane(\s|$)/.test(e.args)
    const doc = e.args.replace(/(^|\s)--pane(\s|$)/, ' ').trim()
    const caller = forcePane || doc === '' ? null : await herdrPane($)
    if (caller) {
      try {
        return { text: await reviewInHerdr($, caller, doc) }
      } catch (err) {
        if (surface === 'herdr') return { text: `Could not open a herdr pane: ${String(err)}` }
        // auto: fall through to the in-Claude pane
      }
    }
    if (doc === '') {
      const open = await read($, view)
      if (open.doc === '' || open.status === 'idle') {
        return { text: 'Usage: /review-doc <doc.md> — a markdown file (a sidecar is created if needed).' }
      }
      await focusPane($, open.doc)
      return { text: `Back in the review pane for ${open.doc}.` }
    }
    await openReview($, doc)
    const v = await read($, view)
    // Only the fullscreen layout docks a pane beside the transcript (from 110
    // columns); the main screen always seats it inline, capped in height.
    const where = e.presentation?.isFullscreen
      ? e.presentation.columns < 110 ? ' It is inline: the side dock needs a terminal 110+ columns wide.' : ''
      : ' It opened inline because this session uses the default layout; run /tui fullscreen to dock it on the side at full height.'
    return { text: v.status === 'error' ? v.message : `Opened the review pane for ${doc}.${where}` }
  })

  on('ui.close', { id: PANE }, async ($, e, next) => {
    request(null)
    await setView($, () => INITIAL_VIEW)
    return next(e)
  }).catch(($, e, next) => next(e))

  on('ui.render', { component: 'Pane', requestId: PANE }, async ($, e) => {
    if (e.surface === 'mobile') {
      const { Text } = $.ui.resolve(e)
      return <Text>The review pane needs a surface with text input; use comments serve on mobile.</Text>
    }
    const { Box, Text, Button, Input } = $.ui.resolve(e)
    const v = await read($, view)
    const snap = await read($, snapshot)
    const width = Math.max(30, e.props.bodyColumns)
    const rows = e.props.scroll?.bodyRows ?? 30

    // Zoom keeps every hotkey (a hotkey only works while its Button is drawn)
    // but shrinks each to a glyph, so the document gets nearly every row.
    const btn = (hotkey: string, label: string, onPress: () => unknown, primary = false, glyph = '') => (
      <Button key={`k-${hotkey}`} plain hotkey={hotkey} variant={primary ? 'primary' : undefined} onPress={() => onPress()}>
        {v.zoom ? glyph : label}
      </Button>
    )

    if (!snap) {
      return (
        <Box flexDirection="column">
          <Text>{v.message || 'Run /review-doc <doc.md> to open a document.'}</Text>
        </Box>
      )
    }

    const lineCount = snap.lines.length
    const here = threadsAt(snap, v.cursor, v.showResolved)
    const thread: ReviewThread | undefined = here[Math.min(v.threadIndex, Math.max(0, here.length - 1))]
    const open = snap.threads.filter(isOpen).length
    const last = snap.reviews[snap.reviews.length - 1]
    const gutter = String(lineCount).length

    // Every row not spent on chrome goes to the document: the thread panel is
    // only as tall as the thread under the cursor, the message row appears
    // only with a message, and zoom folds both to a single line.
    const composing = v.mode === 'comment' || v.mode === 'blocking' || v.mode === 'reply' || v.mode === 'note'
    const threadBody = thread ? (thread.isSuggestion ? 3 : 2) + thread.replies.length : 1
    const threadRows = v.zoom && !composing ? 1 : Math.min(THREAD_ROWS, threadBody)
    const messageRows = v.message !== '' && !v.zoom ? 1 : 0
    const controlRows = v.zoom && !composing ? 1 : 2
    docRows = Math.max(3, rows - 1 - threadRows - messageRows - controlRows)

    const move = (delta: number) => setView($, x => moveCursor(x, delta, lineCount, docRows))
    const jump = (line: number) => setView($, x => placeCursor(x, line, lineCount, docRows))
    const mode = (m: ReviewMode, message = '') => setView($, x => ({ ...x, mode: m, message }))

    const header = (
      <Box flexDirection="row" justifyContent="space-between">
        <Text bold wrap="truncate-end">
          {snap.name}
          {e.props.isFocused ? '' : <Text dimColor> (not focused: /review-doc)</Text>}
        </Text>
        <Text color={snap.gate.decision === 'approved' ? 'green' : 'yellow'}>
          {snap.gate.decision.replace('_', ' ')} · {snap.gate.blocking} blocking · {open} open · {snap.gate.pendingSuggestions} sugg
        </Text>
      </Box>
    )

    const docWindow = (
      <Box flexDirection="column" height={docRows}>
        {snap.lines.slice(v.top - 1, v.top - 1 + docRows).map((text, i) => {
          const n = v.top + i
          const mark = markerFor(snap, n, v.showResolved)
          const isCursor = n === v.cursor
          return (
            <Text key={`l-${n}`} inverse={isCursor} wrap="truncate-end">
              <Text color={mark === '!' ? 'red' : mark === '±' ? 'cyan' : 'yellow'}>{mark}</Text>
              <Text dimColor={!isCursor}> {String(n).padStart(gutter)} </Text>
              {text === '' ? ' ' : text.replace(/\t/g, '  ')}
            </Text>
          )
        })}
      </Box>
    )

    const threadPanel = threadRows === 1 && thread ? (
      <Text wrap="truncate-end">
        <Text bold color={thread.blocking ? 'red' : undefined}>{thread.id}</Text> {thread.author}: {thread.isSuggestion ? `± ${thread.proposedText.split('\n')[0]}` : thread.text}
        {thread.replies.length ? ` (+${thread.replies.length})` : ''}
      </Text>
    ) : (
      <Box flexDirection="column" height={threadRows}>
        {thread ? (
          <Box flexDirection="column">
            <Text wrap="truncate-end" bold>
              {thread.blocking ? 'BLOCKING ' : ''}
              {thread.isSuggestion ? 'Suggestion' : 'Thread'} {thread.id} · {thread.author}
              {thread.resolved ? ' · resolved' : ''}
              {here.length > 1 ? ` · ${v.threadIndex + 1}/${here.length} (t cycles)` : ''}
            </Text>
            {thread.isSuggestion ? (
              <>
                <Text wrap="truncate-end" color="red">- {thread.originalText.split('\n')[0]}</Text>
                <Text wrap="truncate-end" color="green">+ {thread.proposedText.split('\n')[0]}</Text>
              </>
            ) : (
              <Text wrap="truncate-end">{thread.text}</Text>
            )}
            {thread.replies.slice(-Math.max(0, threadRows - (thread.isSuggestion ? 3 : 2))).map(r => (
              <Text key={`r-${r.id}`} wrap="truncate-end" dimColor>
                ↳ {r.author}: {r.text}
              </Text>
            ))}
          </Box>
        ) : (
          <Text dimColor wrap="truncate-end">
            No thread on line {v.cursor}.{last ? ` Last verdict: ${last.decision} by ${last.author}.` : ''}
          </Text>
        )}
      </Box>
    )

    let controls
    if (v.mode === 'comment' || v.mode === 'blocking' || v.mode === 'reply' || v.mode === 'note') {
      const label =
        v.mode === 'reply' ? `Reply to ${thread?.id ?? '?'}: ` :
        v.mode === 'note' ? 'Verdict note: ' :
        `${v.mode === 'blocking' ? 'Blocking comment' : 'Comment'} on L${v.cursor}: `
      // Esc belongs to Claude Code (it always hands the keys back to the
      // prompt and never reaches a plugin), so an empty Enter is the cancel.
      const submit = async (text: string) => {
        if (v.mode === 'note') {
          await setView($, x => ({ ...x, note: text, mode: 'verdict', message: text.trim() ? 'Note saved; pick a decision.' : '' }))
          return
        }
        if (text.trim() === '') return mode('browse', 'Cancelled.')
        const ok =
          v.mode === 'reply'
            ? await act($, { action: 'reply', thread_id: thread?.id ?? '', text }, `Replied to ${thread?.id}.`)
            : await act($, { action: 'add', line: v.cursor, text, blocking: v.mode === 'blocking' }, `Added a comment on line ${v.cursor}.`)
        if (ok) await mode('browse', (await read($, view)).message)
      }
      controls = (
        <Box flexDirection="column">
          <Input key="compose" label={label} autoFocus submitLabel="send" value={v.mode === 'note' ? v.note : ''} onSubmit={text => submit(text)} />
          <Box flexDirection="row" gap={2}>
            <Button key="cancel" plain onPress={() => mode(v.mode === 'note' ? 'verdict' : 'browse')}>Cancel</Button>
            <Text dimColor>Enter sends · empty Enter cancels · Esc leaves the pane (text kept; click or /review-doc to return)</Text>
          </Box>
        </Box>
      )
    } else if (v.mode === 'verdict') {
      const record = async (decision: string) => {
        const note = v.note
        const ok = await act($, { action: 'verdict', decision, note }, `Recorded ${decision.replace('_', ' ')}.`)
        if (!ok) return
        await setView($, x => ({ ...x, mode: 'browse', note: '' }))
        // Hand the turn back to Claude: the verdict is the envelope, the
        // threads are the payload (review-comments skill, inbox first).
        const gate = (await read($, snapshot))?.gate
        await $.prompt.submit({ text: verdictPrompt(v.doc, decision, note, gate) })
      }
      controls = (
        <Box flexDirection="column">
          <Text>Submit review{v.note ? ` — note: "${v.note}"` : ''}</Text>
          <Box flexDirection="row" flexWrap="wrap" columnGap={2}>
            {btn('a', 'approve', () => record('approved'), true)}
            {btn('c', 'request changes', () => record('changes_requested'))}
            {btn('r', 'replies only', () => record('commented'))}
            {btn('n', 'note', () => mode('note'))}
            {btn('q', 'back', () => mode('browse'))}
          </Box>
        </Box>
      )
    } else {
      const needThread = (f: (t: ReviewThread) => unknown) => () => (thread ? f(thread) : say($, `No thread on line ${v.cursor}.`))
      const nav = [
        btn('j', 'down', () => move(1), false, '↓'),
        btn('k', 'up', () => move(-1), false, '↑'),
        btn('d', 'pg dn', () => move(Math.max(1, Math.floor(docRows / 2))), false, '⇟'),
        btn('u', 'pg up', () => move(-Math.max(1, Math.floor(docRows / 2))), false, '⇞'),
        btn('g', 'top', () => jump(1), false, '⤒'),
        btn('e', 'end', () => jump(lineCount), false, '⤓'),
        btn('n', 'next thread', () => {
          const l = nextThreadLine(snap, v.cursor, 1, v.showResolved)
          return l === null ? say($, 'No threads.') : jump(l)
        }, false, '▸'),
        btn('p', 'prev thread', () => {
          const l = nextThreadLine(snap, v.cursor, -1, v.showResolved)
          return l === null ? say($, 'No threads.') : jump(l)
        }, false, '◂'),
        btn('t', 'cycle', () => setView($, x => ({ ...x, threadIndex: here.length ? (x.threadIndex + 1) % here.length : 0 })), false, '↻'),
      ]
      const edit = [
        btn('c', 'comment', () => mode('comment'), false, '+'),
        btn('b', 'blocking', () => mode('blocking'), false, '!'),
        btn('r', 'reply', needThread(() => mode('reply')), false, '↩'),
        btn('s', thread?.resolved ? 'reopen' : 'resolve', needThread(t =>
          act($, { action: t.resolved ? 'reopen' : 'resolve', thread_id: t.id }, `${t.resolved ? 'Reopened' : 'Resolved'} ${t.id}.`),
        ), false, '✓'),
        btn('y', 'accept', needThread(t =>
          t.isSuggestion ? act($, { action: 'accept', thread_id: t.id }, `Accepted ${t.id}; the document was edited.`) : say($, 'Not a suggestion.'),
        ), false, '±'),
        btn('x', 'reject', needThread(t =>
          t.isSuggestion ? act($, { action: 'reject', thread_id: t.id }, `Rejected ${t.id}.`) : say($, 'Not a suggestion.'),
        ), false, '✗'),
        btn('h', v.showResolved ? 'hide resolved' : 'show resolved', () => setView($, x => ({ ...x, showResolved: !x.showResolved })), false, '◌'),
        btn('z', v.zoom ? 'unzoom' : 'zoom', () => setView($, x => ({ ...x, zoom: !x.zoom })), false, '⊡'),
        btn('v', 'submit review', () => mode('verdict'), true, 'review'),
      ]
      controls = v.zoom ? (
        <Box flexDirection="row" columnGap={1} overflow="hidden">
          {[...nav, ...edit]}
        </Box>
      ) : (
        <Box flexDirection="column">
          <Box flexDirection="row" flexWrap="wrap" columnGap={2}>
            {nav}
          </Box>
          <Box flexDirection="row" flexWrap="wrap" columnGap={2}>
            {edit}
          </Box>
        </Box>
      )
    }

    return (
      <Box flexDirection="column" width={width}>
        {header}
        {docWindow}
        {threadPanel}
        {messageRows > 0 && (
          <Text color={v.status === 'error' ? 'red' : undefined} dimColor={v.status !== 'error'} wrap="truncate-end">
            {v.message}
          </Text>
        )}
        {controls}
      </Box>
    )
  })
}

function errorOf(json: unknown): string {
  if (json && typeof json === 'object' && 'error' in json) return String((json as { error: unknown }).error)
  return JSON.stringify(json).slice(0, 200)
}
