# Feedback Pack

2 overall · 1 Job entries · 1 Collection entries

## How Suitability is computed

Each Job is scored 0-100 from the user's Picks and Jev's cached answers to each Option's question.
An answer resolves to yes, no or unknown: the top of P(yes), P(no) and P(not stated) wins when it reaches 0.6, otherwise unknown.

Score = round(100 * (met + 0.5 * 3) / (evaluable + 3)).

- A nice Pick counts per dimension: 1 to evaluable once any of the dimension's nice Picks resolves yes or no, and 1 to met if one resolves yes.
- An avoid Pick resolving yes adds 2 to evaluable and nothing to met.
- A block Pick resolving yes forces the score to 0.
- Unknown counts on neither side.

Current Picks:

| Option | Stance | Question |
|---|---|---|
| Go | nice | Does the role use Go? |
| COBOL (retired) | nice | Does the role use COBOL? |
| Kubernetes | nice | Does the role use Kubernetes? |

## Levers you may change

- Weights: niceWeight (1), avoidWeight (2), priorK (3)
- resolveThreshold (0.6)
- Question wording
- New or retired Options
- The user's Picks

## Overall feedback

> Two lines
> of reasoning.

> Scores run hot for backend roles.

## Job entries

### Backend Engineer, acme: score 72, should be lower

> Go is a given here.

- Model: typesafe/jev-1.13
- Score model: jev-old
- Score fingerprint: score-fp
- Content fingerprint: fp-1

| Option | Stance | Resolved | P(yes) | P(no) | P(not stated) | Confidence |
|---|---|---|---|---|---|---|
| Go | nice | yes | 0.90 | 0.05 | 0.05 | 0.00 |
| COBOL | nice | retired | no cached answer | | | |
| Kubernetes | nice | unknown | no cached answer | | | |

Job state sent to Jev:

```json
{
  "title": "Backend Engineer",
  "company": "acme",
  "location": "",
  "work_arrangement": "",
  "salary_raw": "",
  "description": "Build Go services."
}
```

## Collection entries

### Ranking of 2 Jobs

> The ranking is off.

- Model: typesafe/jev-1.13
- Filters: q=go, scored=true

1. no score ·  — 
2. 72 · Backend Engineer — acme · Go meets

