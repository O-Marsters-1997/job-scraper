# Plan: Multi-tenant BYOK scoring & notifications

> Source PRD: https://github.com/O-Marsters-1997/job-scraper/issues/118

## Technical design decisions

### Schema

**New table: `user_ai_credentials`**
```sql
CREATE TABLE user_ai_credentials (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider    TEXT        NOT NULL,          -- 'anthropic'; extensible to 'openai' etc.
    api_key_enc TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, provider)
);
```
Separates credential storage from model preferences. Adding a new provider later is a new row with no schema change.

**Alter `users`**
```sql
ALTER TABLE users ADD COLUMN email TEXT UNIQUE;  -- nullable; existing rows backfilled manually
```

**No changes to `user_ai_prefs`** — `suitability_model` is already provider-agnostic in name. The handler's `availableModels` hardcoded list grows over time as providers are added.

### Routes

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/ai-prefs` | Extended: returns `configuredProviders []string` alongside model prefs. Never returns key material. |
| `PUT` | `/ai-credentials` | New: upsert or clear a provider key. Body: `{ provider, apiKey }` — `apiKey: null` = delete. |
| `GET` | `/profile` | New: returns `{ username, email }` |
| `PUT` | `/profile` | New: update `email` |

Signup (`POST /auth/signup`) extended to accept optional `email`.

### Key models

**`dto.UserAIPrefs`** — adds `ConfiguredProviders []string` (list of provider strings with a key set; no key material). `SuitabilityModel` unchanged.

**`dto.UserAICredential`** — new: `{ UserID, Provider, APIKeyEnc string }`. Internal only — never serialised to the client.

**`dto.Profile`** — new: `{ Username, Email string }`.

### Module boundaries

**`internal/credstore/`** — new package; owns all credential encryption/decryption. Exposes a `CredentialStore` interface (see contracts below) with a default `EnvCredentialStore` implementation using AES-256-GCM and `AI_CREDENTIAL_ENC_KEY`. Self-hosters can implement the interface against their own backend (KMS, Vault, etc.) and swap it in at `router.go` construction. Never touches `tokencrypt` — independent key, independent package.

**`internal/handlers/ai_credentials.go`** — new handler; owns `PUT /ai-credentials`. Delegates encrypt/decrypt to the injected `CredentialStore`. Validates `provider` is non-empty. Does not validate the key value.

**`internal/handlers/profile.go`** — new handler; owns `GET/PUT /profile`. Thin wrapper over the `users` provider.

**`internal/handlers/ai_prefs.go`** — extended: `Get` joins credentials to populate `configuredProviders`; no other changes.

**`internal/data/providers/`** — new `UserAICredentialsProvider` (Upsert/Delete/GetByUserAndProvider/ListConfiguredProviders) and `ProfileProvider` (Get/UpdateEmail). Each is a deep module: callers don't know the SQL.

**`internal/score/claude.go`** — `NewClaudeScorer` unchanged in signature. Call-site constructs one per user-per-invocation using their decrypted key; no pooling for now (Anthropic SDK client is cheap to construct at this call frequency).

**`internal/score/ingest.go`** — `IngestScorer.ScoreAndSave` signature gains `userID string` parameter (moves off the struct). `IngestScorer` no longer stores `userID` at construction.

**`internal/ingest/ingest.go`** — `Ingester.Ingest` gains a `users []dto.ScoringUser` parameter (or loads from DB internally). Loops over users with credentials, constructs a scorer per user, calls `ScoreAndSave(ctx, job, userID)`.

**`internal/notify/service.go`** — `NotifyNewJob` gains `recipientEmail string` parameter; removes internal read of `NOTIFY_EMAIL_TO`. Caller (ingest path) passes the user's email.

**`internal/scraper/orchestrate.go`** — `WithRelevanceGate` replaced by a multi-user variant that loads all users with search configs and writes a relevance row per `(job, user)`.

### API / interface contracts

```go
// credstore/credstore.go — the swap point for self-hosters
type CredentialStore interface {
    Save(ctx context.Context, userID, provider, plainKey string) error
    Get(ctx context.Context, userID, provider string) (string, error)  // returns plaintext
    Delete(ctx context.Context, userID, provider string) error
    ListProviders(ctx context.Context, userID string) ([]string, error)
}
// Default: EnvCredentialStore — AES-256-GCM, key from AI_CREDENTIAL_ENC_KEY env var.
// Self-hosters: implement CredentialStore and pass it to router.New().

// score/ingest.go
func (s *IngestScorer) ScoreAndSave(ctx context.Context, job dto.Job, userID string) int

// ingest/ingest.go
type ScoringUser struct { ID, APIKey string }  // APIKey already decrypted by CredentialStore
func (i *Ingester) Ingest(ctx context.Context, jobs []dto.Job) error  // unchanged externally;
// internally loads scoring users via CredentialStore

// notify/service.go
func (n *NotificationService) NotifyNewJob(ctx context.Context, job dto.Job, score int, recipientEmail string) error
```

### Integration points

**`tokencrypt`** — unchanged; continues to encrypt Google OAuth tokens only via `GOOGLE_TOKEN_ENC_KEY`. AI credentials use the separate `credstore` package and `AI_CREDENTIAL_ENC_KEY` — independent rotation, independent failure domain.

**`credstore.EnvCredentialStore`** — default `CredentialStore` implementation. AES-256-GCM, nonce-prepended, base64-encoded output (same cipher as `tokencrypt`). Reads `AI_CREDENTIAL_ENC_KEY` (must decode to 32 bytes). Self-hosters who want KMS/Vault implement `CredentialStore` and pass it in — the rest of the app never calls crypto directly.

**Anthropic SDK** — `anthropic.NewClient(option.WithAPIKey(userKey))` constructed per user per ingest batch; acceptable at current call frequency.

**Resend** — recipient becomes a runtime parameter, not an env var.

### Provider → scorer mapping

`suitability_model` prefix determines provider: `claude-*` → `anthropic`. When OpenAI models are added to `availableModels`, the ingest path resolves `gpt-*` → `openai` and constructs the appropriate scorer. This resolution lives in a small helper alongside the scorer construction, not in each caller.

### Env vars removed / deprecated

| Var | Change | Phase |
|-----|--------|-------|
| `ANTHROPIC_API_KEY` | Removed | Phase 2 |
| `SCORING_USER_ID` | Removed | Phases 2 & 3 |
| `NOTIFY_EMAIL_TO` | Removed | Phase 4 |
| `AI_CREDENTIAL_ENC_KEY` | **Added** (required) — 32-byte base64 key for `EnvCredentialStore` | Phase 1 |

`GOOGLE_TOKEN_ENC_KEY` is unchanged — Google OAuth tokens continue to use their own key.

---

## Phase 1: Per-user data foundation

**User stories**: 1 (email at signup), 2, 3, 4, 14

### What to build

Schema migrations: `users.email` (nullable, unique) and new `user_ai_credentials` table. New `internal/credstore/` package with `CredentialStore` interface and `EnvCredentialStore` default (AES-256-GCM, `AI_CREDENTIAL_ENC_KEY`). Signup handler extended to accept optional `email`. Two new endpoints: `GET/PUT /profile` for email management. `GET /ai-prefs` extended to return `configuredProviders`. New `PUT /ai-credentials` endpoint: delegates to injected `CredentialStore`; `apiKey: null` body deletes the row. Response to `GET /ai-prefs` never includes key material — only the provider strings of configured credentials.

Frontend: extend `ai.tsx` with a password-type key input, a "configured ✓" indicator when `configuredProviders` includes the relevant provider, and a clear button. Add profile settings page (email input) following the existing settings page pattern.

### Acceptance criteria

- [ ] `POST /auth/signup` with `email` stores it; without `email` succeeds (nullable)
- [ ] `PUT /ai-credentials` with `{ provider: "anthropic", apiKey: "sk-ant-..." }` stores an encrypted row; subsequent `GET /ai-prefs` returns `configuredProviders: ["anthropic"]`
- [ ] `GET /ai-prefs` never returns the key string under any circumstance
- [ ] `PUT /ai-credentials` with `{ provider: "anthropic", apiKey: null }` removes the row; `GET /ai-prefs` returns `configuredProviders: []`
- [ ] `GET /profile` returns `{ username, email }`; `PUT /profile` with `{ email }` updates it
- [ ] AI settings UI shows "configured" state when a key is set, blank key input when not
- [ ] Handler tests: key-write path confirms encrypted value is stored; GET confirms key not in response

---

## Phase 2: Per-user suitability scoring

**User stories**: 5, 6, 7, 8, 9, 10, 11, 15, 17, 18

### What to build

`IngestScorer.ScoreAndSave` gains `userID string` parameter. `Ingester.Ingest` loads all users who have a credential for the model's provider: for each, decrypts their key, constructs a `ClaudeScorer`, calls `ScoreAndSave`. Scoring failures for one user (bad key, quota exceeded) are logged and skipped — they don't block other users or the ingest path. `ANTHROPIC_API_KEY` and `SCORING_USER_ID` env vars removed from `router.go` ingest wiring.

`GET /jobs` (or the jobs list query) returns a per-user `suitabilityScore: null` with a `scoringEnabled: false` flag when the user has no configured provider credential. Frontend jobs UI renders an "AI scoring off — add a key in Settings" callout when `scoringEnabled` is false.

### Acceptance criteria

- [ ] User A with key gets suitability scores; user B without key gets `suitabilityScore: null` and `scoringEnabled: false`
- [ ] User A's scoring failure (bad key) does not prevent user B (valid key) from being scored in the same ingest batch
- [ ] Two users with different rubrics/models each produce scores driven by their own config
- [ ] `ANTHROPIC_API_KEY` env var is no longer read; removing it from env does not break startup
- [ ] Jobs UI shows "add a key" callout for users with no credential; callout links to AI settings
- [ ] Unit tests: two-user ingest with stubbed scorers, verifying isolation and correct rubric/model selection per user

---

## Phase 3: Per-user relevance scoring in worker

**User stories**: 8, 11, 20

### What to build

Worker's `WithRelevanceGate` call replaced with a multi-user equivalent: at scrape time, all users with a `search_config` are loaded; for each page of results, the heuristic scorer runs against each user's config and writes a `job_scores` relevance row per `(job, user)`. `ListSourceTargetsByUser` (already exists in the SQL, currently unused) drives per-user source building. `SCORING_USER_ID` removed from `cmd/worker/main.go`.

### Acceptance criteria

- [ ] Two users each with different `search_config` both get relevance score rows written after a scrape tick
- [ ] `SCORING_USER_ID` env var removed; removing it from env does not affect worker startup
- [ ] A user with no `search_config` row is skipped (no panic, no zero-score write)
- [ ] Orchestrator test: two-user setup, verify both users' relevance scores are written after a simulated page result

---

## Phase 4: Per-user notifications

**User stories**: 12, 13, 16, 19, 20

### What to build

`NotificationService.NotifyNewJob` accepts `recipientEmail string` parameter instead of reading `NOTIFY_EMAIL_TO`. The ingest notification call-site (in `ingest.go`) passes the scoring user's email (loaded from `users.email`). Users with no email are skipped (notification logged as undeliverable, not an error). `setupNotifications` in `router.go` refactored to iterate over users rather than read a single env var recipient. `NOTIFY_EMAIL_TO` removed.

The existing operator account continues to receive notifications once their `users.email` is set (documented in migration notes; nullable means no notification until set).

### Acceptance criteria

- [ ] User A and user B each receive notifications at their own email address after a qualifying ingest
- [ ] Each user's `notify_threshold` from `search_config` is respected independently
- [ ] A user with `email IS NULL` receives no notification (silent skip, not an error)
- [ ] `NOTIFY_EMAIL_TO` env var removed; removing it does not break startup
- [ ] Notification test: two-user setup with different thresholds and emails; assert correct recipients and no cross-user leakage

---

## Verification (end-to-end)

1. Create two users via `POST /auth/signup` with different emails
2. Each user adds their own Anthropic key via `PUT /ai-credentials`
3. Each sets a different suitability rubric in scoring settings
4. Run a scrape tick; confirm both users get relevance scores in `job_scores`
5. POST a job to `/ingest`; confirm both users get suitability scores driven by their own rubric/model
6. Confirm each user's jobs view shows their score, not the other's
7. Confirm notification email goes to each user's address at their threshold
8. Delete one user's credential; re-ingest; confirm only the user with a key gets scored
