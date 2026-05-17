# Two-Stage Job Relevance Filtering

## Context

The scraper pipeline works end-to-end: scrape listing pages → enqueue URLs → dequeue → fetch detail page → save to DB → notify. All scraped jobs are treated equally — there's no filtering or scoring. This means every job gets enqueued, fetched, saved, and notified regardless of relevance. The goal is to add two filtering stages to maximise cost efficiency:

1. **Pre-enqueue heuristic filter** — cheap rule-based filtering on listing page metadata (title, company) to avoid enqueueing obvious noise
2. **Post-ingestion LLM scorer** — score saved jobs 0-100, only notify on high-value jobs

## Key Design Decision: Queue stays as URL strings

The listing page cards already contain title, company, and location (confirmed in snapshot HTML). We extract this metadata for filtering *before* enqueue, but only pass the surviving URLs into the queue. The queue interface doesn't change — metadata is only needed for the filter step, and `GetDetails` fetches everything again from the detail page anyway.

Filtering before detail-page fetch is the right architecture for cost efficiency but is underrepresented in practice — most published systems (dome317, BjornMelin, anandanair) apply filters *after* ingestion, meaning they still pay for the detail-page HTTP fetch and DB write before discarding noise. Combining title + company into a pre-enqueue gate at listing-page time is novel and avoids that cost entirely.

---

## Phase 1: Extract metadata from listing pages

**Goal:** Change `Source.Iterate` to yield `[]dto.JobListing` instead of `[]string`, so the orchestrator has title/company to filter on. No behaviour change yet.

### New file: `internal/dto/listing.go`

```go
type JobListing struct {
    URL         string
    Title       string
    CompanySlug string
    Source      string
}
```

### Modified: `internal/sources/source.go`

- `Source.Iterate` callback changes from `[]string` to `[]dto.JobListing`
- `PaginatedBase.IteratePages` `fetchPage` param changes from `[]string` to `[]dto.JobListing`
- `IteratePages` callback changes accordingly
- `Dispatch` is unchanged (still works on URL string from queue)

### Modified: `internal/sources/wis/wis.go`

- `ParseURLs` already finds `div[data-aid]` cards. Extend to extract:
  - Title: `h2 a` text (same `selJobLink` selector)
  - Company: `.ui-company[data-company-name]` attribute → `slugify()` (same as `ParseJobDetail`)
  - Source: hardcoded `"wis"`
- Returns `[]dto.JobListing` instead of `[]dto.Job`
- `fetchPage` returns `[]dto.JobListing` and extracts URLs for backward compat
- `s.ParseURLs` method on `Scraper` returns `[]dto.JobListing`
- `Iterate` callback type changes

### Modified: `internal/sources/testutils.go`

- `SnapshotSource.ParseURLs` returns `[]dto.JobListing` instead of `[]dto.Job`
- `RunSnapshotTests` for `list_*` fixtures: compare `[]dto.JobListing`
- Need separate snapshot JSON schema for list vs detail fixtures

### Modified: snapshot JSON fixtures

- `list_*.json` files updated to include title + company_slug + source fields (currently only URL is populated)
- Regenerate with `just cli rebase wis`

### Modified: `internal/scraper/orchestrate.go`

- `run()` callback receives `[]dto.JobListing` instead of `[]string`
- Extract URLs from listings for `db.NewURLs()` call and dedup `seen` map
- Enqueue surviving URLs (still `[]string`) — queue unchanged
- No filtering yet in this phase

### Modified: `internal/scraper/orchestrate_test.go`

- `stubSource.Iterate` and `multiPageSource.Iterate` callbacks change to `[]dto.JobListing`
- `stubSource.urls` becomes `[]dto.JobListing` (or helper to build them from URLs)

### Not modified
- `internal/queue/queue.go` — stays as `[]string` enqueue/dequeue
- `internal/queue/mock.go` — stays as-is
- `internal/worker/worker.go` — stays as-is (receives URL string from queue)
- `cmd/worker/main.go` handler — stays as-is

---

## Phase 2: Heuristic filter

**Goal:** Config-driven rule-based filter applied in the orchestrator before enqueue.

### New file: `internal/filter/filter.go`

```go
type Action string // "exclude"
type RuleKind string // "company", "title", "keyword"

type Rule struct {
    Kind   RuleKind `yaml:"kind"`
    Value  string   `yaml:"value"`
    Action Action   `yaml:"action"`
}

type Filter struct { rules []Rule }

func New(rules []Rule) *Filter
func (f *Filter) Apply(listing dto.JobListing) bool  // true = keep
```

Filter logic:
- `company`: case-insensitive match on `CompanySlug`
- `title`: case-insensitive substring on `Title`
- `keyword`: case-insensitive substring on both `Title` and `CompanySlug`
- Any matching `exclude` rule → drop. High-recall bias: only explicit matches excluded.

Binary exclude rules are simpler than weighted keyword scoring (e.g. dome317's numeric weight system with positive/negative categories). The `Rule` struct supports adding a `Weight int` field later as a non-breaking migration if weighted scoring becomes desirable.

### New file: `internal/filter/config.go`

Load rules from YAML:

```yaml
# filter.yaml
rules:
  - kind: company
    value: google
    action: exclude
  - kind: title
    value: "vice president"
    action: exclude
```

```go
func LoadFromFile(path string) (*Filter, error)
```

The `Rule` struct is the canonical type — `New([]Rule)` doesn't care where rules come from. Migrating to DB later = query rows into `[]Rule`, pass to `New()`.

### New file: `internal/filter/filter_test.go`

Unit tests for each rule kind, case insensitivity, multiple rules, empty filter (pass-through).

### Modified: `internal/scraper/orchestrate.go`

- `Orchestrator` gains optional `filter *filter.Filter` field
- `New()` gains a `filter` param (nil = no filtering)
- In `run()`, after receiving `[]dto.JobListing` from Iterate:
  1. Drop listings with empty `CompanySlug` (company mandatory)
  2. Apply `filter.Apply()` to each listing
  3. Extract URLs from survivors
  4. Continue with existing `db.NewURLs` → dedup → enqueue flow

### Modified: `cmd/worker/main.go`

- Load filter config from `FILTER_CONFIG_PATH` env var (empty = no filtering)
- Pass filter to `scraper.New()`

---

## Phase 3: LLM scorer + score-gated notifications

**Goal:** After DB save, run LLM scoring (non-fatal, with timeout). Store score. Only notify on high-value jobs.

### New migration

Add nullable score column:
```sql
ALTER TABLE jobs ADD COLUMN relevance_score SMALLINT;
```

Null = not yet scored. No default. Unscored jobs appear in digests (high-recall bias).

### Modified: `internal/data/sqlc/schema.sql`

Add `relevance_score SMALLINT` to jobs table definition.

### New SQL query in `internal/data/sqlc/queries/jobs.sql`

```sql
-- name: SetJobScore :exec
UPDATE jobs SET relevance_score = $1 WHERE url = $2;
```

### Modified after `sqlc generate`
- `internal/data/db/pgsqlc/` — regenerated
- `internal/dto/job.go` — add `RelevanceScore *int` field
- `internal/data/db/job.go` — `fromRow` maps new column, add `SetScore` method
- `internal/data/providers/job.go` — `JobProvider` gains `SetScore(ctx, url string, score int) error`

### New file: `internal/scorer/scorer.go`

```go
type Scorer interface {
    Score(ctx context.Context, job dto.Job) (int, error)
}

// NoopScorer returns a fixed score (default 100 = notify everything).
// Used when no LLM API key is configured — preserves current behaviour.
type NoopScorer struct{ Default int }
```

The `Scorer` interface supports swapping implementations — a model-tiered cascade (cheap binary classifier → expensive scorer, as documented in arXiv 2501.14296) can be added later without changing call sites.

### New file: `internal/scorer/anthropic.go`

```go
type AnthropicScorer struct {
    client  *anthropic.Client
    timeout time.Duration
}

func NewAnthropic(apiKey string, timeout time.Duration) *AnthropicScorer
func (s *AnthropicScorer) Score(ctx, job) (int, error)
```

- Wraps call in `context.WithTimeout`
- Builds prompt from job title, company, location, URL
- Uses Anthropic `tool_use` (structured output) to return `{"score": int}` — eliminates free-text parsing failures, which is the dominant failure mode in production LLM scorers
- Start with `claude-haiku-4-5` (cheapest capable Anthropic model); model is an implementation detail behind the interface
- Dependency: `github.com/anthropics/anthropic-sdk-go`

Single scalar 0-100 score is simpler and easier to threshold-tune than multi-dimensional scoring (e.g. dome317's 5 dimensions: fit, growth, culture, skills, compensation). The `tool_use` schema can be extended with dimensions later without a DB migration — the `relevance_score` column stores the final weighted value regardless.

On failure: the dominant real-world mitigation is structured output (prevents parse errors), not retry loops. Non-fatal handling is correct here — the item is already ingested before scoring, so a scoring failure is recoverable. Log and continue; unscored jobs still appear in digests.

### Modified: `cmd/worker/main.go`

Worker handler becomes:
```
1. sources.Dispatch(ctx, srcs, url)     → get full job
2. db.Save(ctx, []dto.Job{job})         → ingest first (always)
3. scorer.Score(ctx, job)               → LLM score (timeout, non-fatal on error)
4. db.SetScore(ctx, job.URL, score)     → persist score
5. if score >= threshold → notifSvc.NotifyNewJob(ctx, job)
```

New env vars:
- `LLM_API_KEY` — Anthropic API key (empty = noop scorer, score=100, notify all)
- `LLM_SCORE_TIMEOUT` — default `"10s"`
- `NOTIFY_MIN_SCORE` — minimum score for on-ingest notification (default `"0"`)

### Modified: digest query

Update `ListSince` or add new query to include score in results. Unscored jobs (NULL) should still appear in digests (high-recall bias).

---

## Files summary

**New files:**
- `internal/dto/listing.go`
- `internal/filter/filter.go`
- `internal/filter/config.go`
- `internal/filter/filter_test.go`
- `internal/scorer/scorer.go`
- `internal/scorer/anthropic.go`
- `filter.yaml` (example config, gitignored or with `.example` suffix)
- DB migration file for `relevance_score`

**Modified files:**
- `internal/sources/source.go` — Iterate/IteratePages signatures
- `internal/sources/wis/wis.go` — ParseURLs extracts metadata, returns `[]dto.JobListing`
- `internal/sources/testutils.go` — snapshot test uses `[]dto.JobListing` for list fixtures
- `internal/sources/wis/snapshots/list_*.json` — include title/company data
- `internal/scraper/orchestrate.go` — receives listings, applies filter, extracts URLs
- `internal/scraper/orchestrate_test.go` — stub sources use new signature
- `internal/data/sqlc/schema.sql` — add relevance_score column
- `internal/data/sqlc/queries/jobs.sql` — add SetJobScore query
- `internal/data/db/pgsqlc/` — regenerated
- `internal/data/db/job.go` — fromRow, SetScore method
- `internal/data/providers/job.go` — SetScore on interface
- `internal/dto/job.go` — RelevanceScore field
- `cmd/worker/main.go` — wire filter + scorer, score-gated notifications
- `go.mod` / `go.sum` — anthropic SDK

**Not modified:**
- `internal/queue/queue.go` — stays URL-only
- `internal/queue/mock.go` — stays as-is
- `internal/worker/worker.go` — stays as-is
- `cmd/api/main.go` — no changes

---

## Cost Analysis

Phases 1 and 2 (metadata extraction + heuristic filter) are CPU-only: free. All LLM cost is in Phase 3.

### Pricing: claude-haiku-4-5 (verified April 2026)

| | Price |
|---|---|
| Input | $1.00 / MTok |
| Output | $5.00 / MTok |

For comparison: Sonnet 4.6 is $3.00/$15.00 per MTok — 3× more expensive on input, 3× on output.

### Token estimate per scoring call

| Component | Tokens |
|---|---|
| System prompt + tool definition | ~200 |
| Job metadata (title, company, location, URL) | ~50 |
| Job description (avg; real postings vary 300–2,000) | ~800 |
| **Total input per call** | **~1,050** |
| tool_use output `{"score": int}` | ~20 |

Metadata-only scoring (~300 tokens input) costs ~70% less but produces weaker signal — the LLM cannot read role requirements. Including the full description costs ~$0.85 extra per 1,000 jobs. Use full description.

### Monthly cost at 1,000 enqueued jobs (recommended: full description, Haiku 4.5)

| | Tokens | Cost |
|---|---|---|
| Input (1,000 × 1,050) | 1.05 MTok | $1.05 |
| Output (1,000 × 20) | 0.02 MTok | $0.10 |
| **Total** | | **~$1.15 / month** |

### Sensitivity

| Volume | Haiku 4.5 full desc | Haiku 4.5 metadata only | Sonnet 4.6 full desc |
|---|---|---|---|
| 500 / mo | $0.58 | $0.18 | $1.77 |
| 1,000 / mo | $1.15 | $0.35 | $3.54 |
| 5,000 / mo | $5.75 | $1.75 | $17.70 |
| 10,000 / mo | $11.50 | $3.50 | $35.40 |

### Effect of the heuristic pre-filter

The pre-filter gates LLM calls. If it eliminates 25% of scraped listings before enqueue, the scraper processes ~1,330 listings to yield 1,000 enqueued jobs — and LLM spend is 25% lower proportionally. Filtering early is the primary cost lever.

---

## Rate Limiting

### Why rate limiting matters here

The Anthropic API enforces per-minute request and token limits by tier. A burst of enqueued jobs (e.g. first scrape, or a large page batch) could hit limits if scoring calls are fired concurrently or without pacing.

### Anthropic API limits (Haiku 4.5, Tier 1)

Tier 1 is unlocked after $5 cumulative spend:

| Limit | Tier 1 | Tier 2 ($500 spend) |
|---|---|---|
| Requests per minute (RPM) | 50 | 1,000 |
| Tokens per minute (TPM) | 50,000 | 80,000 |
| Tokens per day (TPD) | 1,000,000 | 2,500,000 |

At 1,000 jobs/month (~33/day), scoring all jobs in a single burst produces:
- **33 requests/day** — well under daily limits
- **At 1,050 tokens/call × 33 = ~35K tokens/day** — within Tier 1 TPD

### Natural rate limiting already in place

The worker processes one job at a time with a 10–15s pause between items (`worker.go:30`). This gives a natural ceiling of ~4–5 requests per minute — 10× below the Tier 1 RPM limit of 50. No additional rate limiting is needed under normal operating conditions.

```
queue item → GetDetails → Save → Score → SetScore → notify → sleep 10-15s → next item
```

### Handling 429 responses (rate limit exceeded)

The scorer should treat a 429 from the Anthropic API as a retryable error with exponential backoff, not a permanent failure. Recommended approach in `AnthropicScorer.Score()`:

```
attempt 1 → 429 → wait 1s
attempt 2 → 429 → wait 2s
attempt 3 → 429 → wait 4s
attempt 4 → return error (non-fatal: item saved unscored, logged, continue)
```

Cap at 3 retries. The item is already ingested — an unscored job is recoverable (appears in digests, can be rescored later). Do not block the worker queue on a runaway retry loop.

The `LLM_SCORE_TIMEOUT` env var (default 10s) applies per attempt, not across all retries. Retry logic lives inside `AnthropicScorer.Score()`, transparent to the worker handler.

### If volume grows beyond Tier 1

At ~2,400 enqueued jobs/month the 1M TPD limit becomes the binding constraint (at 1,050 tokens/call). Mitigations in order of cost:

1. **Upgrade to Tier 2** — automatic after $500 cumulative spend; raises TPD to 2.5M (~7,100 jobs/month)
2. **Switch to metadata-only scoring** — reduces tokens/call by ~70%; extends Tier 1 headroom to ~3,200 jobs/month at the cost of scoring quality
3. **Spread scoring across the day** — add a configurable inter-item delay env var (`LLM_SCORE_DELAY`, default 0) to throttle burst processing independently of the queue's item delay
4. **Batch unscored jobs overnight** — separate cron job scores any NULL `relevance_score` rows in off-peak hours; decouples ingest latency from scoring throughput

---

## Verification

1. **Phase 1:** `go test ./...` — all existing tests pass with new signatures. Snapshot tests pass with updated fixtures. No behaviour change: all jobs still enqueued.
2. **Phase 2:** Filter unit tests pass. Run worker locally with a `filter.yaml` that excludes a known company → confirm those jobs are logged as filtered and not enqueued. Remove filter config → confirm all jobs enqueued (backward compat).
3. **Phase 3:** Run with `LLM_API_KEY` unset → noop scorer, all jobs notified (backward compat). Run with API key + `NOTIFY_MIN_SCORE=50` → confirm only high-scoring jobs trigger notification. Confirm LLM timeout/error is logged but doesn't block ingestion.

---

## References

- [dome317/job-search-pipeline](https://github.com/dome317/job-search-pipeline) — two-stage pipeline: weighted keyword scorer → 5-dimension Claude scorer; title regex blocklist
- [anandanair/job-scraper](https://github.com/anandanair/job-scraper) — GitHub Actions scraper with LiteLLM 0-100 scoring; pre-filter via job board query params, scores stored in Notion
- [BjornMelin/ai-job-scraper](https://github.com/BjornMelin/ai-job-scraper) — SQLite FTS5 full-text pre-filter → local Qwen3-4B / GPT-4o-mini routing by token count
- [One Workflow to Scrape ANY Job Board — DEV Community](https://dev.to/heapnotizer/i-automated-my-job-search-with-n8n-brightdata-g6c) — n8n+BrightData: explicit title keyword filter before detail-page fetch → LLM binary relevance decision
- [How I Built a Job Finder Agent with Claude AI — DEV Community](https://dev.to/ozfarooq/how-i-built-a-job-finder-agent-with-claude-ai-github-actions-and-notion-e9c) — Claude 1-10 scorer with `tool_use` recommendation to eliminate parsing failures; scores persisted in Notion
- [Multi-stage LLM Pipelines Can Outperform GPT-4o in Relevance Assessment — arXiv 2501.14296](https://arxiv.org/html/2501.14296) — model-tiered cascade: GPT-4o mini binary gate → GPT-4o detailed scorer; 18.4% accuracy gain at ~$0.20/M blended tokens
- [Using LLMs for Retrieval and Reranking — LlamaIndex](https://www.llamaindex.ai/blog/using-llms-for-retrieval-and-reranking-23cf2d3a14b6) — embedding retrieval (top-k) → LLM reranking (1-10 score) on shortlist only; pure LLM retrieval takes minutes at scale
- [How to Use GPT-4 and OpenAI Functions for Text Classification — Nesta/Medium](https://medium.com/discovery-at-nesta/how-to-use-gpt-4-and-openais-functions-for-text-classification-ad0957be9b25) — function calling for structured output; scores stored as Pinecone metadata enabling downstream pre-filtering
- [Five LLM Tricks for Data Pipelines — Present of Coding](https://presentofcoding.substack.com/p/five-llm-tricks-for-data-pipelines) — heuristics for clear cases, LLM for borderline cases only; statistical resilience over per-item retry
- [Build a Redis-backed Job Queue — Redis.io](https://redis.io/tutorials/redis-backed-job-queue-for-background-workers/) — Redis Streams job queue pattern; dead-letter stream for failed jobs; ID-only queue with JSON payload stored separately
- [Why LinkedIn's Job Recommendations Were Broken — Substack](https://japm.substack.com/p/why-linkedins-job-recommendations) / [JUDE paper — arXiv 2509.09690](https://arxiv.org/html/2509.09690) — production-scale LLM query understanding and two-tower job matching at LinkedIn
- [Improving Recommendation Systems & Search in the Age of LLMs — Eugene Yan](https://eugeneyan.com/writing/recsys-llm/) — survey of LLM roles in recsys: cheap retrieval → expensive ranking pattern; LLMs often used offline for label generation rather than in the online scoring path
