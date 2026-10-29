# Replay

Scoring: default weights and Bands

9 scored jobs · 3 positives · 5 negatives

- Positive rank: median rank percentile 44 (lower is better), 3 of 3 in the top 20
- Low-scored positives: 1 in the bottom half: COBOL | Maintainer (acme, score 23)

## Ship metrics

- Precision@20: not enough labels (8 in the last 28 days, need 20)
- Distinct scores in the top 100: 3
- Dismissed jobs in the top 100: 5
- Great band size: 0
- Concordance: 43% of positive/negative pairs score the positive higher

## Labelled jobs

| Job | Company | Label | Source | Score | Band | Rank |
|---|---|---|---|---|---|---|
| Platform Engineer | acme | positive | grade | 66 | good | 1 |
| Dismissed 0 | acme | negative | grade | 66 | good | 1 |
| Dismissed 4 | acme | negative | grade | 66 | good | 1 |
| Go Developer | acme | positive | grade | 50 | fair | 4 |
| Dismissed 1 | acme | negative | grade | 50 | fair | 4 |
| Dismissed 3 | acme | negative | grade | 50 | fair | 4 |
| COBOL \| Maintainer | acme | positive | grade | 23 | poor | 8 |
| Dismissed 2 | acme | negative | grade | 23 | poor | 8 |

## Labels by reason

| Reason | Negatives | Positives |
|---|---|---|
| tech | 2 | 1 |
| salary | 1 | 0 |
| culture | 4 | 0 |
