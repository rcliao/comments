# Scratch: practitioner writing on human/agent collaboration (Q1 sub-report)

Provenance: received 2026-09-18 as an agent message from sub-agent a555da1ef9793ee84 (listed under this session; presumed child of `q1-workflows`, whose handback failed).
Sender's method: each VERBATIM quote fixed-string matched by script against the curl'd raw page; 32/32 matched.
The lead session has NOT re-verified these quotes; re-check any quote before it is pinned in the evidence note.

## 1. Simon Willison

a) https://simonwillison.net/2025/Oct/5/parallel-coding-agents/ — 2025-10-05.
VERBATIM: "AI-generated code needs to be reviewed, which means the natural bottleneck on all of this is how fast I can review the results."
VERBATIM: "Reviewing code that lands on your desk out of nowhere is a lot of work."
Supports: review is the bottleneck; code from your own spec is cheaper to review.

b) https://simonwillison.net/2025/Oct/7/vibe-engineering/ — 2025-10-07.
VERBATIM: "I propose we call this vibe engineering, with my tongue only partially in my cheek."
VERBATIM: "managing a growing army of weird digital interns who will absolutely cheat if you give them a chance, and spending so much time on code review."
Supports: human owns research, architecture, specs, success criteria, QA, review.

c) https://simonwillison.net/2025/Dec/18/code-proven-to-work/ — 2025-12-18.
VERBATIM: "Your job is to deliver code you have proven to work."
VERBATIM: "Not doing that directly shifts the burden of the actual work to whoever is expected to review our code."
Supports: author supplies proof before asking for review.

d) Guide https://simonwillison.net/guides/agentic-engineering-patterns/ — announced 2026-02-23 (https://simonwillison.net/2026/Feb/23/agentic-engineering-patterns/).
/what-is-agentic-engineering/ VERBATIM: "I use the term agentic engineering to describe the practice of developing software with the assistance of coding agents."
Same chapter VERBATIM: "The craft has always been figuring out what code to write."
/anti-patterns/ VERBATIM: "Don't file pull requests with code you haven't reviewed yourself."
Same chapter VERBATIM: "The initial review pass is your responsibility, not something you should farm out to others."
Chapter list includes: Red/green TDD, First run the tests, Agentic manual testing, Linear walkthroughs, Interactive explanations, Hoard things you know how to do.

## 2. Geoffrey Litt

https://www.geoffreylitt.com/2025/10/24/code-like-a-surgeon — October 2025.
VERBATIM: "A surgeon isn’t a manager, they do the actual work!"
VERBATIM: "It’s dangerous to conflate different parts of the autonomy spectrum"
VERBATIM: "The goal isn’t to delegate your core work, it’s to identify and delegate the secondary grunt work tasks, so you can focus on the main thing that matters."
Supports: primary vs secondary task split.
Separate Litt piece on reviewing AI output: not found.

## 3. Addy Osmani

a) https://addyo.substack.com/p/the-70-problem-hard-truths-about — 2024-12-04.
VERBATIM: "They can get 70% of the way there surprisingly quickly, but that final 30% becomes an exercise in diminishing returns."

b) https://addyosmani.com/blog/ai-coding-workflow/ — 2026-01-04.
VERBATIM: "the first step is brainstorming a detailed specification with the AI, then outlining a step-by-step plan, before writing any actual code."
VERBATIM: "you are responsible for quality - always review and test thoroughly."

c) https://addyosmani.com/blog/comprehension-debt/ — 2026-03-14.
VERBATIM: "Comprehension debt is the growing gap between how much code exists in your system and how much of it any human being genuinely understands."
VERBATIM: "comprehension debt breeds false confidence. The codebase looks clean. The tests are green."

d) https://addyosmani.com/blog/code-review-ai/ — 2026-01-07.
VERBATIM: "When output increases faster than verification capacity, review becomes the rate limiter."
VERBATIM: "Human sign-off isn’t going away - it’s evolving to focus on what AI misses, like roadmap alignment and institutional context that AI can’t grasp."
A Substack post "Code Review in the Age of AI" also exists; not fetched, same-text unconfirmed.

e) https://addyo.substack.com/p/the-80-problem-in-agentic-coding — 2026-01-28; subtitle "Managing comprehension debt when leaning on AI to code"; names "Assumption propagation"; no quote extracted.

## 4. Kent Beck

https://newsletter.kentbeck.com/p/augmented-coding-beyond-the-vibes — 2025-06-25 (Q&A section).
VERBATIM: "In vibe coding you don't care about the code, just the behavior of the system."
VERBATIM: "In augmented coding you care about the code, its complexity, the tests, & their coverage."
VERBATIM (warning signs): "Loops. Functionality I hadn't asked for (even if it was a reasonable next step)." and "Any indication that the genie was cheating, for example by disabling or deleting tests."
TDD as guardrail: seen in raw text, not script-verified.

## 5. Thoughtworks Technology Radar

Volume mapping: Vol 31 = 2024/10 and Vol 32 = 2025/04 confirmed from archive paths; Vol 33 = Nov 2025 and Vol 34 = Apr 2026 are inferred.

a) Spec-driven development — https://www.thoughtworks.com/radar/techniques/spec-driven-development — Assess, 2025-11-05; not on the current edition.
VERBATIM: "some generate lengthy spec files that are hard to review"
VERBATIM: "We may be relearning a bitter lesson — that handcrafting detailed rules for AI ultimately doesn’t scale."

b) Complacency with AI-generated code — https://www.thoughtworks.com/radar/techniques/complacency-with-ai-generated-code — Hold Apr 2025 and Nov 2025.
VERBATIM (Nov 2025): "The rise of coding agents further amplifies these risks, since AI now generates larger change sets that are harder to review."
VERBATIM: "We recommend reinforcing established practices such as TDD and static analysis, and embedding them directly into coding workflows"
Oct 2024 text names "automation bias, sunk cost fallacy, anchoring bias and review fatigue" — seen, not script-verified.

c) Context engineering — https://www.thoughtworks.com/radar/techniques/context-engineering — Assess Nov 2025, Adopt Apr 2026.
VERBATIM: "context engineering treats the context window as a design surface and intentionally constructs the AI’s information environment."

d) AI-accelerated shadow IT — https://www.thoughtworks.com/radar/techniques/ai-accelerated-shadow-it — Hold 2025, "Caution" Apr 2026.
VERBATIM (Apr 2026): "Distinguishing between disposable, one-off workflows and critical processes that require durable, production-ready implementation is key to balancing experimentation with control."

e) Curated shared instructions for software teams — https://www.thoughtworks.com/radar/techniques/curated-shared-instructions-for-software-teams — Adopt; no quote extracted.

## 6. RPI / HumanLayer

a) Dex Horthy, "Getting AI to Work in Complex Codebases" — https://github.com/humanlayer/advanced-context-engineering-for-coding-agents/blob/main/ace-fca.md — undated; references a YC talk on 2025-08-20.
VERBATIM: "But a bad line of a **plan** could lead to hundreds of bad lines of code."
VERBATIM: "When you review the research and the plans, you get more leverage than you do when you review the code."

b) RPI revision: YouTube "Everything We Got Wrong About Research-Plan-Implement - Dexter Horthy", AAIF Live, 2026-03-24 — https://www.youtube.com/watch?v=YwZR6tc7qYg.
VERBATIM from the video description (organiser's abstract, NOT Horthy's words): "RPI was supposed to fix AI coding, but Dexter Horthy says it kind of broke it, especially when teams started outsourcing thinking to agents, so now he’s pushing qrspi: fewer magic prompts, more structure, more human ownership"
Transcript not retrieved.
"CRISPY" naming and "1,000-line plans that humans didn't read": SNIPPET ONLY from secondary pages (zenml.io, alexlavaee.me); no primary page found.

c) Not searched: "No Vibes Allowed" talk, Geoffrey Huntley "ralph", Steve Yegge "Gas Town".
