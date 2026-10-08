# Q2 scratch: distilling a technical design for a time-poor reviewer

Accessed 2026-09-18.
Question: how do engineers distill and present a design so a time-poor human can judge it, and what does practice say about (a) a short human-authored framing written BEFORE the detailed work and (b) layered or tiered reading depth over a single document.
Method: every quote marked VERIFIED was matched by `curl` plus a text grep against the live page on the access date.
No quote in this file came through a summarizing fetch model; WebSearch was used only to discover URLs.
Quotes marked SNIPPET ONLY were seen only in a search-result snippet.
Curly quotes and dashes in the originals are kept as found.
Spaces that the HTML-to-text step inserted at tag boundaries before punctuation were removed from quote lines; no other characters were changed.

## 1. Pre-AI design-communication practice

### 1.1 Amazon six-page narrative (Bezos)

- URL: https://www.aboutamazon.com/news/company-news/2017-letter-to-shareholders
- Author/org: Jeff Bezos, Amazon (2017 letter to shareholders).
- Date: 2018 (letter covering fiscal 2017; the exact publication day was not verified on the page).
- Quote 1 (VERIFIED): "We don’t do PowerPoint (or any other slide-oriented) presentations at Amazon. Instead, we write narratively structured six-page memos. We silently read one at the beginning of each meeting in a kind of “study hall.”"
- Quote 2 (VERIFIED): "they mistakenly believe a high-standards, six-page memo can be written in one or two days or even a few hours, when really it might take a week or more!"
- Quote 3 (VERIFIED): "The great memos are written and re-written, shared with colleagues who are asked to improve the work, set aside for a couple of days, and then edited again with a fresh mind."
- Supports: narrative over slides, a fixed length cap, and the claim that the cost of a short document is human rewriting time, not word count.
- Note: the reading happens IN the meeting, which is Amazon's answer to reviewers who do not pre-read.

### 1.2 Google design docs (Malte Ubl)

- URL: https://www.industrialempathy.com/posts/design-docs-at-google/
- Author/org: Malte Ubl (then Google).
- Date: 2020-07-06.
- Quote 1 (VERIFIED): "Design docs should be sufficiently detailed but short enough to actually be read by busy people. The sweet spot for a larger project seems to be around 10-20ish pages."
- Quote 2 (VERIFIED): "It should also be noted that it is absolutely possible to write a 1-3 page “mini design doc”."
- Quote 3 (VERIFIED): "This section gives the reader a very rough overview of the landscape in which the new system is being built and what is actually being built. This isn’t a requirements doc. Keep it succinct!"
- Supports: length is set by the reviewer's attention, a context-and-scope opener precedes detail, and non-goals are first-class.
- Note: the doc is written by the authors "before they embark on the coding project", so the whole doc is the before-work artifact, not only its opening.

### 1.3 Rust RFC template

- URL: https://raw.githubusercontent.com/rust-lang/rfcs/master/0000-template.md
- Author/org: Rust project.
- Date: undated (living template).
- Quote 1 (VERIFIED): "One paragraph explanation of the feature."
- Quote 2 (VERIFIED): "Explain the proposal as if it was already included in the language and you were teaching it to another Rust programmer."
- Quote 3 (VERIFIED): "This is the technical portion of the RFC. Explain the design in sufficient detail that:"
- Supports: tiering inside one document by design, as Summary (one paragraph), then Motivation, then Guide-level explanation, then Reference-level explanation.
- Note: these are different sections for different depths, not a short copy of the long one.

### 1.4 Oxide RFD 1

- URL: https://rfd.shared.oxide.computer/rfd/0001
- Author/org: Jess Frazelle and Bryan Cantrill, Oxide Computer Company.
- Date: undated on page.
- Quote 1 (VERIFIED): "Writing down ideas is important: it allows them to be rigorously formulated (even while nascent), candidly discussed and transparently shared."
- Quote 2 (VERIFIED, quoting RFC 3): "Notes are encouraged to be timely rather than polished."
- Supports: writing as the formulation step, and a bias toward short early documents over polished late ones.

### 1.5 Uber RFCs (Gergely Orosz)

- URL: https://blog.pragmaticengineer.com/scaling-engineering-teams-via-writing-things-down-rfcs/
- Author/org: Gergely Orosz, The Pragmatic Engineer.
- Date: 2018-10-03, last updated 2022-09-21.
- Quote 1 (VERIFIED): "Capture this plan in a short, written document. Once it is clear to the team how and what you do, it should be relatively quick to write down the "how". Don't go overboard."
- Quote 2 (VERIFIED): "Have a few, select people approve this plan before starting work."
- Quote 3 (VERIFIED): "People who review many engineering proposals often had the same type of questions."
- Supports: a short plan approved before work starts, and templates that grow out of recurring reviewer questions.

### 1.6 Architecture Decision Records (Michael Nygard)

- URL: https://www.cognitect.com/blog/2011/11/15/documenting-architecture-decisions
- Author/org: Michael Nygard, Cognitect.
- Date: 2011-11-15.
- Quote 1 (VERIFIED): "Large documents are never kept up to date. Small, modular documents have at least a chance at being updated. Nobody ever reads large documents, either."
- Quote 2 (VERIFIED): "The whole document should be one or two pages long."
- Quote 3 (VERIFIED): "Bullets are acceptable only for visual style, not as an excuse for writing sentence fragments."
- Supports: a hard size cap as a readability and freshness device, and prose over fragments.

### 1.7 Shape Up pitch (Ryan Singer, Basecamp)

- URL: https://basecamp.com/shapeup/1.5-chapter-06
- Author/org: Ryan Singer, Basecamp.
- Date: 2019 (year from memory; not shown on the page).
- Extraction note: quotes 1 and 2 come from a bulleted list flattened to one line, so the line breaks between ingredients are lost.
- Quote 1 (VERIFIED): "There are five ingredients that we always want to include in a pitch: Problem — The raw idea, a use case, or something we’ve seen that motivates us to work on this Appetite — How much time we want to spend and how that constrains the solution"
- Quote 2 (VERIFIED): "Rabbit holes — Details about the solution worth calling out to avoid problems No-gos — Anything specifically excluded from the concept"
- Quote 3 (VERIFIED): "the first step for presenting a pitch is posting the write-up with all the ingredients above somewhere that stakeholders can read it on their own time. This keeps the betting table short and productive."
- Supports: a short human-authored framing, with an explicit budget (appetite) and exclusions, written before the build and read asynchronously by the decision makers.
- Nuance: the pitch is written AFTER shaping, which chapter 2 (https://basecamp.com/shapeup/1.1-chapter-02) calls "a closed-door, creative process", and BEFORE building.
- Nuance: so the pitch distills thinking the human already did; it is not written cold.

### 1.8 C4 model (Simon Brown)

- URL: https://c4model.com/diagrams
- Author/org: Simon Brown.
- Date: undated.
- Quote 1 (VERIFIED): "The different levels of zoom allow you to tell different stories to different audiences."
- Quote 2 (VERIFIED): "you don’t need to use all 4 levels of diagram; only those that add value - the system context and container diagrams are sufficient for most software development teams."
- Supports: tiered depth as zoom levels over one model, and a warning that the deepest tiers are often not worth producing.

### 1.9 Diátaxis (Daniele Procida)

- URL: https://diataxis.fr/
- Author/org: Daniele Procida.
- Date: undated.
- Quote 1 (VERIFIED): "Diátaxis identifies four distinct needs, and four corresponding forms of documentation - tutorials, how-to guides, technical reference and explanation."
- Supports: only weakly relevant; it splits documentation by reader need into separate documents, which is the opposite move from tiering one document by depth.

### 1.10 Amazon PR/FAQ "working backwards" (Bryar and Carr)

- URL: https://workingbackwards.com/concepts/working-backwards-pr-faq-process/
- Author/org: Colin Bryar and Bill Carr (former Amazon executives, authors of "Working Backwards").
- Date: undated.
- Quote 1 (VERIFIED): "a useful working backwards tool: writing the press release and FAQ before you build the product."
- Quote 2 (VERIFIED): "Writing a press release is a forcing function to ensure that the creator of the new product idea is focused on the customer."
- Quote 3 (VERIFIED): "Early on, the document is a solo effort, and the author is typically a product manager. In early drafts, the PR/FAQ typically has many holes and questions that haven’t been answered yet, requiring the author(s) to research to fill in the blanks. The fidelity and detail increase with each draft."
- Supports: the strongest precedent for a short human-written framing produced FIRST, with the short page (the PR) leading and the long part (the FAQ) growing under it.
- Nuance: the framing is revised many times as research fills holes, so "written first" does not mean "frozen first".

### 1.11 BLUF (US Army)

- URL: https://en.wikipedia.org/wiki/BLUF_(communication)
- Author/org: Wikipedia, quoting Army Regulation 25-50.
- Date: undated (wiki).
- Quote 1 (VERIFIED on the wiki page, SECONDARY for the regulation): "Army writing will be concise, organized, and to the point. Two essential requirements include putting the main point at the beginning of the correspondence (bottom line up front) and using the active voice"
- Supports: conclusion-first ordering as a mandated standard for time-poor decision makers.
- Not found: the primary AR 25-50 PDF on armypubs.army.mil returned 404 at the URL tried, so the regulation text is quoted second-hand.

### 1.12 Minto pyramid principle

- URL: https://www.barbaraminto.com/
- Author/org: Barbara Minto, Minto Books International.
- Date: undated.
- Quote 1 (VERIFIED): "The Minto Pyramid Principle says that your thinking will be easy for a reader to grasp if you present the ideas organized as a pyramid under a single point."
- Quote 2 (SNIPPET ONLY, attributed to Minto by https://www.linkedin.com/pulse/whats-so-what-pyramid-principle-barbara-minto-james-raybould): "Ideas at any level in the pyramid must always be summaries of the ideas grouped below them."
- Supports: a layered document where each upper layer is a true summary of the layer below, which is tiering as a logical structure rather than a formatting trick.

### 1.13 Inverted pyramid (journalism) via NN/g

- URL: https://www.nngroup.com/articles/inverted-pyramid/
- Author/org: Amy Schade, Nielsen Norman Group.
- Date: 2018-02-11.
- Quote 1 (VERIFIED): "Start content with the most important piece of information so readers can get the main point, regardless of how much they read."
- Quote 2 (VERIFIED): "Readers can stop reading at any point on the page and still come away with the main point."
- Supports: the exact property a tiered doc wants, namely that stopping early still yields the decision.
- Related (VERIFIED, https://en.wikipedia.org/wiki/Inverted_pyramid_(journalism)): "This is called "cutting from the bottom.""

### 1.14 Abstracts and structured abstracts

- URL: https://www.nlm.nih.gov/bsd/policy/structured_abstracts.html
- Author/org: US National Library of Medicine.
- Date: undated.
- Quote 1 (VERIFIED): "A structured abstract is an abstract with distinct, labeled sections (e.g., Introduction, Methods, Results, Discussion) for rapid comprehension"
- Quote 2 (VERIFIED): "They also guide authors in summarizing the content of their manuscripts precisely, facilitate the peer-review process for manuscripts submitted for publication"
- Supports: a labeled, fixed-shape short layer speeds both reading and review.
- See section 5 for the counterpoint that abstracts are normally derived from the finished work.

## 2. Evidence on reading behaviour and review effectiveness

### 2.1 How little people read (NN/g)

- URL: https://www.nngroup.com/articles/how-little-do-users-read/
- Author/org: Jakob Nielsen, Nielsen Norman Group.
- Date: 2008-05-05.
- Quote 1 (VERIFIED): "On the average Web page, users have time to read at most 28% of the words during an average visit; 20% is more likely."
- Supports: a realistic prior that most of a long document goes unread.
- Caveat: this is web-page browsing data, not engineers reviewing a design they are accountable for.

### 2.2 Scanning, not reading (NN/g)

- URL: https://www.nngroup.com/articles/how-users-read-on-the-web/
- Author/org: Jakob Nielsen, Nielsen Norman Group.
- Date: 1997-09-30.
- Quote 1 (VERIFIED): "79 percent of our test users always scanned any new page they came across; only 16 percent read word-by-word."
- Quote 2 (VERIFIED, from the recommended-practice list): "the inverted pyramid style, starting with the conclusion" and "half the word count (or less) than conventional writing"
- Supports: conclusion-first plus word caps as the evidence-based response to scanning.

### 2.3 F-shaped pattern (NN/g)

- URL: https://www.nngroup.com/articles/f-shaped-pattern-reading-web-content/
- Author/org: Kara Pernice, Nielsen Norman Group.
- Date: 2017-11-12 (page says last reviewed 2026-08-19).
- Quote 1 (VERIFIED): "First lines of text on a page receive more gazes than subsequent lines of text on the same page."
- Quote 2 (VERIFIED): "When people scan in an F-shape, they miss big chunks of content based merely on how text flows in a column. The skipped phrases and words are often as important — or even more important — as those words that are read. But users won’t realize this, since by definition they don’t know what they don’t see."
- Quote 3 (VERIFIED): "Include the most important points in the first two paragraphs on the page."
- Supports: reviewers skipping content do not know what they skipped, so the document must put decision-critical content where the gaze lands.

### 2.4 Progressive disclosure (NN/g)

- URL: https://www.nngroup.com/articles/progressive-disclosure/
- Author/org: Jakob Nielsen, Nielsen Norman Group.
- Date: 2006-12-03.
- Quote 1 (VERIFIED): "Initially, show users only a few of the most important options. Offer a larger set of specialized options upon request."
- Quote 2 (VERIFIED): "You must get the right split between initial and secondary features. You have to disclose everything that users frequently need up front, so that they have to progress to the secondary display only on rare occasions."
- Quote 3 (VERIFIED, CONTRADICTS deep tiering): "In practice, designs that go beyond 2 disclosure levels typically have low usability because users often get lost when moving between the levels."
- Supports: layering works, but the first layer must be sufficient for the common case, and more than two levels is a known usability failure.
- Caveat: the guidance is about UI features, applied to documents by analogy.

### 2.5 Code review size (SmartBear / Cisco)

- URL: https://smartbear.com/learn/code-review/best-practices-for-peer-code-review/
- Author/org: SmartBear (vendor summary of its Cisco Systems study).
- Date: undated.
- Quote 1 (VERIFIED): "A SmartBear study of a Cisco Systems programming team revealed that developers should review no more than 200 to 400 lines of code (LOC) at a time. The brain can only effectively process so much information at a time; beyond 400 LOC, the ability to find defects diminishes."
- Quote 2 (VERIFIED): "Authors should annotate code before the review occurs because annotations guide the reviewer through the changes, showing which files to look at first and defending the reason behind each code modification."
- Supports: review effectiveness falls with artifact size, and author-written orientation improves review.
- Caveat: vendor-published, about code and not prose; the original study report was not fetched.

### 2.6 Google small CLs

- URL: https://google.github.io/eng-practices/review/developer/small-cls.html
- Author/org: Google engineering practices.
- Date: undated.
- Quote 1 (VERIFIED): "It’s easier for a reviewer to find five minutes several times to review small CLs than to set aside a 30 minute block to review one large CL."
- Quote 2 (VERIFIED): "With large changes, reviewers and authors tend to get frustrated by large volumes of detailed commentary shifting back and forth—sometimes to the point where important points get missed or dropped."
- Quote 3 (VERIFIED): "100 lines is usually a reasonable size for a CL, and 1000 lines is usually too large"
- Related (VERIFIED, https://google.github.io/eng-practices/review/developer/cl-descriptions.html): "The first line of a CL description should be a short summary of specifically what is being done by the CL"
- Supports: small review units and an author-written first-line summary as standing practice.

### 2.7 Comprehension is the bottleneck in review (Bacchelli and Bird)

- URL: https://www.microsoft.com/en-us/research/publication/expectations-outcomes-and-challenges-of-modern-code-review/
- Author/org: Alberto Bacchelli and Christian Bird, ICSE 2013.
- Date: 2013 (ICSE 2013).
- Quote 1 (VERIFIED, abstract): "we find that code and change understanding is the key aspect of code reviewing and that developers employ a wide range of mechanisms to meet their understanding needs, most of which are not met by current tools."
- Supports: the reviewer's scarce resource is understanding, which is what a framing layer is supposed to buy.

### 2.8 Research on DESIGN-DOC review specifically

- Not found: no empirical study was located that measures design-document length against reviewer comprehension or decision quality.
- One search surfaced the claim that this area is understudied relative to code review, but only as a search-engine synthesis (SNIPPET ONLY, no citable source).
- Sadowski et al. 2018 "Modern Code Review: A Case Study at Google" was not fetched in this pass.
- Consequence: every length number for design docs above (Ubl's 10-20 pages, Nygard's 1-2 pages, Bezos's 6 pages) is practitioner convention, not measured.

## 3. AI-era specifics

### 3.1 Spec-driven development output is too verbose to review (Böckeler, Thoughtworks)

- URL: https://martinfowler.com/articles/exploring-gen-ai/sdd-3-tools.html
- Author/org: Birgitta Böckeler, Thoughtworks, on martinfowler.com.
- Date: 2025-10-15.
- Quote 1 (VERIFIED): "spec-kit created a LOT of markdown files for me to review. They were repetitive, both with each other, and with the code that already existed. Some contained code already. Overall they were just very verbose and tedious to review."
- Quote 2 (VERIFIED): "To be honest, I’d rather review code than all these markdown files. An effective SDD tool would have to provide a very good spec review experience."
- Quote 3 (VERIFIED, about Kiro, whereas quotes 1 and 2 concern spec-kit): "The requirements document turned this small bug into 4 “user stories” with a total of 16 acceptance criteria"
- Supports: the core complaint, from a credible practitioner, that agent-written specs fail at the review step and that the review experience is the missing product.

### 3.2 "Markdown Madness" (Zaninotto, Marmelab)

- URL: https://marmelab.com/blog/2025/11/12/spec-driven-development-waterfall-strikes-back.html
- Author/org: François Zaninotto, Marmelab.
- Date: 2025-11-12.
- Quote 1 (VERIFIED): "SDD produces too much text, especially in the design phase. Developers spend most of their time reading long Markdown files, hunting for basic mistakes hidden in overly verbose, expert-sounding prose. It’s exhausting."
- Quote 2 (VERIFIED): "a developer wanted to display the current date on a time-tracking app, resulting in 8 files and 1,300 lines of text"
- Quote 3 (VERIFIED): "Specs contain many repetitions, imaginary corner cases, and overkill refinements."
- Supports: padding and invented detail are the named failure modes.
- Also contradicts: the essay's thesis is that heavy up-front documents are waterfall, which cuts against adding more document structure at all.

### 3.3 Plan mode verbosity (Claude Code issue)

- URL: https://github.com/anthropics/claude-code/issues/12337
- Author/org: GitHub user HowardXieh.
- Date: 2025-11-25.
- Quote 1 (VERIFIED): "The new plan mode is too verbose. It basically tries to implement everything in the plan file, and it keeps asking the same questions repeatedly."
- Supports: a single user report, weak on its own, that the same complaint applies to agent plans.

### 3.4 The comprehension bottleneck (Nita)

- URL: https://andreinita.co/blog/comprehension-bottleneck-ai-docs/
- Author/org: Andrei Nita (personal blog).
- Date: June 2026.
- Quote 1 (VERIFIED): "We optimized for output volume and forgot about review surface area: the amount of information a human can comprehend per unit of time. The bottleneck is no longer generation. It is comprehension."
- Supports: framing of the problem.
- Also contradicts: the post argues the fix is visual review, not more or better-layered text.

### 3.5 Agent PR descriptions are unreliable (Gong et al.)

- URL: https://arxiv.org/abs/2601.04886
- Author/org: Jingzhi Gong and three co-authors (arXiv preprint).
- Date: 2026-01-08, revised 2026-01-26.
- Quote 1 (VERIFIED, abstract): "We contributed 974 manually annotated PRs, found 406 PRs (1.7%) exhibited high PR-MCI, and identified eight PR-MCI types, revealing that "descriptions claim unimplemented changes" was the most common issue (45.4%)."
- Quote 2 (VERIFIED, abstract): "high-MCI PRs had 51.7% lower acceptance rates (28.3% vs. 80.0%) and took 3.5 times longer to merge (55.8 vs. 16.0 hours)."
- Supports: an agent-written summary of the agent's own work can assert things the work does not contain, and that costs reviewer trust and time.
- Caveat: only 1.7% of 23,247 PRs were high-inconsistency, so the measured rate is low.

### 3.6 The rationale gap in agent PRs (Tian Pan)

- URL: https://tianpan.co/blog/2026/05/31/the-pr-description-your-coding-agent-cannot-write
- Author/org: Tian Pan (personal blog).
- Date: 2026-05-31.
- Quote 1 (VERIFIED): "it describes the destination, not the journey, because the journey has already fallen out of the window."
- Quote 2 (VERIFIED): "The code was fine. The reviewer was conscientious. The agent did exactly what it was asked. The artifact between them — the pull request — was empty of everything that would have caught the mistake."
- Quote 3 (VERIFIED, second-hand statistic): "Faros AI's 2026 telemetry across 22,000 developers found that teams with heavy AI adoption merge 98% more pull requests, but the PRs are 154% larger and take 91% longer to review."
- Supports: the "why" is the part an agent summarizing its own output cannot supply, which is the argument for the human owning the intent layer.
- Caveat: the Faros numbers are quoted from this blog, not from Faros.

### 3.7 Same-model summaries and self-preference (Panickssery, Bowman, Feng)

- URL: https://arxiv.org/abs/2404.13076
- Author/org: Arjun Panickssery, Samuel R. Bowman, Shi Feng (venue not verified on the page).
- Date: 2024-04-15 (arXiv).
- Quote 1 (VERIFIED, abstract): "new biases are introduced due to the same LLM acting as both the evaluator and the evaluatee. One such bias is self-preference, where an LLM evaluator scores its own outputs higher than others' while human annotators consider them of equal quality."
- Supports: by analogy only, a model judging or distilling its own draft is not an independent check.
- Caveat: the paper is about evaluation scores, not about summaries; no source was found that tests "summary by the authoring model" directly.

### 3.8 LLM summaries hallucinate and omit

- URL: https://arxiv.org/abs/2410.13961
- Author/org: Catarina G. Belem and five co-authors.
- Date: 2024-10-17, revised 2025-04-26 (arXiv).
- Quote 1 (VERIFIED, abstract): "on average, up to 75% of the content in LLM-generated summary is hallucinated, with hallucinations more likely to occur towards the end of the summaries."
- Supports: machine summaries are not a safe substitute for the body.
- Caveat: multi-document summarization with 2024 models; "up to" is a worst case.
- URL: https://arxiv.org/abs/2605.24137
- Author/org: Hinduja Nirujan and four co-authors.
- Date: 2026-05-22.
- Quote 2 (VERIFIED, abstract): "these models frequently produce hallucinations that can be convincing but unsupported by the source report. This can mislead developers and reduce trust in automated maintenance"
- Supports: the same problem in a software-engineering summarization task in 2026.
- Not found: a source specifically about AI summaries going STALE as the underlying doc changes; Nygard's "Large documents are never kept up to date" (1.6) is the nearest, and it predates LLMs.

### 3.9 Workslop (HBR)

- URL: https://hbr.org/2025/09/ai-generated-workslop-is-destroying-productivity
- Author/org: BetterUp Labs and Stanford Social Media Lab researchers, Harvard Business Review.
- Date: September 2025.
- Quote 1 (SNIPPET ONLY; page is paywalled to curl): "AI-generated content that masquerades as good work but lacks the substance to advance a task"
- Supports: the cost of padded AI output lands on the downstream reader.
- Contradicting note: https://pivot-to-ai.com/2025/09/23/workslop-bad-study-but-an-excellent-word/ criticises the study's method (title seen in search only, SNIPPET ONLY).

### 3.10 Tools that layer agent output

- CodeRabbit walkthrough, URL: https://docs.coderabbit.ai/pr-reviews/walkthroughs , CodeRabbit docs, undated.
- Quote 1 (VERIFIED): "Every time CodeRabbit reviews a pull request, it posts a walkthrough comment — a structured overview of the changes that appears at the top of the PR comment thread, separate from inline code comments."
- Quote 2 (VERIFIED): "A short poem at the end of the walkthrough, generated from the changeset. Can be disabled if you prefer a strictly professional output."
- Supports: a shipped two-layer presentation (overview, then inline) that is fully machine-written, with default-on filler that illustrates the padding complaint.
- CodeRabbit slop detection, URL: https://docs.coderabbit.ai/pr-reviews/slop-detection , undated.
- Quote 3 (VERIFIED): "Automatically detect low-quality, AI-generated ‘Slop’ pull requests on GitHub repositories."
- Supports: slop is now a product category a review vendor ships against.
- GitHub Copilot PR summary, URL: https://docs.github.com/en/copilot/how-tos/use-copilot-for-common-tasks/create-a-pr-summary , GitHub Docs, undated.
- Quote 4 (VERIFIED): "Generate an AI-powered summary of your pull request changes to help reviewers quickly understand what you changed and why."
- Supports: the vendor claims the summary conveys "why", which is exactly what 3.6 argues a diff-derived summary cannot know.
- Kiro specs, URL: https://kiro.dev/docs/specs/ , AWS Kiro docs, undated.
- Quote 5 (VERIFIED, with two elisions marked by ellipses that are mine): "Every spec generates three key files that form the foundation of your specification: requirements.md (or bugfix.md)... design.md... tasks.md"
- Kiro Quick Spec, URL: https://kiro.dev/docs/specs/quick-spec/ , page updated 2026-08-04.
- Quote 6 (VERIFIED): "Instead of reviewing and approving each artifact, you front-load your input so Kiro has enough context to produce high-quality specs autonomously."
- Quote 7 (VERIFIED): "For features where requirements quality is critical - compliance-sensitive domains, high-stakes systems, unfamiliar territory - a standard Feature Spec with explicit review gates is usually the better fit."
- Supports: Kiro's own answer to review fatigue is to move human input to the FRONT (clarifying questions) and drop the per-artifact gates, which is a vendor-level endorsement of human-first framing and a partial retreat from staged review.
- Not found: Devin, Cursor, and Graphite documentation of tiered presentation was not fetched in this pass, and no independent critique of CodeRabbit walkthroughs was located.

### 3.11 Who writes the brief: human first, AI expands (Osmani)

- URL: https://addyosmani.com/blog/good-spec/
- Author/org: Addy Osmani (also published on O'Reilly Radar).
- Date: 2026-01-13.
- Quote 1 (VERIFIED): "Kick off your project with a concise high-level spec, then have the AI expand it into a detailed plan."
- Quote 2 (VERIFIED): "Treat this as a “product brief” and let the agent generate a more elaborate spec from it. This leverages the AI’s strength in elaboration while you maintain control of the direction."
- Supports: the human-writes-intent, AI-writes-detail split, stated as a recommendation.
- Caveat: the brief here is aimed at steering the AGENT, not at making the result reviewable by a human.

## 4. Writing as thinking, and the risk of outsourcing it

### 4.1 Paul Graham

- URL: https://paulgraham.com/writes.html
- Author/org: Paul Graham.
- Date: October 2024.
- Quote 1 (VERIFIED): "To write well you have to think clearly, and thinking clearly is hard."
- Quote 2 (VERIFIED): "writing is thinking. In fact there's a kind of thinking that can only be done by writing."
- Quote 3 (VERIFIED): "Almost all pressure to write has dissipated. You can have AI do it for you, both in school and at work."
- URL: https://paulgraham.com/words.html , February 2022.
- Quote 4 (VERIFIED): "Half the ideas that end up in an essay will be ones you thought of while you were writing it."
- Supports: the short human-written framing is where the human's thinking happens, so delegating it removes the thinking.
- Cuts both ways: Quote 4 also says ideas are discovered DURING writing, which argues against expecting the framing to be right before the detailed work exists.

### 4.2 Leslie Lamport

- URL: https://www.microsoft.com/en-us/research/video/leslie-lamport-thinking-code/
- Author/org: Leslie Lamport, Microsoft Research (talk abstract).
- Date: 2014-07-15.
- Quote 1 (VERIFIED): "Architects draw detailed blueprints before a brick is laid or a nail is hammered. Programmers and software engineers seldom do. A blueprint for software is called a specification."
- Quote 2 (VERIFIED as Graham's quotation of Lamport in 4.1, SECONDARY for Lamport): "If you're thinking without writing, you only think you're thinking."
- URL: https://cacm.acm.org/opinion/who-builds-a-house-without-drawing-blueprints/ (live page returned 403; read via https://web.archive.org/web/2025/https://cacm.acm.org/opinion/who-builds-a-house-without-drawing-blueprints/).
- Author/org: Leslie Lamport, Communications of the ACM, Viewpoint.
- Date: 2015-04-01.
- Quote 3 (VERIFIED via web.archive.org snapshot): "it is a good idea to think about what we are going to do before doing it, and as the cartoonist Guindon wrote: “Writing is nature’s way of letting you know how sloppy your thinking is.”"
- Quote 4 (VERIFIED via web.archive.org snapshot): "If we understand something, we can explain it clearly in writing. If we have not explained it in writing, then we do not know if we really understand it."
- Supports: the act of writing the explanation is the test of the writer's own understanding, so a framing the human did not write proves nothing about what the human understands.

### 4.3 Bezos

- See 1.1.
- Quote (VERIFIED): "The great memos are written and re-written, shared with colleagues who are asked to improve the work, set aside for a couple of days, and then edited again with a fresh mind."
- Not found: the often-repeated Bezos line that narrative structure "forces better thought" comes from a 2004 internal email and was not located in a primary source in this pass.

### 4.4 "Your Brain on ChatGPT" (Kosmyna et al., MIT Media Lab)

- URL: https://arxiv.org/abs/2506.08872
- Author/org: Nataliya Kosmyna and co-authors.
- Date: 2025-06-10, revised 2025-12-31 (preprint).
- Quote 1 (VERIFIED, abstract): "Self-reported ownership of essays was the lowest in the LLM group and the highest in the Brain-only group. LLM users also struggled to accurately quote their own work."
- Quote 2 (VERIFIED, abstract): "Over four months, LLM users consistently underperformed at neural, linguistic, and behavioral levels."
- Supports: people who let the model write retain and own less of the content, which matters if the human must later defend or judge the design.
- CONTRADICTING critique, URL: https://arxiv.org/abs/2601.00856 , Milos Stankovic and three co-authors, 2026-01.
- Quote 3 (VERIFIED, abstract): "Our primary concerns focus on: (i) study design considerations, including the limited sample size; (ii) the reproducibility of the analyses; (iii) methodological issues related to the EEG analysis; (iv) inconsistencies in the reporting of results; and (v) limited transparency"
- Caveat: 54 participants, essay writing by students, not peer reviewed at the time of the first version.

### 4.5 Microsoft Research / CMU critical-thinking survey (Lee et al., CHI 2025)

- URL: https://www.microsoft.com/en-us/research/publication/the-impact-of-generative-ai-on-critical-thinking-self-reported-reductions-in-cognitive-effort-and-confidence-effects-from-a-survey-of-knowledge-workers/
- Author/org: Hao-Ping (Hank) Lee, Advait Sarkar, Lev Tankelevitch, Ian Drosos, Sean Rintel, Richard Banks, Nicholas Wilson.
- Date: April 2025.
- Quote 1 (VERIFIED, abstract): "higher confidence in GenAI is associated with less critical thinking, while higher self-confidence is associated with more critical thinking."
- Quote 2 (VERIFIED, abstract): "GenAI shifts the nature of critical thinking toward information verification, response integration, and task stewardship."
- Supports: a reviewer who trusts the agent's draft scrutinises it less, and the human's role drifts from authoring to verifying.
- Caveat: self-reported survey of 319 knowledge workers, correlational.

## 5. Contradicting and complicating evidence

### 5.1 Summaries are conventionally written LAST

- URL: https://captureplanning.com/articles/92131.cfm
- Author/org: CapturePlanning.com (proposal-writing trade site), undated.
- Quote 1 (VERIFIED): "you won't have a full understanding of all the implications until you've fully developed your solution, written the response, and gone through all the related pricing trade-offs. When you write the Executive Summary last, it will reflect a better understanding of the customer, the solution, and the competitive environment."
- Quote 2 (VERIFIED, the other side from the same page): "Writing the Executive Summary first forces you to think through the most important aspects of your proposal."
- URL: https://tenderwriters.com.au/blog/executive-summaries-do-you-write-first-or-last/ , Tender Writers, undated.
- Quote 3 (VERIFIED): "Many writers find that they organise their thoughts and discover their key messages through the act of writing itself. Therefore, they argue you should write the executive summary last."
- Quote 4 (VERIFIED): "Others say that you should write it first. That way, it acts like a roadmap for your response schedule"
- URL: https://writingcenter.unc.edu/tips-and-tools/abstracts/ , UNC Writing Center, undated.
- Quote 5 (VERIFIED): "For the purposes of writing an abstract, try grouping the main ideas of each section of the paper into a single sentence."
- Contradicts: a framing written before the work cannot summarize findings the work has not produced yet.
- Reconciliation offered by the sources themselves: written-first text is a ROADMAP or brief (intent, scope, decision wanted), while written-last text is a SUMMARY (findings); they are different artifacts that happen to sit in the same place.

### 5.2 Writing-as-thinking cuts against a fixed up-front framing

- Graham (4.1, Quote 4) and Tender Writers (5.1, Quote 3) both say key messages are discovered during drafting.
- Bryar and Carr (1.10, Quote 3) say the PR/FAQ starts full of holes and is redrafted as research fills them.
- Shape Up (1.7) writes the pitch after shaping work, not before it.
- Contradicts: the strict reading of "human writes the pitch BEFORE any detailed work"; established practice is a human-owned framing that is written early and REVISED, with the human remaining its author.

### 5.3 Tiering has a depth limit

- Nielsen (2.4, Quote 3): "designs that go beyond 2 disclosure levels typically have low usability because users often get lost when moving between the levels."
- Nielsen (2.4, Quote 2): the first level must contain "everything that users frequently need up front".
- Brown (1.8, Quote 2): "you don’t need to use all 4 levels of diagram".
- Contradicts: a four-tier reading path is deeper than the UI evidence supports, and a first tier that omits something the reviewer usually needs defeats the scheme.

### 5.4 Skipped content is invisible to the skipper

- Pernice (2.3, Quote 2): "users won’t realize this, since by definition they don’t know what they don’t see."
- Contradicts: any tiering that invites a reviewer to stop at tier 1 also licenses approving a design whose risk lives in tier 3; none of the practice sources above measure that failure rate.

### 5.5 More document process may be the wrong fix

- Zaninotto (3.2): spec-first workflows are "Waterfall" and "Systematic Bureaucracy".
- Böckeler (3.1): "I’d rather review code than all these markdown files."
- Nita (3.4): argues for visual review over text.
- Kiro Quick Spec (3.10, Quote 6): removes approval gates in favour of front-loaded questions.
- Contradicts: the assumption that a better-structured prose document is the right review surface at all, at least for small changes.

### 5.6 The AI-writes-the-short-part position exists and ships

- Copilot PR summaries and CodeRabbit walkthroughs (3.10) are default-on, machine-written top layers, and no source found in this pass measures them as worse than human-written ones.
- Gong et al. (3.5) found high description-code inconsistency in only 1.7% of agent PRs.
- Contradicts: the claim that machine-written top layers are generally untrustworthy; the measured failure rate is low, though costly when it occurs.

## 6. Gaps in this pass

- No primary text for Army Regulation 25-50, Minto's book, the HBR workslop article (paywalled live and empty in the web.archive.org snapshot), or the Bezos 2004 email.
- No empirical study of design-document review effectiveness versus length or layering.
- No fetched documentation for Devin, Cursor, or Graphite, and no independent evaluation of CodeRabbit or Copilot summaries.
- No source directly testing "summary written by the same model that wrote the doc"; 3.7 is an analogy from LLM-as-judge research.
- Dates marked "not verified" or "from memory" should be rechecked before citation.
- The Faros AI statistics in 3.6 and the 2004 Bezos email in 4.3 are second-hand and should be traced to their origin before use.
