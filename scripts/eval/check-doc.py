#!/usr/bin/env python3
"""Mechanical quality metrics for a drafted research/plan artifact.

Everything here is checked against the repo, never self-reported by the agent
that wrote the document. An earlier trial reported "all word caps pass" while
holding a real violation, because it validated with a stale binary — so the
eval trusts artifacts, not accounts of artifacts.

Emits JSON:
  citations: total / resolvable / broken (with the specific breakages)
  subsections: per-subsection word counts for the body section
  sections: per-section word counts
"""
import json
import os
import re
import sys

HEADING = re.compile(r'^(#{1,6})\s+(.*)$')
# file.go:12  |  path/to/file.go:12-34  (also inside backticks)
CITATION = re.compile(r'([A-Za-z0-9_./-]+\.(?:go|md|yaml|yml|json)):(\d+)(?:-(\d+))?')


def sections(lines):
    heads = []
    for i, line in enumerate(lines):
        m = HEADING.match(line)
        if m:
            heads.append((len(m.group(1)), m.group(2).strip(), i))
    out = []
    for n, (lvl, title, start) in enumerate(heads):
        end = len(lines)
        for lvl2, _t, start2 in heads[n + 1:]:
            if lvl2 <= lvl:
                end = start2
                break
        out.append({'level': lvl, 'title': title, 'start': start, 'end': end})
    return out


def words(lines, start, end):
    return sum(len(l.split()) for l in lines[start + 1:end])


def resolve(repo, path):
    """Return every file a citation could refer to.

    A bare basename that matches several files is reported as ambiguous rather
    than silently resolved to the first hit: 'doctor.go:169' is not peekable
    when pkg/comment/ and cmd/comments/ both have one, and picking one for the
    agent would hide a real citation defect (and invent false ones).
    """
    direct = os.path.join(repo, path)
    if os.path.isfile(direct):
        return [direct]
    base = os.path.basename(path)
    hits = []
    for root, dirs, files in os.walk(repo):
        dirs[:] = [d for d in dirs if d not in ('.git', 'node_modules')]
        if base in files:
            hits.append(os.path.join(root, base))
    return hits


def check_citations(repo, text):
    total = broken = ambiguous = 0
    details = []
    for m in CITATION.finditer(text):
        path, start, end = m.group(1), int(m.group(2)), m.group(3)
        last = int(end) if end else start
        total += 1
        candidates = resolve(repo, path)
        if not candidates:
            broken += 1
            details.append({'ref': m.group(0), 'why': 'file not found'})
            continue
        in_range = []
        for cand in candidates:
            n = sum(1 for _ in open(cand, errors='replace'))
            if 1 <= start and last <= n:
                in_range.append(cand)
        if not in_range:
            n = sum(1 for _ in open(candidates[0], errors='replace'))
            broken += 1
            details.append({'ref': m.group(0), 'why': f'line out of range (file has {n} lines)'})
        elif len(candidates) > 1:
            ambiguous += 1
            details.append({'ref': m.group(0),
                            'why': f'ambiguous basename, {len(candidates)} files match'})
    return {'total': total, 'valid': total - broken - ambiguous,
            'broken': broken, 'ambiguous': ambiguous, 'details': details}


def main():
    repo, doc, body_heading = sys.argv[1], sys.argv[2], sys.argv[3]
    text = open(doc).read()
    lines = text.split('\n')
    secs = sections(lines)

    body = next((s for s in secs if s['title'].lower().startswith(body_heading.lower())), None)
    subs = []
    if body:
        subs = [{'title': s['title'], 'words': words(lines, s['start'], s['end'])}
                for s in secs if s['level'] == 3 and body['start'] < s['start'] < body['end']]

    print(json.dumps({
        'doc': doc,
        'total_words': len(text.split()),
        'citations': check_citations(repo, text),
        'subsections': subs,
        'sections': [{'title': s['title'], 'words': words(lines, s['start'], s['end'])}
                     for s in secs if s['level'] == 2],
    }, indent=2))


if __name__ == '__main__':
    main()
