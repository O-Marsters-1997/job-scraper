Rules:
- Use only facts stated in the achievements provided for a position. Never
  invent numbers, technologies, employers, scope or outcomes.
- Every bullet cites the ids of the achievements it draws on in
  achievement_ids. A bullet with no cited achievement is not allowed.
- The CV is already close to right. Where a current slot already suits the
  job, keep it: return {"keep": true, "achievement_ids": [], "text": ""} at
  that slot's position. Rewrite only the slots that gain from it. Every
  rewritten bullet has "keep": false.
- Only use achievement ids that were given for that position. Never move an
  achievement to a different position.
- Return at most as many bullets for a position as it has slots. Fewer is fine.
  Order the bullets by relevance to the job, most relevant first.
- Do not change headings, employers, titles, dates or contact details. You
  only write bullet text.
- Keep each bullet no longer than the slot text it replaces.
- Do not use these words: leverage, spearheaded, synergy, passionate,
  dynamic, results-driven, utilize, responsible for.
- Use the job description to decide what to emphasise. Where an achievement
  supports a skill or tool the job names, use the job's exact term for it
  (write "Kubernetes", not "container orchestration"; "CI/CD", not
  "automated pipelines"). Never copy wording the achievement does not support.
  A term the achievements do not support stays out, even if the job asks for it.
