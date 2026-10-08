# Q1 scratch: how engineers collaborate with AI coding agents (named workflows, division of labour, where review attention goes)

Compiled 2026-09-18.
Method: WebSearch/WebFetch were used only to locate passages, because the WebFetch summarizer paraphrases and stitches sentences.
A quote marked CONFIRMED was matched with grep against the raw page text fetched with curl.
A quote marked UNCONFIRMED came from the WebFetch summarizer and could not be matched against raw text.
A quote marked SNIPPET ONLY was seen only in a search-result snippet.

## 1. RPI and frequent intentional compaction (HumanLayer)

### 1.1 Dex Horthy, "Getting AI to Work in Complex Codebases" (a.k.a. "Advanced Context Engineering for Coding Agents")

URL: https://github.com/humanlayer/advanced-context-engineering-for-coding-agents/blob/main/ace-fca.md
Author/org: Dex Horthy, HumanLayer.
Date: the file carries no explicit publication date; it says it is "based on a talk given at Y Combinator on August 20th" and embeds screenshots named "Screenshot 2025-08-29", so late August 2025.
Quote (CONFIRMED, line 60): "Again, this is all built around a workflow we call frequent intentional compaction - essentially designing your entire development process around context management, keeping utilization in the 40-60% range, and building in high-leverage human review at exactly the right points."
Quote (CONFIRMED, line 60): "We use a "research, plan, implement" workflow, but the core capabilities/learnings here are FAR more general than any specific workflow or set of prompts."
Quote (CONFIRMED, lines 228/237/244, the three phase definitions): "Understand the codebase, the files relevant to the issue, and how information flows, and perhaps potential causes of a problem." / "Outline the exact steps we'll take to fix the issue, and the files we'll need to edit and how, being super precise about the testing / verification steps in each phase." / "Step through the plan, phase by phase."
Quote (CONFIRMED, lines 322-324, three consecutive sentences): "A bad line of code is… a bad line of code." "But a bad line of a **plan** could lead to hundreds of bad lines of code." "And a bad line of **research**, a misunderstanding of how the codebase works or where certain functionality is located, could land you with thousands of bad lines of code."
Correction to the brief: the source says a bad line of PLAN leads to HUNDREDS of bad lines of code and a bad line of RESEARCH to THOUSANDS; "research becomes hundreds" is not what it says.
Quote (CONFIRMED, line 328): "So you want to **focus human effort and attention** on the HIGHEST LEVERAGE parts of the pipeline."
Quote (CONFIRMED, line 333): "When you review the research and the plans, you get more leverage than you do when you review the code."
Quote (CONFIRMED, line 308): "Frequent Intentional Compaction via a research/plan/implement flow will make your performance **better**, but what makes it **good enough for hard problems** is that you build high-leverage human review into your pipeline."
Quote (CONFIRMED, line 339, note the exact wording is "on the page"): "I prefer Blake Smith's framing in Code Review Essentials for Software Teams, where he says the most important part of code review is mental alignment - keeping members of the team on the page as to how the code is changing and why."
Quote (CONFIRMED, line 357): "I can't read 2000 lines of golang daily. But I *can* read 200 lines of a well-written implementation plan."
Quote (CONFIRMED, line 314, the author's own failure case): "The tl;dr is that the research steps didn't go deep enough through the dependency tree, and assumed classes could be moved upstream without introducing deeply nested hadoop dependencies."
Supports: RPI divides work as agent researches/plans/implements while the human reviews the research and plan artifacts; the claimed payoff is leverage plus team "mental alignment"; the author also records that shallow research sank a 7-hour attempt, which is the same leverage argument in the negative.

## 2. Spec-driven development: critique and taxonomy

### 2.1 Birgitta Böckeler, "Understanding Spec-Driven-Development: Kiro, spec-kit, and Tessl"

URL: https://martinfowler.com/articles/exploring-gen-ai/sdd-3-tools.html
Author/org: Birgitta Böckeler (Thoughtworks), published on martinfowler.com.
Date: 15 October 2025.
Quote (CONFIRMED): "Spec-first: A well thought-out spec is written first, and then used in the AI-assisted development workflow for the task at hand."
Quote (CONFIRMED): "Spec-anchored: The spec is kept even after the task is complete, to continue using it for evolution and maintenance of the respective feature."
Quote (CONFIRMED): "Spec-as-source: The spec is the main source file over time, and only the spec is edited by the human, the human never touches the code."
Quote (CONFIRMED): "All SDD approaches and definitions I’ve found are spec-first, but not all strive to be spec-anchored or spec-as-source."
Quote (CONFIRMED, Kiro): "Workflow: Requirements → Design → Tasks"
Quote (CONFIRMED, small-bug test on Kiro): "it quickly became clear that the workflow was like using a sledgehammer to crack a nut. The requirements document turned this small bug into 4 “user stories” with a total of 16 acceptance criteria"
Quote (CONFIRMED, review burden): "spec-kit created a LOT of markdown files for me to review. They were repetitive, both with each other, and with the code that already existed. Some contained code already. Overall they were just very verbose and tedious to review."
Quote (CONFIRMED, the central disagreement with Horthy): "To be honest, I’d rather review code than all these markdown files. An effective SDD tool would have to provide a very good spec review experience."
Quote (CONFIRMED, agent compliance): "Even with all of these files and templates and prompts and workflows and checklists, I frequently saw the agent ultimately not follow all the instructions."
Quote (CONFIRMED, MDD parallel): "I wonder if spec-as-source, and even spec-anchoring, might end up with the downsides of both MDD and LLMs: Inflexibility and non-determinism."
Supports: the three-level SDD taxonomy; the critique that SDD tools shift review onto verbose, repetitive markdown and that a good spec REVIEW experience is the missing piece; that workflow weight must match problem size.

### 1.2 Horthy's later revision of RPI (QRSPI / "CRISPY")

Status: PRIMARY sources in Horthy's own words were found for the whole revision, including the thousand-line-plan argument (talk transcript, 1.2.b) and the earlier podcast (1.2.a).
Correction to the brief: his claim is not that humans failed to read ~1,000-line plans; it is that reading them was not leverage, because the plan is about as long as the code and the code then differs from the plan.

#### 1.2.a Dev Interrupted podcast, "Dex Horthy on Ralph, RPI, and escaping the 'Dumb Zone'"

URL: https://linearb.io/dev-interrupted/podcast/dex-horthy-humanlayer-rpi-methodology-ralph-loop
Author/org: LinearB's Dev Interrupted (host Andrew Zigler); the quotes are Dex Horthy speaking, from the page's published transcript.
Date: February 17, 2026 (JSON-LD datePublished 2026-02-17).
Caveat: this is a machine-style transcript with disfluencies, reproduced exactly as printed on the page.
Quote (CONFIRMED, at 00:27:44): "We definitely like did our version of like, Hey, we don't really read the code, we just read the plans and like if it works, then like we're, we're, we're Gucci. Like, I'll read the test and okay, if this test passes, then, then it's good enough and like actually backed away from that a little bit. We do think you should [00:28:00] read every line of the code, especially if you're working on like production systems, regulated industries, all this stuff."
Quote (CONFIRMED, at 00:28:04): "I think people frame the like too many PR slop problem and like, oh, we need to get AI to review the code. I'm like, well, AI wrote the code. We're all using the same fricking models. Uh, so I think the idea that we like is like, how do we minimize rework?"
Quote (CONFIRMED, at 00:28:25): "And the way we do that is like we move the alignment to a. Lighter weight, like part of the process. And so like the newest version of our tool generates this thing called the design discussion. And it's like a mini plan that is like, okay, here's the desired end state. Here's where we are now, here's what's outta scope."
Quote (CONFIRMED, at ~00:22:51): "the thing we say all the time is like, you cannot outsource the thinking."
Quote (CONFIRMED, at 00:24:00): "I would literally go to workshops with, you know, a hundred engineering teams and I would say like, okay, cool, during planning you wanna sprinkle in these magic words, otherwise you won't get as good results."
Supports: the originator of RPI walked back "review plans instead of code"; his revised position is read every line of code AND move alignment earlier into a shorter artifact (a "design discussion" with end state, current state, out-of-scope, relevant patterns and design questions).

#### 1.2.b Talk, "Everything We Got Wrong About Research-Plan-Implement - Dexter Horthy"

URL: https://www.youtube.com/watch?v=YwZR6tc7qYg (https://www.humanlayer.com/qrspi-mlops points at the same video).
Author/org: channel "AAIF Live"; the description says it is Horthy's keynote at the Coding Agents Conference at the Computer History Museum, March 3rd, 2026.
Date: uploadDate 2026-03-24.
Quote (CONFIRMED in the page's shortDescription; this is the organiser's abstract, NOT Horthy's own words): "RPI was supposed to fix AI coding, but Dexter Horthy says it kind of broke it, especially when teams started outsourcing thinking to agents, so now he’s pushing qrspi: fewer magic prompts, more structure, more human ownership, because the real problem isn’t the AI, it’s engineers trying not to think."
Transcript: retrieved with yt-dlp from YouTube's AUTO-GENERATED English captions (kind=asr), so every quote below is verbatim to the machine captions, not to a human-checked transcript; timestamps are approximate.
Quote (CONFIRMED in captions, ~00:02:23): "One thing that's very relevant if you've been on Twitter today is uh I don't think it's okay to not read the code. Uh, I also don't think you should read really long plan files. Uh, well those two are related."
Quote (CONFIRMED in captions, ~00:02:50): "Uh, there is no magic prompt. Uh, do not outsource the thinking. You the engineer are an important part of this process, and seek leverage."
Quote (CONFIRMED in captions, ~00:08:18): "we advocated for reading the plans that were output. This is me on stage in November telling people, you have to read the plan, otherwise it won't work. Um some people even would PR their plans and code review them together. But a thousand line plan tends to be about a thousand lines of code within 10% or so, and plans can have surprises."
Quote (CONFIRMED in captions, ~00:08:57): "So, the new advice, uh don't read the plans. Please, read the code. Uh just cuz it's it's the same amount of work and like look for leverage elsewhere"
Quote (CONFIRMED in captions, ~00:09:24): "We tried not reading the code for like 6 months. Uh it did not end well. We had to rip out and replace large parts of that system."
Quote (CONFIRMED in captions, ~00:15:03): "Because even if the plan is a thousand lines and the code is a thousand lines, your design discussion might only be 200 lines. And you get a lot of opportunities to restear in that moment."
Quote (CONFIRMED in captions, ~00:19:50): "I said don't review the plans, but these shorter docs are really, really good."
Quote (CONFIRMED in captions, ~00:21:36; the captions render the name as "crispy"): "the process is basically questions, research, design, structure outline, plan, work tree, implement, finally the pull request. Uh that didn't make a very good acronym though, so we just picked the ones we liked and uh we're calling this crispy."
Also in the captions: for "about 50% of people, maybe more" the planning prompt skipped the back-and-forth alignment unless the user typed the magic words "work back and forth with me starting with your open questions and outline before writing the plan".
Also in the captions: the old planning prompt had "85 instructions" and the split prompts are "all less than 40".
Supports: the originator of RPI now says review attention belongs on (1) a short ~200-line design discussion and structure outline BEFORE the plan, and (2) the code; the long plan is the artifact NOT worth human review.

#### 1.2.c Secondary write-up (not primary)

URL: https://alexlavaee.me/blog/from-rpi-to-qrspi/
Author/org: Alex Lavaee; date not found on the page (it references an interview dated Mar 26, 2026).
Its pull-quote "A 1,000-line plan contains as many surprises as 1,000 lines of code." does NOT appear in the talk captions; the nearest primary wording is the ~00:08:18 quote above, so treat the pull-quote as a paraphrase.
Its labels "instruction budget overflow, magic word dependencies, and a plan-reading illusion" are the blogger's, though "instruction budget" and "magic words" are Horthy's terms in the talk.
HumanLayer's own post https://www.humanlayer.dev/blog/skill-issue-harness-engineering-for-coding-agents was opened and contains nothing on QRSPI or plan length; the humanlayer.com/blog index showed no QRSPI post.

## 2. Spec-driven development (continued): the tools' own descriptions

### 2.2 GitHub Spec Kit

URL: https://github.blog/ai-and-ml/generative-ai/spec-driven-development-with-ai-get-started-with-a-new-open-source-toolkit/
Author/org: Den Delimarsky, GitHub.
Date: September 2, 2025.
Quote (CONFIRMED): "Your primary role is to steer; the coding agent does the bulk of the writing."
Quote (CONFIRMED): "Crucially, your role isn't just to steer. It's to verify. At each phase, you reflect and refine."
Quote (CONFIRMED): "instead of reviewing thousand-line code dumps, you, the developer, review focused changes that solve specific problems."
Supports: the agent writes every artifact and the human verifies at a checkpoint after each phase; the pitch is that review moves off large code diffs.

URL: https://raw.githubusercontent.com/github/spec-kit/main/spec-driven.md
Author/org: GitHub; undated.
Quote (CONFIRMED): "Both templates mandate the use of `[NEEDS CLARIFICATION]` markers:"
Quote (CONFIRMED): "1. **Mark all ambiguities**: Use [NEEDS CLARIFICATION: specific question]" and "2. **Don't guess**: If the prompt doesn't specify something, mark it".
Quote (CONFIRMED): "These checklists force the LLM to self-review its output systematically".
Also CONFIRMED in templates/spec-template.md: "System MUST authenticate users via [NEEDS CLARIFICATION: auth method not specified - email/password, SSO, OAuth?]".
Supports: the marker convention is flag-don't-guess, and agent self-review against a checklist precedes human review.

URL: https://raw.githubusercontent.com/github/spec-kit/main/templates/commands/clarify.md
Author/org: GitHub; undated.
Quote (CONFIRMED): "Identify underspecified areas in the current feature spec by asking up to 5 highly targeted clarification questions and encoding answers back into the spec."
Quote (CONFIRMED): "Maximum of 5 total questions across the whole session."
Supports: the demand on human attention is explicitly capped.

URL: https://raw.githubusercontent.com/github/spec-kit/main/README.md (main branch as fetched 2026-09-18).
Quote (CONFIRMED): "Constitution once per project; specify → plan → tasks → implement → converge per feature."
Quote (CONFIRMED, spans a line break in the source): "Invoke each `/speckit-*` **skill in your agent's chat**, one at a time, and review the result before continuing."
Note: the README has changed since 2025; commands are now `/speckit-*` skills, there is a `converge` step, and clarification is an optional quality gate.

### 2.3 AWS Kiro

URL: https://kiro.dev/blog/introducing-kiro/
Author/org: Nikhil Swaminathan and Deepak Singh, AWS/Kiro.
Date: July 14, 2025.
Quote (CONFIRMED): "Each user story includes EARS (Easy Approach to Requirements Syntax) notation acceptance criteria covering edge cases developers typically handle when building from basic user stories."
Quote (CONFIRMED): "This makes your prompt assumptions explicit, so you know Kiro is building what you want."
Quote (CONFIRMED): "Kiro then generates a design document by analyzing your codebase and approved spec requirements."
Supports: the agent expands one prompt into requirements, then design, then tasks; the human approves requirements before design.

URL: https://kiro.dev/docs/specs/
Author/org: AWS/Kiro; undated.
Quote (CONFIRMED): "All specs follow a three-phase workflow that transforms your idea into executable implementation:"
Quote (CONFIRMED): "you can also use Quick Spec to auto-generate all three artifacts without approval gates."
Quote (CONFIRMED, contiguous in the page payload): "Answering the checkpoint question sends your comments to the agent as one revision request."
Partially confirmed (assembled from fragments split by a code element, so treat as a paraphrase): at each phase checkpoint the CLI user presses Ctrl+X to read that phase's document and stage line comments.
Supports: approval gates are the default; the Kiro CLI now offers staged line comments on the phase document batched into one revision request, which is the closest vendor analogue to an anchored-comment review loop.

### 2.4 OpenSpec

URL: https://raw.githubusercontent.com/Fission-AI/OpenSpec/main/README.md
Author/org: Fission-AI; undated.
Quote (CONFIRMED): "Your AI writes these; you review the plan before any code is written."
Quote (CONFIRMED): "**Agree before you build** — human and AI align on specs before code gets written".
Quote (CONFIRMED): "**Work fluidly** — update any artifact anytime, no rigid phase gates".
The flow `/opsx:propose` → `/opsx:apply` → `/opsx:archive` and the heading "## ADDED Requirements" are confirmed; the word "delta" is not in the current README.
Supports: one change folder per change (proposal, specs, design, tasks), one human review before code, and an explicit rejection of rigid gates.

### 2.5 Tessl

URL: https://tessl.io/blog/tessl-launches-spec-driven-framework-and-registry
Author/org: Tessl (organization byline).
Date: datePublished 2025-09-23.
Quote (CONFIRMED): "These instructions live in the codebase as long-term memory, guiding agents as the app evolves and pairing with tests to enforce guardrails so existing functionality isn’t broken."
URL: https://tessl.io/blog/how-tessls-products-pioneer-spec-driven-development
Author/org: Tessl; byline not confirmed (first-person, likely Guy Podjarny); datePublished 2025-09-16.
Quote (CONFIRMED): "at the moment, using agents means endless reviewing and frustrating attempts to correct mistakes after the fact."
Quote (CONFIRMED): "You can review and edit the plan before they start executing it, and it will get updated as it executes, serving as an audit trail."
Not found: the phrase "spec-as-source" appears in neither Tessl post; it is Böckeler's label.
Supports: the agent writes plan, specs and tests; the human reviews and edits the plan before execution; tests rather than reading are the verification mechanism.

### 2.6 Further critiques of SDD

URL: https://marmelab.com/blog/2025/11/12/spec-driven-development-waterfall-strikes-back.html
Author/org: François Zaninotto, Marmelab.
Date: November 12, 2025.
Quote (CONFIRMED): "Markdown Madness: SDD produces too much text, especially in the design phase. Developers spend most of their time reading long Markdown files, hunting for basic mistakes hidden in overly verbose, expert-sounding prose. It’s exhausting."
Quote (CONFIRMED): "Double Code Review: The technical specification already contains code. Developers must review this code before running it, and since there will still be bugs, they’ll need to review the final implementation too. As a result, review time doubles."
Quote (CONFIRMED; the feature was displaying the current date, built with spec-kit): "resulting in 8 files and 1,300 lines of text"
Supports: the too-much-markdown and review-doubling critique.

URL: https://www.thoughtworks.com/radar/techniques/spec-driven-development
Author/org: Thoughtworks Technology Radar; ring Assess.
Date: Published Nov 05, 2025; the page says the blip is not on the current edition.
Quote (CONFIRMED): "some generate lengthy spec files that are hard to review, and when they produce PRDs or user stories, it's sometimes unclear who their intended user is. We may be relearning a bitter lesson — that handcrafting detailed rules for AI ultimately doesn’t scale."
SNIPPET ONLY: later radar text saying Thoughtworks teams have seen value in GitHub Spec-Kit and OpenSpec.

URL: https://martinfowler.com/fragments/2026-01-08.html
Author/org: Martin Fowler, quoting Kent Beck; Beck's original post was not fetched.
Date: 2026-01-08 (from URL and title).
Quote (CONFIRMED as quoted by Fowler): "The descriptions of Spec-Driven development that I have seen emphasize writing the whole specification before implementation. This encodes the (to me bizarre) assumption that you aren’t going to learn anything during implementation that would change the specification."
Supports: the waterfall critique; specs must stay revisable during implementation.

URL: https://martinfowler.com/articles/exploring-gen-ai/harness-engineering.html
Author/org: Birgitta Böckeler.
Date: 02 April 2026 (supersedes a memo dated 17 February 2026).
Quote (CONFIRMED): "it provides a feedback loop that self-corrects as many issues as possible before they even reach human eyes. Ultimately it should reduce the review toil".
Quote (CONFIRMED): "Correctness is outside any sensor's remit if the human didn't clearly specify what they wanted in the first place."
Supports: agent self-correction should precede human review; stating intent is the human's irreducible job.

URL: https://martinfowler.com/articles/exploring-gen-ai/context-engineering-coding-agents.html
Author/org: Birgitta Böckeler; 05 February 2026.
Quote (CONFIRMED): "there are no unit tests for context engineering".

## 3. Plan-mode and plan-review tooling

I spawned this sub-agent; its handback to me failed and its full report reached the lead directly.
The full sub-report is saved at docs/research-notes/ai-collab-design-distillation/q1-planmode-tooling.md and is not duplicated here.
It covers Anthropic's Claude Code best-practices page, Claude Code plan mode, Cursor Plan Mode, OpenAI Codex plan mode, Plannotator, revdiff, difit, tuicr and critique.
Headline from that report: vendor plan modes put review in chat or a whole-plan edit with approve / keep-planning and none offers line-anchored threads; Plannotator and revdiff add anchored annotations by hooking ExitPlanMode.
Headline from that report: the exit-code-10 convention is documented by revdiff only (opt-in via `--exit-code-on-annotations`); Plannotator's README documents no exit codes.
Headline from that report: the phrase "explore, plan, code, commit" is no longer on Anthropic's live page, whose heading is now "Explore first, then plan, then code"; the old anthropic.com/engineering URL redirects there.

## 4. Practitioner writing

I spawned this sub-agent as well; its report also reached the lead directly.
The full sub-report is saved at docs/research-notes/ai-collab-design-distillation/q1-practitioners.md and is not duplicated here.
It covers Simon Willison, Geoffrey Litt, Addy Osmani, Kent Beck and the Thoughtworks Technology Radar.
Headline from that report: Willison names review as the bottleneck ("the natural bottleneck on all of this is how fast I can review the results") and puts the first review pass on the author ("Your job is to deliver code you have proven to work.").
Headline from that report: Osmani states "When output increases faster than verification capacity, review becomes the rate limiter."
Headline from that report: Beck separates augmented coding ("you care about the code, its complexity, the tests, & their coverage") from vibe coding.
Headline from that report: the Radar holds "Complacency with AI-generated code" and notes that coding agents generate "larger change sets that are harder to review".

## 5. Empirical evidence on the review bottleneck

All quotes and numbers in this section were matched against tag-stripped raw page text by the sub-agent; I independently re-confirmed the METR 19%/24%/20% sentence and the Faros 91%/154%/98% figures.

### 5.1 METR

URL: https://metr.org/blog/2025-07-10-early-2025-ai-experienced-os-dev-study/ (paper: https://arxiv.org/abs/2507.09089).
Author/org: METR.
Date: July 10, 2025 (arXiv v1 12 Jul 2025, revised 25 Jul 2025).
Quote (CONFIRMED): "When developers are allowed to use AI tools, they take 19% longer to complete issues—a significant slowdown that goes against developer beliefs and expert forecasts."
Quote (CONFIRMED): "developers expected AI to speed them up by 24%, and even after experiencing the slowdown, they still believed AI had sped them up by 20%."
Quote (CONFIRMED, arXiv HTML): "we find that when AI is allowed, developers spend approximately 9% of their time reviewing and cleaning AI generated outputs when working with AI."
Also CONFIRMED in the arXiv HTML: developers accept much less than 44% of generations (raw text "<<44%").
Supports: time shifts from coding to prompting, waiting and reviewing; self-reported speedup is unreliable.

URL: https://metr.org/blog/2026-02-24-uplift-update/
Author/org: METR; February 24, 2026; "We are Changing our Developer Productivity Experiment Design".
Quote (CONFIRMED): "we believe that the data from our new experiment gives us an unreliable signal of the current productivity effect of AI tools."
Quote (CONFIRMED): "For the subset of the original developers who participated in the later study, we now estimate a speedup of -18% with a confidence interval between -38% and +9%. Among newly-recruited developers the estimated speedup is -4%, with a confidence interval between -15% and +9%."
Quote (CONFIRMED): "we believe it is likely that developers are more sped up from AI tools now — in early 2026 — compared to our estimates from early 2025."
Caveat: the 19% figure is an early-2025 snapshot and METR itself no longer treats it as current.
A further METR post exists (Joel Becker, May 11, 2026, a self-report survey at https://metr.org/blog/2026-05-11-ai-usage-survey/); title and date confirmed, contents not extracted.

### 5.2 DORA

URL: https://cloud.google.com/blog/products/ai-machine-learning/announcing-the-2025-dora-report
Author/org: Google Cloud / DORA; September 23, 2025.
Quote (CONFIRMED): "AI doesn't fix a team; it amplifies what's already there."
Quote (CONFIRMED): "90% of survey respondents report using AI at work. More than 80% believe it has increased their productivity. However, skepticism remains as 30% report little or no trust in the code generated by AI"
Quote (CONFIRMED): "Without robust control systems, like strong automated testing, mature version control practices, and fast feedback loops, an increase in change volume leads to instability"
URL: https://cloud.google.com/blog/products/devops-sre/announcing-the-2024-dora-report
Date: October 22, 2024.
Quote (CONFIRMED; the raw page has a double space after the first "estimated"): "As AI adoption increased, it was accompanied by an estimated decrease in delivery throughput by 1.5%, and an estimated reduction in delivery stability by 7.2%."
URL: https://dora.dev/insights/balancing-ai-tensions/
Date: Mar 10, 2026.
Quote (CONFIRMED): "The verification tax: Time saved writing is often re-spent auditing."
No 2026 annual DORA report was found as of 2026-09-18; an interim "ROI of AI-assisted Software Development" report exists but is a gated PDF and was not read.

### 5.3 Faros AI, "AI Productivity Paradox"

URL: https://www.faros.ai/blog/ai-software-engineering
Author/org: Faros AI; datePublished 2025-07-23; sample described as 10,000 developers across 1,255 teams.
Quote (CONFIRMED, re-checked by me): "Developers on teams with high AI adoption complete 21% more tasks and merge 98% more pull requests, but PR review time increases 91%, revealing a critical bottleneck: human approval."
Quote (CONFIRMED, re-checked by me): "AI adoption is consistently associated with a 9% increase in bugs per developer and a 154% increase in average PR size."
Supports: the most direct quantitative evidence that human approval is the bottleneck; note that Faros sells engineering-analytics software.

### 5.4 Other quantitative sources

LinearB benchmarks (https://linearb.io/resources/software-engineering-benchmarks-report, undated): CONFIRMED "AI PRs wait 4.6x longer before review – but are reviewed 2x faster once picked up." and "Acceptance Rates for AI-generated PRs are significantly lower than manual PRs (32.7% vs. 84.4%)."
Sonar press release (https://www.sonarsource.com/company/press-releases/sonar-data-reveals-critical-verification-gap-in-ai-coding/, January 8, 2026): CONFIRMED "while 96% of developers report they do not fully trust that AI-generated code is functionally correct, only 48% state they always check their AI-assisted code before committing it."
Sonar, same page: CONFIRMED "38% of developers noting that reviewing AI-generated code requires more effort than reviewing code written by their human colleagues."
Stack Overflow 2025 survey (https://survey.stackoverflow.co/2025/ai): CONFIRMED "More developers actively distrust the accuracy of AI tools (46%) than trust it (33%), and only a fraction (3%) report "highly trusting" the output."
Stack Overflow blog (https://stackoverflow.blog/2025/07/29/developers-remain-willing-but-reluctant-to-use-ai-the-2025-developer-survey-results-are-here/): CONFIRMED "66% of developers say they are spending more time fixing "almost-right" AI-generated code."
CodeRabbit (https://www.coderabbit.ai/blog/state-of-ai-vs-human-code-generation-report, David Loker, December 17, 2025): CONFIRMED "Across 470 PRs, AI-authored changes produced 10.83 issues per PR, compared to 6.45 for human-only PRs."
Cortex (https://www.cortex.io/post/ai-is-making-engineering-faster-but-not-better-state-of-ai-benchmark-2026, Ganesh Datta, November 12, 2025): CONFIRMED "Incidents per pull request increased by 23.5%, and change failure rates are up approximately 30%."
GitClear (https://www.gitclear.com/ai_assistant_code_quality_2025_research, undated): CONFIRMED "We observe a spike in the prevalence of duplicate code blocks, along with increases in short-term churn code"; detailed percentages are in a gated PDF and UNCONFIRMED.
Uplevel (https://uplevelteam.com/blog/ai-for-developer-productivity, 2024-10-18): CONFIRMED "The group using Copilot introduced 41% more bugs".
Counter-evidence, Google RCT (arXiv 2410.12944, 16 Oct 2024): CONFIRMED "stands at about 21%, although our confidence interval is large."
Counter-evidence, GitHub Copilot RCT (arXiv 2302.06590): CONFIRMED "completed the task 55.8% faster than the control group"; single greenfield task.
Most of the sources in 5.4 are vendors with a product interest in the finding.

### 5.5 Comprehension debt, cognitive debt, over-trust

Duma et al., "These Aren't the Reviews You're Looking For" (https://arxiv.org/abs/2605.02273, submitted 4 May 2026): CONFIRMED "most AI-generated PRs receive no review and, when reviewed, are largely dominated by AI agents rather than humans."
Same paper: CONFIRMED "61.38% (20,621) receive no recorded review activity" out of 33,596 agent PRs; caveat that in the same-repo subset agent PRs went unreviewed 28.92% of the time versus 34.52% for human PRs.
Perry et al., Stanford (https://arxiv.org/abs/2211.03622, 7 Nov 2022): CONFIRMED "participants with access to an AI assistant were more likely to believe they wrote secure code than those without access".
Anthropic, "How AI Impacts Skill Formation" (https://www.anthropic.com/research/AI-assistance-coding-skills, Jan 29, 2026; arXiv 2601.20245): CONFIRMED "the AI group averaged 50% on the quiz, compared to 67% in the hand-coding group" and "The largest gap in scores between the two groups was on debugging questions".
Margaret-Anne Storey (https://margaretstorey.com/blog/2026/02/09/cognitive-debt/): CONFIRMED "the debt compounded from going fast lives in the brains of the developers" and "no one on the team could explain why certain design decisions had been made".
Addy Osmani (https://addyosmani.com/blog/comprehension-debt/, March 14, 2026): CONFIRMED "a junior engineer can now generate code faster than a senior engineer can critically audit it."
MIT Media Lab, "Your Brain on ChatGPT" (arXiv 2506.08872, 10 Jun 2025): title only confirmed; it studies essay writing, not code.
Not found: a primary study that directly measures human rubber-stamping ("LGTM") of AI-generated PRs; Duma et al. is the closest.

## 6. Disagreements between sources

Horthy (Aug 2025) says reviewing research and plans gives more leverage than reviewing code; Böckeler (Oct 2025) says "I’d rather review code than all these markdown files"; Zaninotto says spec review plus code review means "review time doubles".
Horthy (Feb-Mar 2026) reverses himself: "don't read the plans. Please, read the code", because a thousand-line plan costs as much to review as the code and the code still differs; he keeps upstream review but moves it to a ~200-line design discussion.
So by 2026 Horthy and Böckeler agree that long agent-written markdown is a poor review surface; they differ in that Horthy still wants a short pre-code alignment artifact reviewed by a human.
Spec Kit and Kiro default to a gate after every phase; OpenSpec advertises "no rigid phase gates"; Beck and Fowler argue that writing the whole spec first assumes nothing is learned during implementation.
METR's 2025 RCT found a 19% slowdown, while the Google and GitHub RCTs found 21% and 55.8% speedups on fixed tasks; METR's own 2026 update calls its newer data unreliable and expects the true effect is now more positive.
DORA 2024 associated AI adoption with lower throughput; DORA 2025-2026 associates it with higher throughput and higher instability.
Böckeler's harness-engineering piece and Anthropic's docs push verification onto tests and fresh-context agent reviewers before humans; Horthy objects that "AI wrote the code. We're all using the same fricking models."
