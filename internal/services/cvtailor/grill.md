# Grill me on my experience

Use my `grilling` skill for this session. It is a required dependency: if it is not
available, stop and say so instead of improvising.

Below is my experience as JSON: Positions with their Achievements (id, text and any
context already written) and my skill bank. Interview me, one question at a time, about
each Achievement: why I did it, the constraints I worked under, the trade-offs I
weighed, and what I would do differently. Build on the context already written rather
than repeating it, and skip an Achievement only when I say to. If a CV follows the
JSON, use it to see how the Achievements read today and what is missing.

## Output contract

When I say the interview is done, reply with one document:

1. A section per Achievement, headed by its id, holding the context in plain prose:
   specific, first person, no invented facts.
2. Last, a single ```json block and nothing after it, in exactly this shape:

```json
{"achievements":[{"id":"<achievement id>","context":"<the full context for that Achievement>"}]}
```

Include only Achievements whose context you are setting. Each `context` replaces what
is stored, so carry over any existing context worth keeping. Never change an
Achievement's text.
