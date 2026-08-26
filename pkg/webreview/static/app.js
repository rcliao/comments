const app = {
  state: null,
  mode: "rendered",
  filter: "open",
  events: null,
  busy: false,
  author: "",
  reviewEditing: false,
  selectingTarget: false,
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
  $("#rendered-document").hidden = app.mode !== "rendered";
  $("#source-document").hidden = app.mode !== "source";
  $("#rendered-mode").classList.toggle("active", app.mode === "rendered");
  $("#source-mode").classList.toggle("active", app.mode === "source");
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
    list.innerHTML = `<div class="empty">Nothing in this view.<br>The review queue is clear.</div>`;
    return;
  }
  list.innerHTML = threads.map(threadCard).join("");
  $$('[data-action="jump"]', list).forEach(button => button.addEventListener("click", () => jumpToLine(Number(button.dataset.line))));
  $$('[data-action="reply-toggle"]', list).forEach(button => button.addEventListener("click", () => $(`#reply-${CSS.escape(button.dataset.id)}`).classList.toggle("open")));
  $$('[data-action="reply"]', list).forEach(button => button.addEventListener("click", () => submitReply(button.dataset.id)));
  $$('[data-action="resolve"]', list).forEach(button => button.addEventListener("click", () => mutate({action: "resolve", thread_id: button.dataset.id})));
  $$('[data-action="reopen"]', list).forEach(button => button.addEventListener("click", () => mutate({action: "reopen", thread_id: button.dataset.id})));
  $$('[data-action="accept"]', list).forEach(button => button.addEventListener("click", () => mutate({action: "accept", thread_id: button.dataset.id})));
  $$('[data-action="reject"]', list).forEach(button => button.addEventListener("click", () => mutate({action: "reject", thread_id: button.dataset.id})));
  $$(".reply-box textarea", list).forEach(textarea => textarea.addEventListener("keydown", event => {
    if (event.key === "Enter" && !event.shiftKey && !event.isComposing) {
      event.preventDefault();
      submitReply(textarea.dataset.threadId);
    }
  }));
  $$(".thread-card", list).forEach(card => bindLinkedHover(card, [Number(card.dataset.threadLine)], card));
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
  return `<section class="thread-card ${thread.blocking ? "blocking" : ""} ${thread.resolved ? "resolved" : ""}" id="thread-${thread.id}" data-thread-line="${thread.line}">
    <div class="thread-head"><div class="thread-meta"><span class="comment-avatar" aria-hidden="true">${escapeHTML((thread.author || "C")[0].toUpperCase())}</span><span class="thread-author">${escapeHTML(thread.author)}</span>${typeBadge}${thread.blocking ? '<span class="badge warn">Blocking</span>' : ""}</div>${stateLabel}</div>
    <div class="thread-body"><p class="thread-text">${escapeHTML(displayThreadText(thread))}</p>${diff}${replies}</div>
    <div class="thread-actions"><button data-action="jump" data-line="${thread.line}" aria-label="Show comment in source" title="Show in source">View</button><button data-action="reply-toggle" data-id="${thread.id}">Reply</button>${resolveAction}${suggestionActions}</div>
    <div class="reply-box" id="reply-${thread.id}"><div class="reply-input-wrap"><textarea id="reply-input-${thread.id}" data-thread-id="${thread.id}" rows="2" aria-label="Reply text" placeholder="Reply to this thread"></textarea><span class="reply-hint">Enter or ⌘↵ sends · Shift+Enter adds a line</span></div><button class="primary" data-action="reply" data-id="${thread.id}">Send</button></div>
  </section>`;
}

function selectThreadFilter(filter) {
  app.filter = filter;
  $$('.filters button').forEach(button => button.classList.toggle("active", button.dataset.filter === filter));
  renderThreads();
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
  setTargetSelection(true);
  showNotice(app.mode === "source" ? "Click a source line to add a comment. Press Escape to cancel." : "Click a passage or section to add a comment. Press Escape to cancel.");
}

function openComment(line = 1, end = line) {
  const targetLine = Math.max(1, Math.min(line, app.state.lines.length));
  const targetEnd = Math.max(targetLine, Math.min(end, app.state.lines.length));
  setTargetSelection(false);
  $("#comment-line").value = targetLine;
  $("#comment-target-label").textContent = commentTargetLabel(targetLine, targetEnd);
  $("#comment-text").value = "";
  $("#comment-dialog").showModal();
  $("#comment-text").focus();
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

$("#doc-select").addEventListener("change", event => loadState(event.target.value).catch(error => showNotice(error.message)));
$("#rendered-mode").addEventListener("click", () => { app.mode = "rendered"; render(); });
$("#source-mode").addEventListener("click", () => { app.mode = "source"; render(); });
$("#theme-button").addEventListener("click", toggleTheme);
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
$("#comment-form").addEventListener("submit", async event => {
  event.preventDefault();
  if (event.submitter?.value === "cancel") { $("#comment-dialog").close(); return; }
  const saved = await mutate({action: "add", line: Number($("#comment-line").value), text: $("#comment-text").value, type: $("#comment-type").value, priority: $("#comment-priority").value, blocking: $("#comment-blocking").checked});
  if (saved) $("#comment-dialog").close();
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
$$('[data-verdict]').forEach(button => button.addEventListener("click", async () => {
  app.reviewEditing = false;
  const recorded = await mutate({action: "verdict", decision: button.dataset.verdict, note: $("#verdict-note").value});
  if (!recorded) {
    app.reviewEditing = true;
    renderReviewState();
    return;
  }
  $("#verdict-note").value = "";
  showNotice(`Review recorded: ${button.dataset.verdict.replace("_", " ")}`);
}));
document.addEventListener("keydown", event => {
  if (event.target.matches("input, textarea, select") || $("#comment-dialog").open || $("#author-dialog").open) return;
  if (event.key === "Escape" && app.selectingTarget) { setTargetSelection(false); showNotice("Comment target selection cancelled."); }
  if (event.key === "c") beginTargetSelection();
  if (event.key === "s") { app.mode = app.mode === "source" ? "rendered" : "source"; render(); }
});

applyTheme(storedPreference(THEME_KEY) || systemTheme());
loadState(new URLSearchParams(location.search).get("doc") || "").catch(error => showNotice(error.message));
