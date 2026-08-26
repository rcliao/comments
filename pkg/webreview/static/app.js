const app = {
  state: null,
  mode: "rendered",
  filter: "open",
  events: null,
  busy: false,
  author: "",
  reviewEditing: false,
  selectingTarget: false,
  selectedThreadId: "",
  focusedThreadId: "",
  keyboardScope: "review",
  selectedLine: 1,
  lineSelectionReturnMode: "rendered",
  verdictReturnScope: "review",
  parsedCommentType: "",
};

const AUTHOR_KEY = "comments-review-author";
const THEME_KEY = "comments-review-theme";

const $ = (selector, root = document) => root.querySelector(selector);
const $$ = (selector, root = document) => [...root.querySelectorAll(selector)];

function escapeHTML(value = "") {
  const node = document.createElement("span");
  node.textContent = value;
  return node.innerHTML;
}

function docQuery() {
  return `doc=${encodeURIComponent(app.state?.doc_id || new URLSearchParams(location.search).get("doc") || "")}`;
}

async function loadState(doc = "") {
  const response = await fetch(`/api/state?doc=${encodeURIComponent(doc)}`);
  if (!response.ok) throw new Error((await response.json()).error || "Could not load review");
  app.state = await response.json();
  if (!app.author) app.author = storedPreference(AUTHOR_KEY) || app.state.author;
  history.replaceState(null, "", `/?doc=${encodeURIComponent(app.state.doc_id)}`);
  render();
  connectEvents();
}

function connectEvents() {
  app.events?.close();
  app.events = new EventSource(`/api/events?${docQuery()}`);
  app.events.addEventListener("refresh", async (event) => {
    const change = JSON.parse(event.data);
    if (change.revision !== app.state.revision && !app.busy) {
      showNotice("The document changed on disk. Review state was refreshed.");
      await loadState(app.state.doc_id);
    }
  });
  app.events.onopen = () => { $("#live-indicator").classList.remove("offline"); $("#live-indicator").textContent = "● Live"; };
  app.events.onerror = () => { $("#live-indicator").classList.add("offline"); $("#live-indicator").textContent = "● Reconnecting"; };
}

function render() {
  const state = app.state;
  document.title = `${state.name} · Comments Review`;
  $("#document-name").textContent = state.name;
  $("#rendered-document").innerHTML = state.rendered_html;
  renderDocuments();
  renderGate();
  renderSource();
  renderThreads();
  renderRenderedMarkers();
  bindRenderedCommentTargets();
  renderAuthor();
  renderReviewState();
  renderKeyboardModeHint();
  $("#rendered-document").hidden = app.mode !== "rendered";
  $("#source-document").hidden = app.mode !== "source";
  $("#rendered-mode").classList.toggle("active", app.mode === "rendered");
  $("#source-mode").classList.toggle("active", app.mode === "source");
  if (app.keyboardScope === "line-select") applyLineSelection({scroll: false});
}

function reviewDecision(decision) {
  if (decision === "approved") return {label: "Approved", icon: "✓", className: "approved"};
  if (decision === "changes_requested") return {label: "Changes requested", icon: "!", className: "changes-requested"};
  return {label: "Commented", icon: "↩", className: "commented"};
}

function reviewTime(timestamp) {
  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleString([], {month: "short", day: "numeric", hour: "numeric", minute: "2-digit"});
}

function renderReviewState() {
  const reviews = app.state.document.reviews || [];
  const documentState = $("#document-review-state");
  const summary = $("#review-summary");
  const controls = $("#verdict-controls");
  if (!reviews.length) {
    documentState.hidden = true;
    summary.hidden = true;
    controls.hidden = false;
    return;
  }

  const latest = reviews[reviews.length - 1];
  const decision = reviewDecision(latest.decision);
  const meta = [latest.author, reviewTime(latest.timestamp)].filter(Boolean).join(" · ");
  documentState.className = `document-review-state ${decision.className}`;
  documentState.innerHTML = `<span aria-hidden="true">${decision.icon}</span><strong>${decision.label}</strong><span>${escapeHTML(latest.author)}</span>`;
  documentState.hidden = false;

  $("#review-summary-icon").textContent = decision.icon;
  $("#review-summary-decision").textContent = decision.label;
  $("#review-summary-meta").textContent = meta;
  $("#review-summary-note").textContent = latest.note || "";
  $("#review-summary-note").hidden = !latest.note;
  summary.className = `review-summary ${decision.className}`;
  summary.hidden = app.reviewEditing;
  controls.hidden = !app.reviewEditing;
}

function storedPreference(key) {
  try {
    return localStorage.getItem(key)?.trim() || "";
  } catch {
    return "";
  }
}

function savePreference(key, value) {
  try {
    localStorage.setItem(key, value);
  } catch {
    // Private browsing or storage policy can make preferences session-only.
  }
}

function currentAuthor() {
  return app.author || app.state?.author || "Reviewer";
}

function renderAuthor() {
  const name = currentAuthor();
  $("#author-name").textContent = name;
  $("#author-avatar").textContent = (name.trim()[0] || "R").toUpperCase();
}

function openAuthorDialog() {
  $("#author-input").value = currentAuthor();
  $("#author-dialog").showModal();
  $("#author-input").select();
}

function systemTheme() {
  return matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function applyTheme(theme) {
  document.documentElement.dataset.theme = theme;
  const dark = theme === "dark";
  const button = $("#theme-button");
  button.textContent = dark ? "☀" : "☾";
  button.setAttribute("aria-pressed", String(dark));
  button.setAttribute("aria-label", dark ? "Use light mode" : "Use dark mode");
  button.title = dark ? "Use light mode" : "Use dark mode";
}

function toggleTheme() {
  const theme = document.documentElement.dataset.theme === "dark" ? "light" : "dark";
  savePreference(THEME_KEY, theme);
  applyTheme(theme);
}

function renderDocuments() {
  const select = $("#doc-select");
  select.hidden = app.state.documents.length < 2;
  select.innerHTML = app.state.documents.map(doc => `<option value="${escapeHTML(doc.id)}" ${doc.id === app.state.doc_id ? "selected" : ""}>${doc.blocking ? "● " : ""}${escapeHTML(doc.name)}</option>`).join("");
}

function renderGate() {
  const gate = app.state.gate;
  const reviews = app.state.document.reviews || [];
  if (gate.blocking) {
    $("#gate-summary").innerHTML = `<span class="review-status blocking">${gate.blocking} blocking</span>`;
  } else if (gate.pending_suggestions) {
    $("#gate-summary").innerHTML = `<span class="review-status">${gate.pending_suggestions} pending</span>`;
  } else if (!reviews.length) {
    $("#gate-summary").innerHTML = `<span class="review-status clean">Ready to approve</span>`;
  } else {
    $("#gate-summary").innerHTML = "";
  }
}

function rootThreads() { return app.state.document.threads || []; }

function threadIsOpen(thread) {
  return !thread.resolved && (!thread.is_suggestion || thread.accepted == null);
}

function markerState(threads) {
  if (threads.some(thread => threadIsOpen(thread) && thread.blocking)) return "blocking";
  if (threads.every(thread => !threadIsOpen(thread))) return "resolved";
  return "open";
}

function threadsAtLine(line) {
  return rootThreads().filter(thread => thread.line === line);
}

function renderSource() {
  $("#source-document").innerHTML = app.state.lines.map((line, index) => {
    const number = index + 1;
    const threads = threadsAtLine(number);
    const state = threads.length ? markerState(threads) : "";
    const marker = threads.length
      ? `<button class="source-thread-marker ${state}" data-thread-line="${number}" aria-label="${threads.length} ${threads.length === 1 ? "thread" : "threads"} on line ${number}" title="Focus ${threads.length} ${threads.length === 1 ? "thread" : "threads"} on line ${number}">${threads.length}</button>`
      : "";
    const rowClass = threads.length ? `has-thread ${state === "blocking" ? "has-blocking-thread" : ""}` : "";
    return `<div class="source-line ${rowClass}" id="line-${number}"><button class="line-number" data-line="${number}" title="Add a comment on line ${number}">${number}</button><span class="source-thread-slot">${marker}</span><span class="line-text">${escapeHTML(line) || " "}</span></div>`;
  }).join("");
  $$(".line-number").forEach(button => button.addEventListener("click", () => openComment(Number(button.dataset.line))));
  $$(".source-thread-marker").forEach(button => button.addEventListener("click", () => focusThreadsAtLine(Number(button.dataset.threadLine))));
  $$(".source-line.has-thread").forEach(row => bindLinkedHover(row, [Number(row.id.replace("line-", ""))]));
  $$(".source-line").forEach(row => row.addEventListener("click", event => {
    if (!app.selectingTarget || event.target.closest("button, a, input, textarea, select")) return;
    openComment(Number(row.id.replace("line-", "")));
  }));
}

function renderRenderedMarkers() {
  const blocks = $$(".rendered-block", $("#rendered-document"));
  const threads = rootThreads();
  $("#gutter-legend").hidden = !threads.length;
  if (!blocks.length || !threads.length) return;

  const grouped = new Map();
  for (const thread of threads) {
    let best = null;
    let bestDistance = Number.POSITIVE_INFINITY;
    for (const block of blocks) {
      const start = Number(block.dataset.startLine);
      const end = Number(block.dataset.endLine);
      const distance = thread.line < start ? start - thread.line : thread.line > end ? thread.line - end : 0;
      if (distance < bestDistance) {
        best = block;
        bestDistance = distance;
      }
      if (distance === 0) break;
    }
    if (!grouped.has(best)) grouped.set(best, []);
    grouped.get(best).push(thread);
  }

  for (const [block, blockThreads] of grouped) {
    if (!block) continue;
    block.classList.add("has-threads");
    const lines = [...new Set(blockThreads.map(thread => thread.line))].sort((a, b) => a - b);
    const marker = document.createElement("button");
    marker.className = `rendered-thread-marker ${markerState(blockThreads)}`;
    marker.setAttribute("aria-label", `${blockThreads.length} ${blockThreads.length === 1 ? "thread" : "threads"} near ${lines.map(line => `line ${line}`).join(", ")}`);
    marker.title = `Focus ${blockThreads.length} ${blockThreads.length === 1 ? "thread" : "threads"} · ${lines.map(line => `L${line}`).join(", ")}`;
    marker.innerHTML = `<span class="rendered-thread-count">${blockThreads.length}</span>`;
    marker.addEventListener("click", () => focusThreadsAtLine(lines[0], block));
    block.dataset.threadLines = lines.join(",");
    bindLinkedHover(block, lines);
    block.prepend(marker);
  }
}

function bindRenderedCommentTargets() {
  $$(".rendered-block", $("#rendered-document")).forEach(block => {
    const start = Number(block.dataset.startLine);
    const end = Number(block.dataset.endLine) || start;
    block.addEventListener("click", event => {
      if (!app.selectingTarget || event.target.closest("button, a, input, textarea, select")) return;
      openComment(start, end);
    });
  });
}

function visibleThreads() {
  return rootThreads().filter(thread => {
    if (app.filter === "blocking") return thread.blocking && !thread.resolved;
    if (app.filter === "open") return !thread.resolved && (!thread.is_suggestion || thread.accepted == null);
    return true;
  });
}

function renderThreads() {
  const threads = visibleThreads();
  $("#thread-count").textContent = `${threads.length} ${threads.length === 1 ? "comment" : "comments"}`;
  const list = $("#thread-list");
  if (!threads.length) {
    app.selectedThreadId = "";
    list.innerHTML = `<div class="empty">Nothing in this view.<br>The review queue is clear.</div>`;
    return;
  }
  list.innerHTML = threads.map(threadCard).join("");
  $$('[data-action="jump"]', list).forEach(button => button.addEventListener("click", () => jumpToLine(Number(button.dataset.line))));
  $$('[data-action="reply-toggle"]', list).forEach(button => button.addEventListener("click", () => toggleReplyComposer(button.dataset.id)));
  $$('[data-action="reply"]', list).forEach(button => button.addEventListener("click", () => submitReply(button.dataset.id)));
  $$('[data-action="resolve"]', list).forEach(button => button.addEventListener("click", () => mutate({action: "resolve", thread_id: button.dataset.id})));
  $$('[data-action="reopen"]', list).forEach(button => button.addEventListener("click", () => mutate({action: "reopen", thread_id: button.dataset.id})));
  $$('[data-action="accept"]', list).forEach(button => button.addEventListener("click", () => mutate({action: "accept", thread_id: button.dataset.id})));
  $$('[data-action="reject"]', list).forEach(button => button.addEventListener("click", () => mutate({action: "reject", thread_id: button.dataset.id})));
  $$(".reply-box textarea", list).forEach(textarea => textarea.addEventListener("keydown", event => {
    if (event.key === "Escape") {
      event.preventDefault();
      event.stopPropagation();
      closeReplyComposer(textarea.dataset.threadId);
      return;
    }
    if (event.key === "Enter" && !event.shiftKey && !event.isComposing) {
      event.preventDefault();
      submitReply(textarea.dataset.threadId);
    }
  }));
  $$(".thread-card", list).forEach(card => bindLinkedHover(card, [Number(card.dataset.threadLine)], card));
  $$(".thread-card", list).forEach(card => card.addEventListener("click", event => {
    if (event.target.closest("button, input, textarea, select, a")) return;
    selectThread(card.dataset.threadId, {scroll: false, syncDocument: false});
  }));
  if (!threads.some(thread => thread.id === app.selectedThreadId)) app.selectedThreadId = threads[0].id;
  if (!threads.some(thread => thread.id === app.focusedThreadId)) app.focusedThreadId = "";
  applyThreadSelection();
}

const typeNames = {Q: "Question", S: "Suggestion", B: "Bug", T: "Task", E: "Enhancement"};

function displayThreadText(thread) {
  const text = thread.text || "";
  if (!thread.type) return text;
  const escapedType = thread.type.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return text.replace(new RegExp(`^(?:[❓💡🐛📌✨]\\s*)?\\[${escapedType}\\]\\s*`), "");
}

function threadCard(thread) {
  const state = thread.resolved ? "Resolved" : thread.is_suggestion && thread.accepted != null ? (thread.accepted ? "Accepted" : "Rejected") : "Open";
  const stateLabel = state === "Open" ? "" : `<span class="thread-state">${state}</span>`;
  const typeBadge = thread.type ? `<span class="badge">${escapeHTML(typeNames[thread.type] || thread.type)}</span>` : "";
  const replies = (thread.replies || []).map(reply => `<div class="reply"><span class="reply-author">${escapeHTML(reply.author)}</span><p class="thread-text">${escapeHTML(reply.text)}</p></div>`).join("");
  const diff = thread.is_suggestion ? `<div class="suggestion-diff"><div class="diff-del">− ${escapeHTML(thread.original_text)}</div><div class="diff-add">+ ${escapeHTML(thread.proposed_text)}</div></div>` : "";
  const suggestionActions = thread.is_suggestion && thread.accepted == null ? `<button data-action="accept" data-id="${thread.id}">Accept edit</button><button data-action="reject" data-id="${thread.id}">Reject</button>` : "";
  const resolveAction = thread.resolved ? `<button data-action="reopen" data-id="${thread.id}">Reopen</button>` : `<button data-action="resolve" data-id="${thread.id}">Resolve</button>`;
  return `<section class="thread-card ${thread.blocking ? "blocking" : ""} ${thread.resolved ? "resolved" : ""}" id="thread-${thread.id}" data-thread-id="${thread.id}" data-thread-line="${thread.line}" tabindex="-1">
    <div class="thread-head"><div class="thread-meta"><span class="comment-avatar" aria-hidden="true">${escapeHTML((thread.author || "C")[0].toUpperCase())}</span><span class="thread-author">${escapeHTML(thread.author)}</span>${typeBadge}${thread.blocking ? '<span class="badge warn">Blocking</span>' : ""}</div>${stateLabel}</div>
    <div class="thread-body"><p class="thread-text">${escapeHTML(displayThreadText(thread))}</p>${diff}${replies}</div>
    <div class="thread-actions"><button data-action="jump" data-line="${thread.line}" aria-label="Show comment in source" title="Show in source">View</button><button data-action="reply-toggle" data-id="${thread.id}">Reply</button>${resolveAction}${suggestionActions}</div>
    <div class="reply-box" id="reply-${thread.id}"><div class="reply-input-wrap"><textarea id="reply-input-${thread.id}" data-thread-id="${thread.id}" rows="2" aria-label="Reply text" placeholder="Reply to this thread"></textarea><span class="reply-hint">Enter or ⌘↵ sends · Shift+Enter adds a line · Esc cancels</span></div><button class="primary" data-action="reply" data-id="${thread.id}">Send</button></div>
  </section>`;
}

function selectThreadFilter(filter) {
  app.filter = filter;
  $$('.filters button').forEach(button => button.classList.toggle("active", button.dataset.filter === filter));
  renderThreads();
}

function applyThreadSelection() {
  $$(".thread-card").forEach(card => {
    const selected = card.dataset.threadId === app.selectedThreadId;
    const focused = card.dataset.threadId === app.focusedThreadId;
    card.classList.toggle("keyboard-selected", selected);
    card.classList.toggle("keyboard-focused", focused);
    if (selected) card.setAttribute("aria-current", "true");
    else card.removeAttribute("aria-current");
  });
}

function renderedBlockAtLine(line) {
  return $$(".rendered-block", $("#rendered-document")).find(block => {
    const start = Number(block.dataset.startLine);
    const end = Number(block.dataset.endLine) || start;
    return start <= line && line <= end;
  });
}

function selectThread(id, {scroll = true, syncDocument = true} = {}) {
  const thread = visibleThreads().find(item => item.id === id);
  if (!thread) return;
  if (app.focusedThreadId && app.focusedThreadId !== id) app.focusedThreadId = "";
  app.selectedThreadId = id;
  applyThreadSelection();
  const card = $(`#thread-${CSS.escape(id)}`);
  if (scroll) card?.scrollIntoView({block: "nearest"});
  if (!syncDocument) return;
  const target = app.mode === "source" ? $(`#line-${thread.line}`) : renderedBlockAtLine(thread.line);
  target?.scrollIntoView({block: "center"});
}

function selectRelativeThread(delta) {
  const threads = visibleThreads();
  if (!threads.length) return;
  const current = Math.max(0, threads.findIndex(thread => thread.id === app.selectedThreadId));
  const next = Math.max(0, Math.min(current + delta, threads.length - 1));
  selectThread(threads[next].id);
}

function selectThreadBoundary(last) {
  const threads = visibleThreads();
  if (threads.length) selectThread(threads[last ? threads.length - 1 : 0].id);
}

function selectedThread() {
  return visibleThreads().find(thread => thread.id === app.selectedThreadId);
}

function focusSelectedThread() {
  const thread = selectedThread();
  if (!thread) return;
  app.focusedThreadId = thread.id;
  applyThreadSelection();
  const card = $(`#thread-${CSS.escape(thread.id)}`);
  card?.scrollIntoView({behavior: "smooth", block: "center"});
  card?.focus({preventScroll: true});
}

function clearThreadFocus() {
  const card = app.focusedThreadId ? $(`#thread-${CSS.escape(app.focusedThreadId)}`) : null;
  app.focusedThreadId = "";
  applyThreadSelection();
  card?.blur();
}

function replyToSelectedThread() {
  const thread = selectedThread();
  if (!thread) return;
  app.focusedThreadId = thread.id;
  applyThreadSelection();
  const box = $(`#reply-${CSS.escape(thread.id)}`);
  box?.classList.add("open");
  $(`#reply-input-${CSS.escape(thread.id)}`)?.focus();
}

function toggleReplyComposer(id) {
  const box = $(`#reply-${CSS.escape(id)}`);
  if (!box) return;
  if (box.classList.contains("open")) {
    closeReplyComposer(id);
    return;
  }
  app.selectedThreadId = id;
  app.focusedThreadId = id;
  applyThreadSelection();
  box.classList.add("open");
  $(`#reply-input-${CSS.escape(id)}`)?.focus();
}

function closeReplyComposer(id) {
  const box = $(`#reply-${CSS.escape(id)}`);
  const input = $(`#reply-input-${CSS.escape(id)}`);
  box?.classList.remove("open");
  if (input) input.value = "";
  app.selectedThreadId = id;
  app.focusedThreadId = id;
  applyThreadSelection();
  $(`#thread-${CSS.escape(id)}`)?.focus({preventScroll: true});
}

function actOnSelectedThread(action) {
  const thread = selectedThread();
  if (!thread || app.busy) return;
  if (action === "accept" && thread.is_suggestion && thread.accepted == null) {
    mutate({action: "accept", thread_id: thread.id});
    return;
  }
  if (action !== "reject-or-resolve") return;
  if (thread.is_suggestion && thread.accepted == null) {
    mutate({action: "reject", thread_id: thread.id});
  } else {
    mutate({action: thread.resolved ? "reopen" : "resolve", thread_id: thread.id});
  }
}

function bindLinkedHover(element, lines, exactCard = null) {
  const enter = () => setLinkedHover(lines, true, exactCard);
  const leave = () => setLinkedHover(lines, false, exactCard);
  element.addEventListener("mouseenter", enter);
  element.addEventListener("mouseleave", leave);
  element.addEventListener("focusin", enter);
  element.addEventListener("focusout", event => {
    if (!element.contains(event.relatedTarget)) leave();
  });
}

function setLinkedHover(lines, active, exactCard = null) {
  const lineSet = new Set(lines);
  $$(".thread-card").forEach(card => {
    const matches = exactCard ? card === exactCard : lineSet.has(Number(card.dataset.threadLine));
    card.classList.toggle("linked-hover", active && matches);
  });
  $$(".source-line.has-thread").forEach(row => {
    row.classList.toggle("linked-hover", active && lineSet.has(Number(row.id.replace("line-", ""))));
  });
  $$(".rendered-block.has-threads").forEach(block => {
    const blockLines = (block.dataset.threadLines || "").split(",").filter(Boolean).map(Number);
    block.classList.toggle("linked-hover", active && blockLines.some(line => lineSet.has(line)));
  });
}

function focusThreadsAtLine(line, renderedBlock = null) {
  let card = $(`.thread-card[data-thread-line="${line}"]`);
  if (!card) {
    selectThreadFilter("all");
    card = $(`.thread-card[data-thread-line="${line}"]`);
  }
  if (!card) return;
  app.selectedThreadId = card.dataset.threadId;
  applyThreadSelection();
  $$(".thread-card.focused").forEach(item => item.classList.remove("focused"));
  $$(".rendered-block.comment-selected").forEach(item => item.classList.remove("comment-selected"));
  card.classList.add("focused");
  renderedBlock?.classList.add("comment-selected");
  card.scrollIntoView({behavior: "smooth", block: "center"});
  setTimeout(() => {
    card.classList.remove("focused");
    renderedBlock?.classList.remove("comment-selected");
  }, 1800);
}

function jumpToLine(line) {
  app.mode = "source";
  render();
  requestAnimationFrame(() => $(`#line-${line}`)?.scrollIntoView({behavior: "smooth", block: "center"}));
}

function sectionAtLine(line) {
  return (app.state.sections || [])
    .filter(section => section.start_line <= line && line <= section.end_line)
    .sort((a, b) => b.level - a.level || b.start_line - a.start_line)[0];
}

function commentTargetLabel(start, end = start) {
  const section = sectionAtLine(start);
  const range = start === end ? `line ${start}` : `lines ${start}–${end}`;
  return section ? `${section.path} · ${range}` : range[0].toUpperCase() + range.slice(1);
}

function setTargetSelection(active) {
  app.selectingTarget = active;
  document.body.classList.toggle("selecting-comment-target", active);
}

function beginTargetSelection() {
  app.keyboardScope = "review";
  renderKeyboardModeHint();
  setTargetSelection(true);
  showNotice(app.mode === "source" ? "Click a source line to add a comment. Press Escape to cancel." : "Click a passage or section to add a comment. Press Escape to cancel.");
}

function renderKeyboardModeHint() {
  const hint = $("#keyboard-mode-hint");
  if (app.keyboardScope === "line-select") {
    hint.innerHTML = `<strong>LINE SELECT</strong><span><kbd>j</kbd>/<kbd>k</kbd> move · <kbd>g</kbd>/<kbd>G</kbd> first/last · <kbd>Enter</kbd> comment · <kbd>Esc</kbd> cancel</span>`;
    hint.hidden = false;
    return;
  }
  hint.hidden = true;
}

function applyLineSelection({scroll = true, center = false, focus = scroll} = {}) {
  const total = app.state?.lines.length || 1;
  app.selectedLine = Math.max(1, Math.min(app.selectedLine, total));
  let selectedRow = null;
  $$(".source-line").forEach(row => {
    const selected = Number(row.id.replace("line-", "")) === app.selectedLine;
    row.classList.toggle("keyboard-line-selected", selected);
    if (selected) {
      row.setAttribute("aria-current", "true");
      row.tabIndex = -1;
      selectedRow = row;
    } else {
      row.removeAttribute("aria-current");
      row.removeAttribute("tabindex");
    }
  });
  if (focus) selectedRow?.focus({preventScroll: true});
  if (scroll) selectedRow?.scrollIntoView({block: center ? "center" : "nearest"});
}

function beginLineSelection() {
  app.lineSelectionReturnMode = app.mode;
  app.mode = "source";
  app.keyboardScope = "line-select";
  setTargetSelection(false);
  app.selectedLine = selectedThread()?.line || app.selectedLine || 1;
  render();
  applyLineSelection({center: true});
}

function moveLineSelection(delta) {
  const total = app.state?.lines.length || 1;
  app.selectedLine = Math.max(1, Math.min(app.selectedLine + delta, total));
  applyLineSelection();
}

function selectLineBoundary(last) {
  app.selectedLine = last ? app.state.lines.length : 1;
  applyLineSelection({center: true});
}

function jumpLineToThread(direction) {
  const lines = [...new Set(visibleThreads().map(thread => thread.line))].sort((a, b) => a - b);
  const target = direction > 0
    ? lines.find(line => line > app.selectedLine)
    : lines.reverse().find(line => line < app.selectedLine);
  if (target != null) {
    app.selectedLine = target;
    applyLineSelection({center: true});
  }
}

function closeLineSelection() {
  app.keyboardScope = "review";
  app.mode = app.lineSelectionReturnMode;
  render();
}

function openComment(line = 1, end = line) {
  const targetLine = Math.max(1, Math.min(line, app.state.lines.length));
  const targetEnd = Math.max(targetLine, Math.min(end, app.state.lines.length));
  setTargetSelection(false);
  $("#comment-line").value = targetLine;
  $("#comment-target-label").textContent = commentTargetLabel(targetLine, targetEnd);
  $("#comment-text").value = "";
  $("#comment-type").value = "";
  $("#comment-priority").value = "medium";
  $("#comment-blocking").checked = false;
  app.parsedCommentType = "";
  setCommentComposerStatus("");
  $("#comment-dialog").showModal();
  $("#comment-text").focus();
}

const commentTypeOrder = ["", "Q", "S", "B", "T", "E"];
const commentPriorityOrder = ["medium", "high", "low"];

function cycleSelect(select, values) {
  const current = Math.max(0, values.indexOf(select.value));
  select.value = values[(current + 1) % values.length];
}

function setCommentComposerStatus(message) {
  $("#comment-compose-status").textContent = message;
}

function leadingCommentType(text) {
  return text.match(/^\s*\[([QSBTE])\](?:\s|$)/)?.[1] || "";
}

function replaceLeadingCommentType(type) {
  const input = $("#comment-text");
  const match = input.value.match(/^(\s*)\[([QSBTE])\](\s*)/);
  if (!match) return;
  const replacement = type ? `${match[1]}[${type}]${match[3] || " "}` : match[1];
  const delta = replacement.length - match[0].length;
  const start = input.selectionStart;
  const end = input.selectionEnd;
  input.value = replacement + input.value.slice(match[0].length);
  input.setSelectionRange(Math.max(0, start + delta), Math.max(0, end + delta));
}

function syncCommentTypeFromText() {
  const parsed = leadingCommentType($("#comment-text").value);
  if (parsed) {
    if ($("#comment-type").value !== parsed) {
      $("#comment-type").value = parsed;
      setCommentComposerStatus(`Type: ${typeNames[parsed]} (from [${parsed}])`);
    }
    app.parsedCommentType = parsed;
    return;
  }
  if (app.parsedCommentType) {
    $("#comment-type").value = "";
    app.parsedCommentType = "";
    setCommentComposerStatus("Type: General");
  }
}

function cycleCommentType() {
  const select = $("#comment-type");
  const hadPrefix = Boolean(leadingCommentType($("#comment-text").value));
  cycleSelect(select, commentTypeOrder);
  if (hadPrefix) replaceLeadingCommentType(select.value);
  app.parsedCommentType = hadPrefix ? select.value : "";
  setCommentComposerStatus(`Type: ${select.value ? typeNames[select.value] : "General"}`);
}

function cycleCommentPriority() {
  cycleSelect($("#comment-priority"), commentPriorityOrder);
  setCommentComposerStatus(`Priority: ${$("#comment-priority").value}`);
}

function toggleCommentBlocking() {
  $("#comment-blocking").checked = !$("#comment-blocking").checked;
  setCommentComposerStatus($("#comment-blocking").checked ? "Blocks approval: on" : "Blocks approval: off");
}

function isCommentSubmitShortcut(event) {
  const enter = event.key === "Enter" || event.code === "Enter" || event.code === "NumpadEnter";
  return enter && (event.ctrlKey || event.metaKey);
}

function submitCommentFromShortcut(event) {
  if (!isCommentSubmitShortcut(event)) return;
  event.preventDefault();
  event.stopPropagation();
  if (event.repeat || app.busy || !$("#comment-dialog").open) return;
  $("#comment-form").requestSubmit($("#submit-comment"));
}

async function submitReply(id) {
  const input = $(`#reply-input-${CSS.escape(id)}`);
  if (!input.value.trim()) return;
  await mutate({action: "reply", thread_id: id, text: input.value.trim()});
}

async function mutate(action) {
  if (app.busy) return false;
  app.busy = true;
  try {
    const response = await fetch(`/api/action?${docQuery()}`, {
      method: "POST",
      headers: {"Content-Type": "application/json"},
      body: JSON.stringify({...action, revision: app.state.revision, author: currentAuthor()}),
    });
    const body = await response.json();
    if (response.status === 409) {
      app.state = body.state;
      render();
      showNotice(body.error);
      return false;
    }
    if (!response.ok) throw new Error(body.error || "Action failed");
    app.state = body;
    render();
    return true;
  } catch (error) {
    showNotice(error.message);
    return false;
  } finally {
    app.busy = false;
  }
}

function showNotice(message) {
  const notice = $("#notice");
  notice.textContent = message;
  notice.hidden = false;
  clearTimeout(showNotice.timer);
  showNotice.timer = setTimeout(() => { notice.hidden = true; }, 6000);
}

function openShortcutHelp() {
  if (!$("#shortcut-dialog").open) $("#shortcut-dialog").showModal();
}

function restoreLineSelectionFocus() {
  if (app.keyboardScope === "line-select") {
    requestAnimationFrame(() => applyLineSelection({scroll: false, focus: true}));
  }
}

function closeShortcutHelp() {
  $("#shortcut-dialog").close();
  restoreLineSelectionFocus();
}

function openVerdictKeyboardMode() {
  app.verdictReturnScope = app.keyboardScope;
  app.keyboardScope = "verdict";
  app.reviewEditing = true;
  renderReviewState();
  renderKeyboardModeHint();
  $(".verdict-card").classList.add("keyboard-selected");
  $(".verdict-card").scrollIntoView({block: "nearest"});
  showNotice("Verdict mode: a approve · c request changes · r reply pass · n edit note · Esc back");
}

function closeVerdictKeyboardMode() {
  app.keyboardScope = app.verdictReturnScope || "review";
  $(".verdict-card").classList.remove("keyboard-selected");
  renderKeyboardModeHint();
  if (app.keyboardScope === "line-select") applyLineSelection({scroll: false});
}

async function submitVerdict(decision) {
  app.reviewEditing = false;
  closeVerdictKeyboardMode();
  const recorded = await mutate({action: "verdict", decision, note: $("#verdict-note").value});
  if (!recorded) {
    app.reviewEditing = true;
    renderReviewState();
    return;
  }
  $("#verdict-note").value = "";
  showNotice(`Review recorded: ${decision.replace("_", " ")}`);
}

$("#doc-select").addEventListener("change", event => loadState(event.target.value).catch(error => showNotice(error.message)));
$("#rendered-mode").addEventListener("click", () => { app.mode = "rendered"; render(); });
$("#source-mode").addEventListener("click", () => { app.mode = "source"; render(); });
$("#theme-button").addEventListener("click", toggleTheme);
$("#shortcut-button").addEventListener("click", openShortcutHelp);
$("#close-shortcuts").addEventListener("click", closeShortcutHelp);
$("#author-button").addEventListener("click", openAuthorDialog);
$("#add-comment").addEventListener("click", beginTargetSelection);
$("#change-comment-target").addEventListener("click", () => {
  $("#comment-dialog").close();
  beginTargetSelection();
});
$("#review-again").addEventListener("click", () => {
  app.reviewEditing = true;
  renderReviewState();
  $("#verdict-note").focus();
});
$$('.filters button').forEach(button => button.addEventListener("click", () => {
  selectThreadFilter(button.dataset.filter);
}));
$("#comment-text").addEventListener("input", syncCommentTypeFromText);
$("#comment-type").addEventListener("change", () => {
  if (leadingCommentType($("#comment-text").value)) replaceLeadingCommentType($("#comment-type").value);
  app.parsedCommentType = leadingCommentType($("#comment-text").value);
  setCommentComposerStatus(`Type: ${$("#comment-type").value ? typeNames[$("#comment-type").value] : "General"}`);
});
// Capture the chord before dialog-level handlers, and keep a keyup fallback
// for embedded browser shells that reserve the Meta+Enter keydown. app.busy
// and dialog.open make the two paths idempotent when both events arrive.
$("#comment-form").addEventListener("keydown", submitCommentFromShortcut, true);
$("#comment-form").addEventListener("keyup", submitCommentFromShortcut, true);
$("#comment-dialog").addEventListener("keydown", event => {
  const ctrlShortcut = event.ctrlKey && !event.metaKey && !event.altKey;
  if (ctrlShortcut && event.key.toLowerCase() === "t") {
    event.preventDefault();
    cycleCommentType();
    return;
  }
  if (ctrlShortcut && event.key.toLowerCase() === "p") {
    event.preventDefault();
    cycleCommentPriority();
    return;
  }
  if (ctrlShortcut && event.key.toLowerCase() === "b") {
    event.preventDefault();
    toggleCommentBlocking();
    return;
  }
  if (ctrlShortcut && event.key.toLowerCase() === "s") {
    event.preventDefault();
    $("#comment-form").requestSubmit($("#submit-comment"));
  }
});
$("#comment-form").addEventListener("submit", async event => {
  event.preventDefault();
  if (event.submitter?.value === "cancel") {
    $("#comment-dialog").close();
    restoreLineSelectionFocus();
    return;
  }
  const saved = await mutate({action: "add", line: Number($("#comment-line").value), text: $("#comment-text").value, type: $("#comment-type").value, priority: $("#comment-priority").value, blocking: $("#comment-blocking").checked});
  if (saved) {
    $("#comment-dialog").close();
    restoreLineSelectionFocus();
  }
});
$("#author-form").addEventListener("submit", event => {
  event.preventDefault();
  if (event.submitter?.value === "cancel") { $("#author-dialog").close(); return; }
  const name = $("#author-input").value.trim();
  if (!name) return;
  app.author = name;
  savePreference(AUTHOR_KEY, name);
  renderAuthor();
  $("#author-dialog").close();
  showNotice(`Commenting as ${name}`);
});
$$('[data-verdict]').forEach(button => button.addEventListener("click", () => submitVerdict(button.dataset.verdict)));
document.addEventListener("keydown", event => {
  if ($("#shortcut-dialog").open) {
    if (event.key === "?" || event.key === "Escape") { event.preventDefault(); closeShortcutHelp(); }
    return;
  }
  if ($("#comment-dialog").open) {
    if (event.key === "Escape") {
      event.preventDefault();
      $("#comment-dialog").close();
      restoreLineSelectionFocus();
    }
    return;
  }
  if ($("#author-dialog").open) {
    if (event.key === "Escape") { event.preventDefault(); $("#author-dialog").close(); }
    return;
  }
  if (event.target.matches("input, textarea, select")) {
    if (app.keyboardScope === "verdict" && event.target === $("#verdict-note") && event.key === "Escape") {
      event.preventDefault();
      event.target.blur();
      showNotice("Verdict mode: a approve · c request changes · r reply pass · n edit note · Esc back");
    }
    return;
  }
  // Keep native keyboard activation for focused controls. Letter shortcuts
  // still work after clicking a button, but Enter/Space must click that button.
  if (event.target.closest("button, a") && (event.key === "Enter" || event.key === " ")) return;
  if (event.metaKey || event.ctrlKey || event.altKey) return;

  if (app.keyboardScope === "verdict") {
    if (event.key === "Escape" || event.key === "q") { event.preventDefault(); closeVerdictKeyboardMode(); return; }
    if (event.key === "n") { event.preventDefault(); $("#verdict-note").focus(); return; }
    const decision = {a: "approved", c: "changes_requested", r: "commented"}[event.key];
    if (decision && !event.repeat) { event.preventDefault(); submitVerdict(decision); }
    return;
  }

  if (app.keyboardScope === "line-select") {
    if (event.key === "Escape") { event.preventDefault(); closeLineSelection(); return; }
    if (event.key === "?") { event.preventDefault(); openShortcutHelp(); return; }
    if (event.key === "j" || event.key === "ArrowDown") { event.preventDefault(); moveLineSelection(1); return; }
    if (event.key === "k" || event.key === "ArrowUp") { event.preventDefault(); moveLineSelection(-1); return; }
    if (event.key === "g") { event.preventDefault(); selectLineBoundary(false); return; }
    if (event.key === "G") { event.preventDefault(); selectLineBoundary(true); return; }
    if (event.key === "]") { event.preventDefault(); jumpLineToThread(1); return; }
    if (event.key === "[") { event.preventDefault(); jumpLineToThread(-1); return; }
    if (event.key === "Enter" || event.key === "c") { event.preventDefault(); openComment(app.selectedLine); return; }
    if (event.key === "q") { event.preventDefault(); openVerdictKeyboardMode(); return; }
    return;
  }

  if (event.key === "Escape" && app.selectingTarget) {
    event.preventDefault();
    setTargetSelection(false);
    showNotice("Comment target selection cancelled.");
    return;
  }
  if (event.key === "Escape" && app.focusedThreadId) {
    event.preventDefault();
    clearThreadFocus();
    return;
  }
  if (event.key === "?") { event.preventDefault(); openShortcutHelp(); return; }
  if (event.key === "j" || event.key === "ArrowDown") { event.preventDefault(); selectRelativeThread(1); return; }
  if (event.key === "k" || event.key === "ArrowUp") { event.preventDefault(); selectRelativeThread(-1); return; }
  if (event.key === "g") { event.preventDefault(); selectThreadBoundary(false); return; }
  if (event.key === "G") { event.preventDefault(); selectThreadBoundary(true); return; }
  if (event.key === "Enter") { event.preventDefault(); focusSelectedThread(); return; }
  if (event.key === "r") { event.preventDefault(); replyToSelectedThread(); return; }
  if (event.key === "a" && !event.repeat) { event.preventDefault(); actOnSelectedThread("accept"); return; }
  if (event.key === "x" && !event.repeat) { event.preventDefault(); actOnSelectedThread("reject-or-resolve"); return; }
  if (event.key === "1") { event.preventDefault(); selectThreadFilter("open"); return; }
  if (event.key === "2") { event.preventDefault(); selectThreadFilter("blocking"); return; }
  if (event.key === "3") { event.preventDefault(); selectThreadFilter("all"); return; }
  if (event.key === "q") { event.preventDefault(); openVerdictKeyboardMode(); return; }
  if (event.key === "c") { event.preventDefault(); beginLineSelection(); return; }
  if (event.key === "s") { event.preventDefault(); app.mode = app.mode === "source" ? "rendered" : "source"; render(); }
});

applyTheme(storedPreference(THEME_KEY) || systemTheme());
loadState(new URLSearchParams(location.search).get("doc") || "").catch(error => showNotice(error.message));
