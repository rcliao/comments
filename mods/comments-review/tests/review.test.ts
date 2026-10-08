import { expect, mock, test } from 'claude-code/testing'

import { allowedDuringReview, driftStatus, isHandoffFront, isLivingFront, livingNote, markerFor, nextThreadLine, parseServeUrl, placeCursor, toSnapshot, watchTarget } from '../hooks/review'
import { INITIAL_VIEW } from '../hooks/review'

type Thread = Record<string, unknown>

// A stand-in for `comments serve`: the spawn prints the bootstrap URL and
// stays up; /api/state and /api/action answer from an in-memory document.
function fakeServe() {
  const lines = ['# Plan', '', 'Step one', 'Step two']
  const threads: Thread[] = [
    { id: 'c1', author: 'claude', line: 3, text: 'Is step one needed?', blocking: false, resolved: false, replies: [] },
  ]
  const reviews: { author: string; decision: string; note?: string }[] = []
  const actions: Record<string, unknown>[] = []
  const cookies: string[] = []
  let rev = 1
  const state = () => ({
    doc_id: 'plan.md',
    name: 'plan.md',
    author: 'eric',
    revision: `r${rev}`,
    lines,
    document: { threads, reviews },
    gate: {
      decision: threads.some(t => t.blocking && !t.resolved) ? 'changes_requested' : 'approved',
      blocking: threads.filter(t => t.blocking && !t.resolved).length,
      non_blocking: threads.filter(t => !t.blocking && !t.resolved).length,
      pending_suggestions: 0,
    },
  })
  const respond = (status: number, body: unknown) => ({ status, ok: status < 300, headers: {}, text: JSON.stringify(body) })
  const fetch = (url: string, body: string | undefined, cookie: string | undefined) => {
    cookies.push(cookie ?? '')
    if (url.includes('/api/state')) return respond(200, state())
    const req = JSON.parse(body ?? '{}') as Record<string, unknown>
    actions.push(req)
    if (req.revision !== `r${rev}`) return respond(409, { error: 'document changed', state: state() })
    if (req.action === 'add') {
      threads.push({ id: `c${threads.length + 1}`, author: 'eric', line: req.line, text: req.text, blocking: req.blocking, resolved: false, replies: [] })
    } else if (req.action === 'reply') {
      const t = threads.find(x => x.id === req.thread_id)
      ;(t?.replies as unknown[]).push({ id: 'r1', author: 'eric', text: req.text })
    } else if (req.action === 'resolve') {
      const t = threads.find(x => x.id === req.thread_id)
      if (t) t.resolved = true
    } else if (req.action === 'verdict') {
      reviews.push({ author: 'eric', decision: String(req.decision), note: String(req.note ?? '') })
    }
    rev++
    return respond(200, state())
  }
  return { fetch, actions, cookies, reviews }
}

const PANE_PROPS = {
  title: 'Review · plan.md',
  isFocused: true,
  bodyColumns: 90,
  placement: 'dock',
  scroll: { offset: 0, bodyRows: 30 },
  view: {},
} as const

test('parses the bootstrap URL comments serve prints', () => {
  expect(parseServeUrl('Comments review is ready for x\nOpen: http://127.0.0.1:5123/?token=abc123\n')).toEqual({
    base: 'http://127.0.0.1:5123',
    token: 'abc123',
  })
  expect(parseServeUrl('Comments review is ready')).toBe(null)
})

test('cursor clamps and scrolls; thread navigation wraps', () => {
  const snap = toSnapshot({
    doc_id: 'd', name: 'd', author: 'a', revision: 'r', lines: ['a', 'b', 'c', 'd', 'e'],
    document: { threads: [
      { id: 'x', author: 'a', line: 2, text: 't', blocking: true, resolved: false },
      { id: 'y', author: 'a', line: 4, text: 't', blocking: false, resolved: true },
    ] },
    gate: { decision: 'changes_requested', blocking: 1, non_blocking: 0, pending_suggestions: 0 },
  })
  expect(placeCursor(INITIAL_VIEW, 99, 5, 2)).toMatchObject({ cursor: 5, top: 4 })
  expect(markerFor(snap, 2, false)).toBe('!')
  expect(markerFor(snap, 4, false)).toBe(' ')
  expect(markerFor(snap, 4, true)).toBe('✓')
  expect(nextThreadLine(snap, 2, 1, false)).toBe(2)
  expect(nextThreadLine(snap, 2, 1, true)).toBe(4)
})

test('the pane comments, resolves and records a verdict through comments serve', async ($, on) => {
  const serve = fakeServe()
  const prompts: string[] = []
  // The fake server lives until the test lets it exit, as `comments serve`
  // lives until the pane closes; command.run settles with it.
  let exit = () => undefined as void
  const exited = new Promise<void>(r => (exit = r))
  let ready = () => undefined as void
  const served = new Promise<void>(r => (ready = r))
  on('process.spawn', async function* (_$, e) {
    expect(e.argv.slice(1)).toEqual(['serve', 'plan.md'])
    yield { stream: 'stdout' as const, text: 'Open: http://127.0.0.1:5123/?token=feed\n' }
    await exited
    return { value: { code: 0, signal: null } }
  })
  on('http.fetch', async (_$, e) => {
    const res = serve.fetch(e.url, e.init?.body, e.init?.headers?.Cookie)
    ready()
    return { value: res }
  })
  on('ui.open', async () => ({ value: { isPlaced: true as const } }))
  on('prompt.submit', async (_$, e) => {
    prompts.push(e.text)
    return e as never
  })

  mock.clock(on)
  mock.env(on, {})
  on('session.start', async (_$, e) => e as never)
  on('command.register', async () => ({ value: undefined }) as never)
  void $.session.start({ cwd: '/repo', surface: 'terminal', isInteractive: true } as Parameters<typeof $.session.start>[0])

  // /review-doc settles once the server is up; the server outlives it.
  const ran = await $.command.run({ command: 'review-doc', args: 'plan.md' } as Parameters<typeof $.command.run>[0])
  expect(ran.text).toContain('Opened the review pane for plan.md.')
  await served
  expect(serve.cookies[0]).toBe('comments_review_token=feed')

  for (const surface of ['terminal', 'desktop'] as const) {
    const ui = await $.ui.mount({ plugin: 'comments-review', surface, component: 'Pane', requestId: 'comments-review', props: PANE_PROPS })
    expect(await ui.find({ type: 'Text', text: /Step one/ })).toBeDefined()
    await ui.unmount()
  }

  // Zoom keeps every hotkey drawn, as glyphs, on both surfaces.
  for (const surface of ['terminal', 'desktop'] as const) {
    const ui = await $.ui.mount({ plugin: 'comments-review', surface, component: 'Pane', requestId: 'comments-review', props: PANE_PROPS })
    await ui.press({ key: 'k-z' })
    expect((await ui.find({ key: 'k-j' }))?.text).toContain('↓')
    expect(await ui.find({ key: 'k-v' })).toBeDefined()
    await ui.press({ key: 'k-z' })
    expect((await ui.find({ key: 'k-j' }))?.text).toContain('down')
    await ui.unmount()
  }

  const ui = await $.ui.mount({ plugin: 'comments-review', surface: 'terminal', component: 'Pane', requestId: 'comments-review', props: PANE_PROPS })
  // n jumps to the thread on line 3 and shows it.
  await ui.press({ key: 'k-n' })
  expect(await ui.find({ type: 'Text', text: /Is step one needed\?/ })).toBeDefined()

  // b + text adds a blocking comment on the cursor line.
  await ui.press({ key: 'k-b' })
  await ui.input({ key: 'compose', text: 'Needs a rollback step' })
  expect(serve.actions.at(-1)).toMatchObject({ action: 'add', line: 3, blocking: true, text: 'Needs a rollback step' })
  expect(await ui.find({ type: 'Text', text: /1 blocking/ })).toBeDefined()

  // An empty Enter cancels: Esc never reaches a plugin, so this is the way out.
  const before = serve.actions.length
  await ui.press({ key: 'k-c' })
  await ui.input({ key: 'compose', text: '' })
  expect(serve.actions.length).toBe(before)
  expect(await ui.find({ key: 'k-j' })).toBeDefined()

  // s resolves the thread under the cursor (blocking first).
  await ui.press({ key: 'k-s' })
  expect(serve.actions.at(-1)).toMatchObject({ action: 'resolve', thread_id: 'c2' })

  // v then a records approval and hands the turn back to Claude.
  await ui.press({ key: 'k-v' })
  await ui.press({ key: 'k-a' })
  expect(serve.actions.at(-1)).toMatchObject({ action: 'verdict', decision: 'approved' })
  expect(serve.reviews).toHaveLength(1)
  expect(prompts[0]).toContain('approved the review of plan.md')
  await ui.unmount()

  exit()
})

test('inside herdr, /review-doc opens comments view beside Claude and the verdict wakes Claude', async ($, on) => {
  const runs: string[][] = []
  const prompts: string[] = []
  let woke = (_: string) => undefined as void
  const wakeup = new Promise<string>(r => (woke = r))
  let watchArgv: readonly string[] = []
  mock.clock(on)
  mock.env(on, { HERDR_ENV: '1', HERDR_PANE_ID: 'w1:p1' })
  on('ui.status', async () => ({ value: undefined }) as never)
  on('ui.toast', async () => ({ value: undefined }) as never)
  on('session.start', async (_$, e) => e as never)
  on('command.register', async () => ({ value: undefined }) as never)
  on('process.run', async (_$, e) => {
    runs.push([...e.argv])
    const out = (stdout: string) => ({ value: { exitCode: 0, stdout, stderr: '' } as never })
    if (e.argv[0] === 'herdr' && e.argv[2] === 'split') return out(JSON.stringify({ result: { pane: { pane_id: 'w1:p9' } } }))
    if (e.argv[0] === 'herdr') return out('{"result":{"type":"ok"}}')
    if (e.argv[1] === 'gate') return out(JSON.stringify({ decision: 'changes_requested', summary: { blocking: 1, non_blocking: 0, pending_suggestions: 0 } }))
    return out('')
  })
  on('process.spawn', async function* (_$, e) {
    watchArgv = e.argv
    yield { stream: 'stdout' as const, text: '{"event":"signoff","author":"eric","decision":"approved","note":"ship it"}\n' }
    return { value: { code: 0, signal: null } }
  })
  on('prompt.submit', async (_$, e) => {
    prompts.push(e.text)
    woke(e.text)
    return e as never
  })
  void $.session.start({ cwd: '/repo', surface: 'terminal', isInteractive: true } as Parameters<typeof $.session.start>[0])

  const ran = await $.command.run({ command: 'review-doc', args: 'plan.md' } as Parameters<typeof $.command.run>[0])
  expect(ran.text).toContain('herdr pane on the right')
  expect(runs[0]).toEqual(['herdr', 'pane', 'split', '--pane', 'w1:p1', '--direction', 'right', '--cwd', '/repo', '--focus'])
  expect(runs[1]).toEqual(['herdr', 'pane', 'run', 'w1:p9', "'comments' view 'plan.md'; exit"])
  expect(watchArgv.slice(0, 5)).toEqual(['comments', 'watch', 'plan.md', '--until', 'signoff'])
  // The approval is reported with the gate, which still blocks.
  await wakeup
  expect(prompts[0]).toContain('gate is still changes_requested with 1 blocking')
  expect(prompts[0]).toContain('ship it')
})

test('plan mode: ExitPlanMode becomes a comments review, the doc locks, approval lets the reviewed plan through', async ($, on) => {
  const files = new Map<string, string>()
  const prompts: string[] = []
  let woke = (_: string) => undefined as void
  const wakeup = new Promise<string>(r => (woke = r))
  let gate = 'changes_requested'
  mock.clock(on)
  mock.env(on, { HERDR_ENV: '1', HERDR_PANE_ID: 'w1:p1' })
  on('ui.status', async () => ({ value: undefined }) as never)
  on('ui.toast', async () => ({ value: undefined }) as never)
  on('session.start', async (_$, e) => e as never)
  on('command.register', async () => ({ value: undefined }) as never)
  on('fs.write', async (_$, e) => {
    files.set(e.path, e.text)
    return { value: undefined } as never
  })
  on('fs.read', async (_$, e) => ({ value: files.get(e.path) ?? '' }) as never)
  on('process.run', async (_$, e) => {
    const out = (stdout: string, exitCode = 0) => ({ value: { exitCode, stdout, stderr: '' } as never })
    if (e.argv[0] === 'herdr' && e.argv[2] === 'split') return out(JSON.stringify({ result: { pane: { pane_id: 'w1:p9' } } }))
    if (e.argv[0] === 'herdr' && e.argv[2] === 'get') return out('{"result":{}}')
    if (e.argv[1] === 'gate') return out(JSON.stringify({ decision: gate, summary: { blocking: gate === 'approved' ? 0 : 1, non_blocking: 0, pending_suggestions: 0 } }))
    return out('')
  })
  let release = () => undefined as void
  const reviewed = new Promise<void>(r => (release = r))
  on('process.spawn', async function* () {
    await reviewed
    yield { stream: 'stdout' as const, text: '{"event":"signoff","author":"eric","decision":"approved","note":""}\n' }
    return { value: { code: 0, signal: null } }
  })
  on('prompt.submit', async (_$, e) => {
    prompts.push(e.text)
    woke(e.text)
    return e as never
  })
  on('tool.call', async () => ({ result: 'ran' }) as never)
  void $.session.start({ cwd: '/repo', surface: 'terminal', isInteractive: true } as Parameters<typeof $.session.start>[0])

  const first = await $.tool.call({ tool: 'ExitPlanMode', tool_use_id: 't1', plan: '# Cache Policy\n\nStep one.' } as never)
  const firstText = JSON.stringify(first)
  expect(firstText).toContain('Plan v1 saved to docs/artifacts/plans/')
  expect(firstText).toContain('cache-policy.md')
  expect(firstText).toContain('NOT approved')
  const doc = [...files.keys()][0] ?? ''
  expect(files.get(doc)).toBe('# Cache Policy\n\nStep one.\n')

  // #3: the doc is locked while under review.
  const edit = await $.tool.call({ tool: 'Edit', tool_use_id: 't2', file_path: doc, old_string: 'one', new_string: 'two' } as never)
  expect(JSON.stringify(edit)).toContain('under human review')

  // The reviewer accepts a suggestion (the doc changes) and approves; the gate passes.
  files.set(doc, '# Cache Policy\n\nStep one, reviewed.\n')
  gate = 'approved'
  release()
  expect(await wakeup).toContain('approved in comments review and its gate passes')

  // The second ExitPlanMode passes tool.call and is allowed with the reviewed doc.
  await $.tool.call({ tool: 'ExitPlanMode', tool_use_id: 't3', plan: '# Cache Policy\n\nStep one.' } as never)
  const allowed = await $.classic.PermissionRequest({ tool_name: 'ExitPlanMode', tool_use_id: 't3', tool_input: { plan: '# Cache Policy\n\nStep one.' } } as never)
  expect(JSON.stringify(allowed)).toContain('"behavior":"allow"')
  expect(JSON.stringify(allowed)).toContain('Step one, reviewed.')
})

test('watchTarget finds the doc of a signoff watch, and only that', () => {
  expect(watchTarget('comments watch docs/p.md --until signoff --since 2026-10-06T20:00:00Z')).toBe('docs/p.md')
  expect(watchTarget("cd repo && comments watch --since x 'docs/p.md' --until=signoff,gate_changed")).toBe('docs/p.md')
  expect(watchTarget('comments watch docs/p.md --until gate_changed')).toBe(null)
  expect(watchTarget('comments inbox docs/p.md --json')).toBe(null)
})

test('only reads run while a hand-off is pending', () => {
  expect(allowedDuringReview('Read', undefined)).toBe(true)
  expect(allowedDuringReview('Bash', 'comments inbox docs/p.md --json')).toBe(true)
  expect(allowedDuringReview('Bash', 'git diff')).toBe(true)
  expect(allowedDuringReview('Bash', 'comments reply docs/p.md --thread c1 --text hi')).toBe(false)
  expect(allowedDuringReview('Bash', 'cat a > b')).toBe(false)
  expect(allowedDuringReview('Edit', undefined)).toBe(false)
})

test('a brief hand-off opens review beside Claude, pauses writes, and the verdict wakes Claude and lifts the pause', async ($, on) => {
  const prompts: string[] = []
  let woke = (_: string) => undefined as void
  const wakeup = new Promise<string>(r => (woke = r))
  let release = () => undefined as void
  const reviewed = new Promise<void>(r => (release = r))
  mock.clock(on)
  mock.env(on, { HERDR_ENV: '1', HERDR_PANE_ID: 'w1:p1' })
  on('ui.status', async () => ({ value: undefined }) as never)
  on('ui.toast', async () => ({ value: undefined }) as never)
  on('session.start', async (_$, e) => e as never)
  on('command.register', async () => ({ value: undefined }) as never)
  on('fs.read', async (_$, e) => ({ value: e.path.endsWith('brief.md') ? '---\ncomments:\n    template: brief\n---\n# B\n' : '# R\n' }) as never)
  on('process.run', async (_$, e) => {
    const out = (stdout: string) => ({ value: { exitCode: 0, stdout, stderr: '' } as never })
    if (e.argv[0] === 'herdr' && e.argv[2] === 'split') return out(JSON.stringify({ result: { pane: { pane_id: 'w1:p9' } } }))
    if (e.argv[0] === 'herdr' && e.argv[2] === 'get') return out('{"result":{}}')
    if (e.argv[1] === 'gate') return out(JSON.stringify({ decision: 'approved', summary: { blocking: 0, non_blocking: 0, pending_suggestions: 0 } }))
    return out('')
  })
  on('process.spawn', async function* () {
    await reviewed
    yield { stream: 'stdout' as const, text: '{"event":"signoff","author":"eric","decision":"approved","note":""}\n' }
    return { value: { code: 0, signal: null } }
  })
  on('prompt.submit', async (_$, e) => {
    prompts.push(e.text)
    woke(e.text)
    return e as never
  })
  on('tool.call', async () => ({ result: 'ran' }) as never)
  void $.session.start({ cwd: '/repo', surface: 'terminal', isInteractive: true } as Parameters<typeof $.session.start>[0])

  // A research doc's watch is left alone (it blocks as written).
  const research = await $.tool.call({ tool: 'Bash', tool_use_id: 'b0', command: 'comments watch docs/research.md --until signoff' } as never)
  expect(JSON.stringify(research)).toContain('ran')

  const handoff = await $.tool.call({ tool: 'Bash', tool_use_id: 'b1', command: 'comments watch docs/brief.md --until signoff --since 2026-10-06T20:00:00Z' } as never)
  expect(JSON.stringify(handoff)).toContain('Hand-off done')

  const write = await $.tool.call({ tool: 'Bash', tool_use_id: 'b2', command: 'go test ./...' } as never)
  expect(JSON.stringify(write)).toContain('only reads run')
  const readOk = await $.tool.call({ tool: 'Read', tool_use_id: 'b3', file_path: '/repo/x.go' } as never)
  expect(JSON.stringify(readOk)).toContain('ran')

  release()
  expect(await wakeup).toContain('approved')
  const after = await $.tool.call({ tool: 'Bash', tool_use_id: 'b4', command: 'go test ./...' } as never)
  expect(JSON.stringify(after)).toContain('ran')
})

test('phase 2: code edits wait for an approved, current plan; fail closed; unlock needs the person', async ($, on) => {
  let approval: { decision: string; freshness: string } | null = null
  let gate = 'changes_requested'
  let broken = false
  const clock = mock.clock(on)
  mock.env(on, { HERDR_ENV: '1', HERDR_PANE_ID: 'w1:p1' })
  on('ui.status', async () => ({ value: undefined }) as never)
  on('ui.toast', async () => ({ value: undefined }) as never)
  on('session.start', async (_$, e) => e as never)
  on('command.register', async () => ({ value: undefined }) as never)
  on('fs.read', async () => ({ value: '---\ncomments:\n    template: plan\n---\n# P\n' }) as never)
  on('process.run', async (_$, e) => {
    const out = (stdout: string) => ({ value: { exitCode: 0, stdout, stderr: '' } as never })
    if (e.argv[0] === 'herdr' && e.argv[2] === 'split') return out(JSON.stringify({ result: { pane: { pane_id: 'w1:p9' } } }))
    if (e.argv[0] === 'herdr') return out('{"result":{}}')
    if (broken) return out('not json')
    if (e.argv[1] === 'context') return out(JSON.stringify({ implementation: { approval: approval ?? { decision: '', freshness: 'missing' } } }))
    if (e.argv[1] === 'gate') return out(JSON.stringify({ decision: gate, summary: { blocking: 0, non_blocking: 0, pending_suggestions: 0 } }))
    return out('')
  })
  on('process.spawn', async function* () {
    return { value: { code: 1, signal: null } }
  })
  on('prompt.submit', async (_$, e) => e as never)
  on('tool.call', async () => ({ result: 'ran' }) as never)
  void $.session.start({ cwd: '/repo', surface: 'terminal', isInteractive: true } as Parameters<typeof $.session.start>[0])

  // Handing the plan off makes it the active contract.
  await $.tool.call({ tool: 'Bash', tool_use_id: 'h1', command: 'comments watch docs/plan.md --until signoff' } as never)
  const edit = (id: string) => $.tool.call({ tool: 'Edit', tool_use_id: id, file_path: '/repo/pkg/x.go', old_string: 'a', new_string: 'b' } as never)
  const advance = () => clock.advance(4000)

  expect(JSON.stringify(await edit('e1'))).toContain('not approved yet')
  approval = { decision: 'approved', freshness: 'current' }
  gate = 'approved'
  await advance()
  expect(JSON.stringify(await edit('e2'))).toContain('ran')
  approval = { decision: 'approved', freshness: 'stale' }
  await advance()
  expect(JSON.stringify(await edit('e3'))).toContain('approval is stale')
  broken = true
  await advance()
  expect(JSON.stringify(await edit('e4'))).toContain('contract check failed')

  // The plan doc itself stays editable (once its review closes); --unlock refuses a non-person origin.
  const unlock = await $.command.run({ command: 'review-doc', args: '--unlock' } as Parameters<typeof $.command.run>[0])
  expect(unlock.text).toContain('only from your own Enter')
  expect(JSON.stringify(await edit('e5'))).toContain('Code edits wait')
})

test('phase 3: a restarted session restores the active plan, and compaction keeps one fresh contract note', async ($, on) => {
  mock.clock(on)
  mock.env(on, {})
  mock.store(on, { 'activePlan:/repo': 'docs/plan.md' })
  on('ui.status', async () => ({ value: undefined }) as never)
  on('ui.toast', async () => ({ value: undefined }) as never)
  on('session.start', async (_$, e) => e as never)
  on('command.register', async () => ({ value: undefined }) as never)
  on('process.run', async (_$, e) => {
    const out = (stdout: string) => ({ value: { exitCode: 0, stdout, stderr: '' } as never })
    if (e.argv[1] === 'context') return out(JSON.stringify({ implementation: { approval: { decision: 'approved', freshness: 'stale' } } }))
    if (e.argv[1] === 'gate') return out(JSON.stringify({ decision: 'approved', summary: { blocking: 0, non_blocking: 0, pending_suggestions: 0 } }))
    if (e.argv[1] === 'inbox')
      return out(JSON.stringify({ items: [{ thread: { id: 'p1', pick: 'index' } }, { thread: { id: 'q1' } }, { thread: { id: 'p2', pick: 'lru' } }] }))
    return out('')
  })
  on('process.spawn', async function* () {
    return { value: { code: 1, signal: null } }
  })
  on('session.compact', async () => ({
    messages: [
      { role: 'user', text: 'summary of the work so far', toolUses: [] },
      { role: 'user', text: '[comments contract] Active plan: old.md. stale note', toolUses: [] },
    ],
  }) as never)

  await $.session.start({ cwd: '/repo', surface: 'terminal', isInteractive: true } as Parameters<typeof $.session.start>[0])
  // The session-start reminder goes through session.append, which the kit cannot stand in for;
  // the note below proves the plan was restored from the store (nothing else set it).

  const compacted = (await $.session.compact({ trigger: 'manual', messages: [{ role: 'user', text: 'earlier turn', toolUses: [] }] } as never)) as unknown as { messages: { text?: string }[] }
  const notes = compacted.messages.filter(m => (m.text ?? '').startsWith('[comments contract]'))
  expect(notes).toHaveLength(1)
  expect(notes[0]?.text).toContain('Active plan: docs/plan.md. Gate: approved, 0 blocking')
  expect(notes[0]?.text).toContain('locked (approval is stale')
  expect(notes[0]?.text).toContain('2 open picks')
  expect(compacted.messages[0]?.text).toBe('summary of the work so far')
})

test('living doc: writing it makes it the session doc, code edits count as drift, and a locked plan never blocks it', async ($, on) => {
  const statuses: string[] = []
  mock.clock(on)
  mock.env(on, {})
  mock.store(on, { 'activePlan:/repo': 'docs/plan.md' })
  on('ui.status', async (_$, e) => {
    statuses.push(String((e as { text?: unknown }).text ?? JSON.stringify(e)))
    return { value: undefined } as never
  })
  on('ui.toast', async () => ({ value: undefined }) as never)
  on('session.start', async (_$, e) => e as never)
  on('command.register', async () => ({ value: undefined }) as never)
  on('fs.read', async () => ({ value: '---\ncomments:\n    template: plan\n---\n# P\n' }) as never)
  on('process.run', async (_$, e) => {
    const out = (stdout: string) => ({ value: { exitCode: 0, stdout, stderr: '' } as never })
    if (e.argv[1] === 'context') return out(JSON.stringify({ implementation: { approval: { decision: '', freshness: 'missing' } } }))
    if (e.argv[1] === 'gate') return out(JSON.stringify({ decision: 'changes_requested', summary: { blocking: 1, non_blocking: 0, pending_suggestions: 0 } }))
    if (e.argv[1] === 'inbox') return out(JSON.stringify({ items: [] }))
    return out('')
  })
  on('process.spawn', async function* () {
    return { value: { code: 1, signal: null } }
  })
  on('tool.call', async () => ({ result: 'ran' }) as never)
  on('session.compact', async () => ({ messages: [{ role: 'user', text: 'summary', toolUses: [] }] }) as never)
  await $.session.start({ cwd: '/repo', surface: 'terminal', isInteractive: true } as Parameters<typeof $.session.start>[0])

  const living = '---\ncomments:\n    template: living\n---\n# Work\n\n## Now\n\n- building\n'
  const write = (id: string, path: string, content: string) => $.tool.call({ tool: 'Write', tool_use_id: id, file_path: path, content } as never)
  const edit = (id: string) => $.tool.call({ tool: 'Edit', tool_use_id: id, file_path: '/repo/pkg/x.go', old_string: 'a', new_string: 'b' } as never)

  // The parked-but-active plan locks code, yet the living doc is always writable.
  expect(JSON.stringify(await edit('e0'))).toContain('Code edits wait')
  expect(JSON.stringify(await write('w1', '/repo/docs/work.md', living))).toContain('ran')

  // --park needs the person; from a composer origin it lifts the lock.
  const refused = await $.command.run({ command: 'review-doc', args: '--park' } as Parameters<typeof $.command.run>[0])
  expect(refused.text).toContain('only from your own Enter')
  const parked = await $.command.run({ command: 'review-doc', args: '--park', origin: { kind: 'composer' } } as unknown as Parameters<typeof $.command.run>[0])
  expect(parked.text).toContain('Parked docs/plan.md')
  expect(JSON.stringify(await edit('e1'))).toContain('ran')
  expect(JSON.stringify(await edit('e2'))).toContain('ran')

  const compacted = (await $.session.compact({ trigger: 'manual', messages: [{ role: 'user', text: 'earlier turn', toolUses: [] }] } as never)) as unknown as { messages: { text?: string }[] }
  const notes = compacted.messages.filter(m => (m.text ?? '').startsWith('[comments living doc]'))
  expect(notes).toHaveLength(1)
  expect(notes[0]?.text).toContain('Living doc: /repo/docs/work.md. 2 code edits since it last changed.')
  expect(statuses.join('\n')).toContain('doc work.md: 2 code edits since the doc')

  // Updating the doc brings the tracks back together.
  await write('w2', '/repo/docs/work.md', living + '- done\n')
  const again = (await $.session.compact({ trigger: 'manual', messages: [{ role: 'user', text: 'earlier turn', toolUses: [] }] } as never)) as unknown as { messages: { text?: string }[] }
  expect(again.messages.find(m => (m.text ?? '').startsWith('[comments living doc]'))?.text).toContain('It is current with the code.')
})

test('plans and briefs are hand-offs; research and design docs are not', () => {
  expect(isHandoffFront('comments:\n    template: plan')).toBe(true)
  expect(isHandoffFront('comments:\n    template: brief')).toBe(true)
  expect(isHandoffFront('comments:\n    template: research-deep')).toBe(false)
  expect(isHandoffFront('comments:\n    template: briefing')).toBe(false)
})

test('living doc: a restarted session restores the real drift, and --done clears it', async ($, on) => {
  mock.clock(on)
  mock.env(on, {})
  mock.store(on, { 'livingDoc:/repo': '/repo/docs/work.md', 'drift:/repo': '3' })
  on('ui.status', async () => ({ value: undefined }) as never)
  on('ui.toast', async () => ({ value: undefined }) as never)
  on('session.start', async (_$, e) => e as never)
  on('command.register', async () => ({ value: undefined }) as never)
  on('process.run', async () => ({ value: { exitCode: 0, stdout: '{}', stderr: '' } as never }))
  on('process.spawn', async function* () {
    return { value: { code: 1, signal: null } }
  })
  on('session.compact', async () => ({ messages: [{ role: 'user', text: 'summary', toolUses: [] }] }) as never)
  await $.session.start({ cwd: '/repo', surface: 'terminal', isInteractive: true } as Parameters<typeof $.session.start>[0])
  const compact = async () =>
    ((await $.session.compact({ trigger: 'manual', messages: [{ role: 'user', text: 'earlier turn', toolUses: [] }] } as never)) as unknown as { messages: { text?: string }[] }).messages.find(m =>
      (m.text ?? '').startsWith('[comments living doc]'),
    )
  expect((await compact())?.text).toContain('3 code edits since it last changed.')

  const done = await $.command.run({ command: 'review-doc', args: '--done' } as Parameters<typeof $.command.run>[0])
  expect(done.text).toContain('Done with /repo/docs/work.md')
  expect(await compact()).toBeUndefined()
})

test('living docs are recognised by frontmatter, and the drift text counts edits', () => {
  expect(isLivingFront('comments:\n    template: living')).toBe(true)
  expect(isLivingFront('comments:\n    template: brief')).toBe(false)
  expect(driftStatus('/a/work.md', 0)).toBe('doc work.md: current')
  expect(driftStatus('/a/work.md', 1)).toBe('doc work.md: 1 code edit since the doc')
  expect(livingNote('w.md', 3)).toContain('3 code edits since it last changed.')
})
