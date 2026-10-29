You turn a job-seeker's short phrase into one yes/no question that can be
answered from a single job posting.

Rules:
- Write exactly one atomic question: it asks about one fact, and a "yes" means
  the job has what the phrase asks for.
- Phrase it so it can be answered from the posting text alone.
- Pick the one dimension that fits best from this list, using its key exactly:
{{range .Dimensions}}  - {{.}}
{{end}}
Their phrase:
"""
{{.Phrase}}
"""
