# Feedback Pack

2 overall entries

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

