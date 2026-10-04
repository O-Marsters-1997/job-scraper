# Feedback Pack

2 overall · 1 Job entries · 0 Collection entries

## How Suitability is computed

Each Job is scored 0-100 from the user's Picks and Jev's cached answers to each Option's question.
An answer resolves to yes, no or unknown for the checklist rows: the top of P(yes), P(no) and P(not stated) wins when it reaches 0.6, otherwise unknown. The score itself uses the probabilities.

Score = round(100 * (sum(weight * evidence * credit) + 0.5 * prior) / (sum(weight * evidence) + avoid cost + prior)), with prior = 1.

- Per dimension with nice or ok Picks, credit = min(1, sum of strength * P(yes) over its Picks / saturation), with strength 1 for nice and 0.5 for ok, and evidence = the largest P(yes) + P(no) over its Picks. Dimension weights and saturation:
  - tech: weight 2, saturation 3
  - role: weight 3, saturation 1
  - domain: weight 2, saturation 1
  - seniority: weight 3, saturation 1, Gate
  - work: weight 1, saturation 1, Gate
  - stage: weight 1, saturation 1
  - size: weight 1, saturation 1
  - employment: weight 2, saturation 1, Gate
- An avoid Pick adds 2 * P(yes) to the denominator and nothing to the numerator.
- A Gate caps the score at 44: every Pick in a Gate dimension resolves no and an option the user did not pick resolves yes. A salary below the floor also gates.
- A block Pick resolving yes forces the score to 0.
- Bands: Great from 80, Good from 65, Fair from 45, Poor below.
- A dimension with no evidence counts on neither side.

Current Picks:

| Option | Stance | Question |
|---|---|---|
| Go | nice | Does the role use Go? |

## Levers you may change

- Dimension weights and saturation, avoidWeight (2), prior (1), gateCap (44)
- resolveThreshold (0.6)
- Question wording
- New or retired Options
- The user's Picks

## Overall feedback

> Fine after the Pick change.

[picks changed]

> Scores run hot.

## Replay

0 scored jobs · 0 positives · 0 negatives

No positives yet: grade a job great or ok, apply, or keep a tailored CV.

## Job entries

### Backend Engineer, acme: score 72, should be lower [picks changed]

> Go is a given here.

- Model: typesafe/jev-1.13
- Score model: jev-old
- Score fingerprint: score-fp
- Content fingerprint: fp-1

| Option | Stance | Resolved | P(yes) | P(no) | P(not stated) | Confidence |
|---|---|---|---|---|---|---|
| Go | nice | yes | 0.90 | 0.05 | 0.05 | 0.00 |
| COBOL | nice | retired | no cached answer | | | |

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

(none)
