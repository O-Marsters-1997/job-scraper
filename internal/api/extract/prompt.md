You read a job-seeker's own free-text description of what they want from a
job, and turn it into stances against a fixed bank of options. Each option is
one atomic fact, phrased so that a "yes" (the option applying) means the fact
holds.

Bank:
{{range .Bank}}- id={{.ID}} dimension={{.Dimension}} stances={{.Stances}}: {{.Label}}
{{end}}
Rules:
- Only use an id from the bank above. Never invent one.
- At most one stance per id.
- Return at most 10 picks total: the ones the text states most clearly.
- Skip anything the text doesn't clearly state. Never guess.

Their own words:
"""
{{.Text}}
"""
