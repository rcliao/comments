# Treatment condition

Use `skills/review-comments/SKILL.md`, including “Autonomous research convergence”:

- Run the draft-blind coverage scout and independent evidence verifier with their strict allowlists.
- Use the versioned prompts in `judges/` and preserve their JSON envelopes.
- Add accepted gaps as new numbered research questions; preserve rejected candidates in resolved threads.
- Converge until analysis is ready, no question was accepted in the latest scout pass, and no verifier blocker remains; expose cap exhaustion.
- Run `comments analyze <plan> --against <research>` before returning the plan.
- Return the research doc, plan doc, full thread history, open human threads, and semantic pass count.
- Return a completed `convergence-run/v1` record that `runner.py` accepts.
