# Scratch: SDD tools — division of labour and where the human reviews (Q1 sub-report)

Provenance: hand-back from sub-agent a35a27a7131e348b4 (child of `q1-workflows`), received 2026-09-18.
Sender's method: every VERBATIM quote grep-confirmed against curl'd raw page text.
Lead re-verification status is recorded in the evidence note, not here.

## 1. GitHub Spec Kit

1a. https://github.blog/ai-and-ml/generative-ai/spec-driven-development-with-ai-get-started-with-a-new-open-source-toolkit/ — Den Delimarsky, 2025-09-02.
VERBATIM: "Your primary role is to steer; the coding agent does the bulk of the writing."
VERBATIM: "Crucially, your role isn't just to steer. It's to verify. At each phase, you reflect and refine."
VERBATIM: "The AI generates the artifacts; you ensure they're right."
VERBATIM: "instead of reviewing thousand-line code dumps, you, the developer, review focused changes that solve specific problems."

1b. https://raw.githubusercontent.com/github/spec-kit/main/spec-driven.md — undated.
VERBATIM: "Both templates mandate the use of `[NEEDS CLARIFICATION]` markers:"
VERBATIM: "2. **Don't guess**: If the prompt doesn't specify something, mark it"
VERBATIM: "These checklists force the LLM to self-review its output systematically"

1c. https://raw.githubusercontent.com/github/spec-kit/main/templates/commands/clarify.md — undated.
VERBATIM: "Identify underspecified areas in the current feature spec by asking up to 5 highly targeted clarification questions and encoding answers back into the spec."
VERBATIM: "Maximum of 5 total questions across the whole session."

1d. https://raw.githubusercontent.com/github/spec-kit/main/README.md — fetched 2026-09-18.
VERBATIM: "Constitution once per project; specify → plan → tasks → implement → converge per feature."
VERBATIM: "Invoke each `/speckit-*` **skill in your agent's chat**, one at a time, and review the result before continuing."
VERBATIM: "Add clarification, checklists, and consistency analysis when you need extra quality gates."

## 2. AWS Kiro

2a. https://kiro.dev/blog/introducing-kiro/ — Swaminathan and Singh, 2025-07-14.
VERBATIM: "This makes your prompt assumptions explicit, so you know Kiro is building what you want."
VERBATIM: "Kiro then generates a design document by analyzing your codebase and approved spec requirements."

2b. https://kiro.dev/docs/specs/ — undated.
VERBATIM: "All specs follow a three-phase workflow that transforms your idea into executable implementation:"
VERBATIM: "you can also use Quick Spec to auto-generate all three artifacts without approval gates."
VERBATIM: "Answering the checkpoint question sends your comments to the agent as one revision request."
Assembled from fragments split by a code element: "At each phase checkpoint, press Ctrl+X to read that phase's document and stage line comments."
VERBATIM (web mode): "Review and refine the plan directly in your browser - chat with Kiro to add a requirement, rethink part of the design, or adjust the task breakdown, and the agent updates the artifacts in place."
Note: Kiro CLI now has staged line comments on the phase document, batched into one revision request at a phase gate — the same shape as the comments review loop.
Note: this post-dates `docs/research-rpi-harness-comparison.md` F2, which says the Kiro source "does not describe inline commenting".

## 3. OpenSpec

https://raw.githubusercontent.com/Fission-AI/OpenSpec/main/README.md — undated.
VERBATIM: "Your AI writes these; you review the plan before any code is written."
VERBATIM: "**Agree before you build** — human and AI align on specs before code gets written"
VERBATIM: "**Work fluidly** — update any artifact anytime, no rigid phase gates"
VERBATIM: "proposal.md — why we're doing this, what's changing"

## 4. Tessl

4a. https://tessl.io/blog/tessl-launches-spec-driven-framework-and-registry — 2025-09-23.
VERBATIM: "The Tessl Framework lets teams define what to build before they start coding, using carefully crafted specifications (“specs”) or AI-generated “vibe-specs.”"
4b. https://tessl.io/blog/how-tessls-products-pioneer-spec-driven-development — 2025-09-16; byline unconfirmed.
VERBATIM: "at the moment, using agents means endless reviewing and frustrating attempts to correct mistakes after the fact."
VERBATIM: "You can review and edit the plan before they start executing it, and it will get updated as it executes, serving as an audit trail."
Not found: the phrase "spec-as-source" in either Tessl post; it is Boeckeler's label.

## 5. Critiques

5a. Boeckeler, https://martinfowler.com/articles/exploring-gen-ai/context-engineering-coding-agents.html — 2026-02-05.
VERBATIM: "there are no unit tests for context engineering"
5b. Boeckeler, https://martinfowler.com/articles/exploring-gen-ai/harness-engineering.html — 2026-04-02.
VERBATIM: "The human's job in this is to steer the agent by iterating on the harness."
VERBATIM: "it provides a feedback loop that self-corrects as many issues as possible before they even reach human eyes. Ultimately it should reduce the review toil"
VERBATIM: "Correctness is outside any sensor's remit if the human didn't clearly specify what they wanted in the first place."
5c. Zaninotto, https://marmelab.com/blog/2025/11/12/spec-driven-development-waterfall-strikes-back.html — 2025-11-12.
VERBATIM: "Markdown Madness: SDD produces too much text, especially in the design phase. Developers spend most of their time reading long Markdown files, hunting for basic mistakes hidden in overly verbose, expert-sounding prose. It’s exhausting."
VERBATIM: "Double Code Review: The technical specification already contains code. Developers must review this code before running it, and since there will still be bugs, they’ll need to review the final implementation too. As a result, review time doubles."
5d. Thoughtworks Radar SDD blip — see `q1-practitioners.md`.
5e. Kent Beck as quoted by Martin Fowler, https://martinfowler.com/fragments/2026-01-08.html — 2026-01-08; Beck's original not fetched.
VERBATIM (as quoted): "The descriptions of Spec-Driven development that I have seen emphasize writing the whole specification before implementation. This encodes the (to me bizarre) assumption that you aren’t going to learn anything during implementation that would change the specification."
VERBATIM (Fowler): "the learning loop of experimentation is essential to the model building that’s at the heart of any kind of worthwhile specification."

## Cross-cutting (sender's synthesis)

All four tools have the agent author the markdown and place the human at gates before code is written.
None except Kiro's CLI documents a line-anchored comment mechanism for that review.
The critics' shared complaint is the volume and verbosity of what the human must read.
