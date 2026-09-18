#!/usr/bin/env python3
"""Surface-parity acceptance test: drive the skill's review loop through the
comments CLI and through a live MCP server, and fail if they ever disagree.

    python3 scripts/eval/surface-parity/drive.py ./comments <empty-workdir> ./verdict

Twin workspaces (cli/, mcp/) get identical documents. Every AGENT step runs
through the surface under test, with no TTY and no COMMENTS_ACTOR — exactly how
an agent calls it. HUMAN replies go through the CLI with COMMENTS_ACTOR=human
(humans never use MCP); the human's VERDICT goes through the `verdict` helper,
because the tool deliberately has no command that records one.

It checks four things: every step succeeds or fails identically on both
surfaces; every step returns the same JSON keys on both; the two workspaces end
with equivalent sidecars; and the authority rules hold (an agent cannot resolve
in a human zone, a refused resolve posts nothing, and no command or tool can
accept a suggestion or record a verdict).
"""
import json, os, subprocess, sys, threading, time, shutil, datetime

BIN = os.path.abspath(sys.argv[1])
ROOT = os.path.abspath(sys.argv[2])
VERDICT = os.path.abspath(sys.argv[3])
DOC_REL = "docs/artifacts/designs/cache-policy.md"

BODY = """
## Pitch

Reads are slow and a cache is the smallest fix.
Decision: add a read-through cache in front of the store.

## Problem

Page loads take four seconds because every read hits the store.
Users on slow links abandon the page.

## Goals / Non-Goals

Goal: median read under 200 ms.
Non-goal: rewriting the store.

## Proposed Design

A read-through cache sits in front of the store.
Entries expire after sixty seconds.

## Options Considered

### Option 1: Read-through cache (chosen)

- Pro: small change.
- Con: staleness window.

### Option 2: Rewrite the store

- Pro: fixes the root cause.
- Con: months of work.

## Risks

Stale reads for up to sixty seconds: accepted.

## Definition of Done

- automated: `go test ./cache/...` passes.
- manual: a reviewer confirms the median read time on the dashboard.

## Unresolved Questions

- blocking: should the expiry be configurable per route?
"""


class MCP:
    def __init__(self, cwd):
        env = dict(os.environ)
        env.pop("COMMENTS_ACTOR", None)
        self.p = subprocess.Popen([BIN, "serve-mcp"], cwd=cwd, stdin=subprocess.PIPE,
                                  stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True, env=env)
        self.id = 0
        self.lock = threading.Lock()
        self._rpc("initialize", {"protocolVersion": "2025-06-18", "capabilities": {},
                                 "clientInfo": {"name": "e2e", "version": "0"}})
        self.p.stdin.write(json.dumps({"jsonrpc": "2.0", "method": "notifications/initialized"}) + "\n")
        self.p.stdin.flush()

    def _rpc(self, method, params):
        with self.lock:
            self.id += 1
            rid = self.id
            self.p.stdin.write(json.dumps({"jsonrpc": "2.0", "id": rid, "method": method, "params": params}) + "\n")
            self.p.stdin.flush()
            while True:
                line = self.p.stdout.readline()
                if not line:
                    return {"error": {"message": "server closed"}}
                try:
                    d = json.loads(line)
                except ValueError:
                    continue
                if d.get("id") == rid:
                    return d

    def call(self, tool, args):
        t0 = time.time()
        d = self._rpc("tools/call", {"name": "comments_" + tool, "arguments": args})
        ms = int((time.time() - t0) * 1000)
        if "error" in d:
            return {"ok": False, "out": d["error"].get("message", ""), "ms": ms, "proto_error": True}
        r = d["result"]
        text = "".join(c.get("text", "") for c in r.get("content", []))
        return {"ok": not r.get("isError", False), "out": text, "ms": ms}

    def close(self):
        self.p.kill()


def cli(cwd, args, human=False, stdin=None):
    env = dict(os.environ)
    env.pop("COMMENTS_ACTOR", None)
    if human:
        env["COMMENTS_ACTOR"] = "human"
    t0 = time.time()
    p = subprocess.run([BIN] + args, cwd=cwd, capture_output=True, text=True, env=env, input=stdin, timeout=60)
    return {"ok": p.returncode == 0, "exit": p.returncode, "out": (p.stdout + p.stderr), "ms": int((time.time() - t0) * 1000)}


def jparse(s):
    try:
        return json.loads(s)
    except ValueError:
        return None


def find_ids(ws):
    """thread ids from the sidecar, keyed by a text fragment."""
    side = os.path.join(ws, DOC_REL + ".comments.json")
    d = json.load(open(side))
    out = {}
    for c in d.get("threads", []):
        out[c.get("Text", "")[:60]] = c["ID"]
    return d, out


def run(surface, checks):
    ws = os.path.join(ROOT, surface)
    shutil.rmtree(ws, ignore_errors=True)
    os.makedirs(ws)
    subprocess.run(["git", "init", "-q", "."], cwd=ws)
    mcp = MCP(ws) if surface == "mcp" else None
    log = []

    def agent(step, cli_args, tool, mcp_args, stdin=None):
        r = cli(ws, cli_args, stdin=stdin) if surface == "cli" else mcp.call(tool, mcp_args)
        r.update(step=step, surface=surface)
        log.append(r)
        return r

    def human(cli_args):
        r = cli(ws, cli_args, human=True)
        assert r["ok"], ("human step failed", cli_args, r["out"])

    def check(name, ok, detail=""):
        checks.append((surface, name, bool(ok), detail))

    def payload(r):
        return jparse(r["out"].strip()) or {}

    # ---- create under a template: new -> context (carries the brief) -> validate
    agent("new", ["new", "cache-policy", "--template", "design-doc", "--title", "Cache policy", "--json"],
          "new", {"name": "cache-policy", "template": "design-doc", "title": "Cache policy"})
    doc = os.path.join(ws, DOC_REL)
    scaffold = open(doc).read()
    open(doc, "w").write(scaffold.split("\n## ")[0].rstrip("\n") + "\n" + BODY)
    ctx = agent("context drafting", ["context", DOC_REL, "--for", "drafting", "--json"], "context", {"filepath": DOC_REL, "for": "drafting"})
    brief = payload(ctx).get("brief") or {}
    check("context carries the writing brief with a reading path", brief.get("template") == "design-doc" and brief.get("reading_path"))
    check("brief marks Pitch as the human tier-1 section", any(x.get("heading") == "Pitch" and x.get("zone") == "human" and x.get("tier") == 1 for x in brief.get("sections", [])))
    agent("validate", ["validate", DOC_REL, "--json"], "validate", {"filepath": DOC_REL})

    # ---- annotate: one path, one or many
    one = [{"anchor": "Page loads take four seconds", "author": "claude", "type": "Q", "blocking": True, "text": "Is four seconds measured or estimated?"}]
    agent("add one", ["add", DOC_REL, "--anchor", one[0]["anchor"], "--author", "claude", "--type", "Q", "--blocking", "--text", one[0]["text"], "--json-out"],
          "add", {"filepath": DOC_REL, "comments": one})
    many = [{"author": "claude", "section": "Cache policy > Proposed Design", "text": "Sixty seconds is an assumption, not a measurement."},
            {"author": "claude", "anchor": "Stale reads for up to sixty seconds", "text": "Weakest reasoning: no mitigation offered.", "blocking": True}]
    agent("add many", ["add", DOC_REL, "--json", "-", "--json-out"], "add", {"filepath": DOC_REL, "comments": many}, stdin=json.dumps(many))
    bad = [dict(many[0]), {"author": "claude", "anchor": "this text is not in the doc", "text": "x"}]
    r = agent("add is atomic, names the bad item", ["add", DOC_REL, "--json", "-"], "add", {"filepath": DOC_REL, "comments": bad}, stdin=json.dumps(bad))
    check("bad batch is refused and names comment 2", (not r["ok"]) and "comment 2" in r["out"], r["out"][:120])

    # ---- the single read
    r = agent("inbox", ["inbox", DOC_REL, "--json"], "inbox", {"filepath": DOC_REL})
    box = payload(r)
    check("refused batch added nothing (3 threads)", box.get("count") == 3, box.get("count"))
    check("inbox carries the gate decision", box.get("decision") == "changes_requested")
    check("blocking threads sort first", [i["reasons"][0] for i in box.get("items", [])][:2] == ["blocking", "blocking"])
    ids = {i["thread"]["text"]: i["thread"]["id"] for i in box.get("items", [])}
    q_id = next(v for k, v in ids.items() if "four seconds" in k)
    risk_id = next(v for k, v in ids.items() if "Weakest" in k)
    design_id = next(v for k, v in ids.items() if "Sixty seconds is" in k)
    agent("get all", ["get", DOC_REL, "--json"], "get", {"filepath": DOC_REL})
    agent("get one", ["get", DOC_REL, "--thread", q_id, "--json"], "get", {"filepath": DOC_REL, "comment_id": q_id})

    # ---- hand off and wait; the human replies, adds a blocker, gives a verdict
    since = datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    time.sleep(1.1)
    waiter = {}

    def wait():
        if surface == "cli":
            waiter["r"] = cli(ws, ["watch", DOC_REL, "--until", "signoff", "--since", since, "--interval", "200ms"])
        else:
            waiter["r"] = mcp.call("watch", {"filepath": DOC_REL, "since": since, "timeout_seconds": 30})

    th = threading.Thread(target=wait)
    th.start()
    time.sleep(1.5)
    human(["reply", DOC_REL, "--thread", q_id, "--author", "eric", "--text", "Measured: p50 from the dashboard."])
    human(["add", DOC_REL, "--anchor", "Entries expire after sixty seconds", "--author", "eric", "--blocking", "--text", "Make expiry configurable."])
    v = subprocess.run([VERDICT, doc, "eric", "changes_requested", "expiry must be configurable"], capture_output=True, text=True)
    assert v.returncode == 0, v.stderr
    th.join(timeout=40)
    r = waiter.get("r", {"ok": False, "out": ""})
    if surface == "cli":
        events = [jparse(l) for l in r["out"].strip().split("\n") if l.strip().startswith("{")]
    else:
        events = (jparse(r["out"]) or {}).get("events", [])
    last = events[-1] if events else {}
    check("wait returns the signoff with decision and note", last.get("event") == "signoff" and last.get("decision") == "changes_requested" and last.get("note") == "expiry must be configurable", last)

    # ---- inbox first, then act
    r = agent("inbox since", ["inbox", DOC_REL, "--since", since, "--json"], "inbox", {"filepath": DOC_REL, "since": since})
    box = payload(r)
    reasons = {i["thread"]["text"]: i["reasons"] for i in box.get("items", [])}
    check("inbox flags the human's reply and new thread as news", "new_reply" in next(v for k, v in reasons.items() if "four seconds" in k) and "new_thread" in reasons.get("Make expiry configurable.", []), reasons)
    check("inbox carries the reviewer's verdict", (box.get("files") or [{}])[0].get("last_review", {}).get("note") == "expiry must be configurable")
    expiry_id = next(i["thread"]["id"] for i in box["items"] if i["thread"]["text"] == "Make expiry configurable.")

    agent("reply", ["reply", DOC_REL, "--thread", expiry_id, "--author", "claude", "--text", "Will propose a suggestion.", "--json-out"],
          "reply", {"filepath": DOC_REL, "replies": [{"thread_id": expiry_id, "author": "claude", "text": "Will propose a suggestion."}]})
    r = agent("resolve in a HUMAN zone is refused", ["reply", DOC_REL, "--thread", q_id, "--author", "claude", "--text", "done", "--resolve"],
              "reply", {"filepath": DOC_REL, "replies": [{"thread_id": q_id, "author": "claude", "text": "done", "resolve": True}]})
    check("agent resolve in zone: human refused", (not r["ok"]) and "human-decision zone" in r["out"], r["out"][:120])
    one_thread = payload(agent("get after refusal", ["get", DOC_REL, "--thread", q_id, "--json"], "get", {"filepath": DOC_REL, "comment_id": q_id}))
    check("a refused resolve posts nothing", one_thread.get("comment", {}).get("reply_count") == 1, one_thread.get("comment", {}).get("reply_count"))
    agent("reply and resolve in an agent zone", ["reply", DOC_REL, "--thread", design_id, "--author", "claude", "--text", "Fixed: cites the benchmark.", "--resolve", "--json-out"],
          "reply", {"filepath": DOC_REL, "replies": [{"thread_id": design_id, "author": "claude", "text": "Fixed: cites the benchmark.", "resolve": True}]})
    line = "Entries expire after sixty seconds."
    agent("suggest", ["suggest", DOC_REL, "--anchor", line, "--author", "claude", "--text", "make expiry configurable", "--original", line, "--proposed", "Entries expire after a per-route TTL."],
          "suggest", {"filepath": DOC_REL, "anchor": line, "author": "claude", "text": "make expiry configurable", "original_text": line, "proposed_text": "Entries expire after a per-route TTL."})

    # ---- authority: decisions have no agent-reachable path
    for name in ("accept", "reject", "batch-accept", "signoff"):
        r = agent("no such command: " + name, [name, DOC_REL], name.replace("-", "_"), {"filepath": DOC_REL})
        check(f"{name} does not exist", not r["ok"], r["out"][:80])
    check("the suggestion is still pending", len(payload(agent("inbox pending", ["inbox", DOC_REL, "--json"], "inbox", {"filepath": DOC_REL})).get("pending_suggestions", [])) == 1)
    check("the document text is untouched by the agent's suggestion", line in open(doc).read())

    # ---- edit, then migrate the displaced anchor
    text = open(doc).read()
    open(doc, "w").write(text.replace("## Problem\n", "## Problem\n\nContext: measured in production last week.\nSecond inserted line.\n", 1))
    lines = open(doc).read().split("\n")
    target = next(i + 1 for i, l in enumerate(lines) if l.startswith("Stale reads"))
    moves = [{"comment_id": risk_id, "line": target}]
    agent("reanchor", ["reanchor", DOC_REL, "--json", "-", "--json-out"], "reanchor", {"filepath": DOC_REL, "moves": moves}, stdin=json.dumps(moves))
    box = payload(agent("inbox after edit", ["inbox", DOC_REL, "--json"], "inbox", {"filepath": DOC_REL}))
    ch = (box.get("files") or [{}])[0].get("changes") or {}
    check("inbox reports what changed since the verdict", ch.get("reviewer") == "eric" and ch.get("changed_lines", 0) >= 2, ch)

    # ---- the human closes out; the agent's read and the script's gate agree
    for tid in (q_id, risk_id, expiry_id):
        human(["reply", DOC_REL, "--thread", tid, "--resolve"])
    box = payload(agent("inbox final", ["inbox", DOC_REL, "--json"], "inbox", {"filepath": DOC_REL}))
    check("done: approved with nothing waiting", box.get("decision") == "approved" and box.get("count") == 0, (box.get("decision"), box.get("count")))
    check("gate exit code agrees", cli(ws, ["gate", DOC_REL])["ok"])
    agent("analyze", ["analyze", DOC_REL, "--json"], "analyze", {"filepath": DOC_REL})

    if mcp:
        mcp.close()
    return log


def keyset(out):
    o = jparse(out.strip())
    if isinstance(o, dict):
        return sorted(o)
    if isinstance(o, list):
        return "[array]"
    return None


def sidecar(ws):
    sc = json.load(open(os.path.join(ROOT, ws, DOC_REL + ".comments.json")))
    rows = [(t["Line"], t["Author"], t["Text"], t["Resolved"], t["Blocking"], t["AnchorConfidence"], t["IsSuggestion"], t["Accepted"], len(t["Replies"] or [])) for t in sc["threads"]]
    return sorted(rows, key=str), [(r["author"], r["decision"], r.get("note")) for r in sc.get("reviews", [])]


if __name__ == "__main__":
    checks = []
    out = {s: run(s, checks) for s in ("cli", "mcp")}
    json.dump(out, open(os.path.join(ROOT, "results.json"), "w"), indent=1)
    failures = [c for c in checks if not c[2]]
    b = {r["step"]: r for r in out["mcp"]}
    print(f"{'step':44} {'CLI':>6} {'MCP':>6}  json keys")
    for x in out["cli"]:
        y = b.get(x["step"], {})
        same_ok = bool(x.get("ok")) == bool(y.get("ok"))
        kx, ky = keyset(x.get("out", "")), keyset(y.get("out", ""))
        # error text is prose on both; only successful JSON results must match
        same_keys = (not x.get("ok")) or kx is None or kx == ky
        print(f"{x['step'][:44]:44} {'ok' if x.get('ok') else 'fail':>6} {'ok' if y.get('ok') else 'fail':>6}  {'same' if same_keys else 'DIFFERENT'}")
        if not same_ok:
            failures.append(("both", f"{x['step']}: surfaces disagree on success", False, ""))
        if not same_keys:
            failures.append(("both", f"{x['step']}: JSON keys differ", False, f"CLI {kx} / MCP {ky}"))
    if sidecar("cli") != sidecar("mcp"):
        failures.append(("both", "final sidecars differ between surfaces", False, ""))
    print(f"\n{len(checks) - len([c for c in checks if not c[2]])}/{len(checks)} behaviour checks passed; final sidecars {'match' if sidecar('cli') == sidecar('mcp') else 'DIFFER'}")
    for surface, name, _, detail in failures:
        print(f"  FAIL [{surface}] {name} {detail}")
    print("SURFACE PARITY " + ("PASSED" if not failures else "FAILED"))
    sys.exit(1 if failures else 0)
