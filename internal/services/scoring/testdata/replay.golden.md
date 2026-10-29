# Replay

Scoring: default weights and Bands

9 scored jobs · 3 positives · 5 negatives

- Positive rank: median rank percentile 67 (lower is better), 3 of 3 in the top 20
- Low-scored positives: 2 in the bottom half: Go Developer (acme, score 37); COBOL | Maintainer (acme, score 20)

## Ship metrics

- Precision@20: not enough labels (8 in the last 28 days, need 20)
- Distinct scores in the top 100: 4
- Dismissed jobs in the top 100: 5
- Great band size: 0
- Concordance: 40% of positive/negative pairs score the positive higher

## Labelled jobs

| Job | Company | Label | Source | Score | Band | Rank |
|---|---|---|---|---|---|---|
| Platform Engineer | acme | positive | grade | 56 | fair | 1 |
| Dismissed 0 | acme | negative | grade | 56 | fair | 1 |
| Dismissed 4 | acme | negative | grade | 56 | fair | 1 |
| Dismissed 3 | acme | negative | grade | 50 | fair | 4 |
| Go Developer | acme | positive | grade | 37 | poor | 6 |
| Dismissed 1 | acme | negative | grade | 37 | poor | 6 |
| COBOL \| Maintainer | acme | positive | grade | 20 | poor | 8 |
| Dismissed 2 | acme | negative | grade | 20 | poor | 8 |

## Labels by reason

| Reason | Negatives | Positives |
|---|---|---|
| tech | 2 | 1 |
| salary | 1 | 0 |
| culture | 4 | 0 |
