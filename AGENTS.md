## Vargames Name Gen – Agent & Developer Guide

This document captures the current intent and design of the project so other developers or AI agents can come up to speed quickly and extend it consistently.

---

### 1. Project Purpose

- **Goal**: Build a Go service/tool that:
  - Calls the **Internet Game Database (IGDB)** APIs (via Twitch authentication).
  - Pulls data for various entities: **Games, Characters, Genres, Platforms, Collections**, etc.
  - Processes fields such as **names, descriptions, genres**, and related metadata.
  - Uses this data as **training/seed data** to generate:
    - Random or weighted **game names**.
    - Random or weighted **character names**.
    - Potentially other “name-like” artifacts (collections, series, etc.).
  - Allows weighting by attributes (e.g., genre, platform) when generating new names.

- **Long-term vision**:
  - A small, idiomatic Go codebase that can:
    - Act as a **CLI tool**, a **micro HTTP API**, or both.
    - Support experimentation with different name-generation strategies (recombination, Markov-ish approaches, filters by genre/era, etc.).

---

### 2. Current State of the Codebase

**Implementation snapshot (sync with `planning/README.md` before large changes):**

| Area | Status | Key paths |
|------|--------|-----------|
| IGDB fetch pipeline | **~85% done** | `main.go`, `src/igdb/` — concurrent paging, partial, incremental, profiles |
| Forge corpus load | **~90% done** | `src/forge/corpus.go` — `LoadFromDir`, 7 entities |
| Title generation | **~60% done** | `src/forge/title/` — `GameTitle`, `pick`/`concat`, genre filter |
| Identity generation | **~55% done** | `src/forge/identity/` — `CharacterName`, genre join via games |
| CLI `generate` | **Done** | `src/cli/generate.go`, `cmd/forge/` — title + character |
| CLI `fetch` subcommand | **Pending** | Still flat `-fetch=` flags in `main.go` |
| HTTP API | **Not started** | No `src/httpapi/` |
| Sample corpus | **Done** | `testdata/corpus/` (7 JSON files + README) |
| CI | **Basic** | `go fix` + `go test ./...` on push/PR |

**Next priorities:** NAME-QUALITY Q1+, `fetch` subcommand migration, HTTP API H1–H4. See [planning/README.md](planning/README.md).

**Language / Tooling**

- Go module: `vargames-name-gen`
- Go version: `go 1.26`
- After upgrading Go or editing code, run `go fix ./...` and commit any changes — CI fails if modernizers would modify the tree.
- Follows recommendations from **Effective Go** (`https://go.dev/doc/effective_go`), especially:
  - Clear naming, small interfaces, value types where appropriate.
  - Explicit error handling with wrapped errors.
  - Avoid logging secrets; only log safe metadata.

**Key Files (as of now)**

- `main.go`
  - Entrypoint for the CLI.
  - **Subcommand dispatch:** if `os.Args[1] == "generate"`, delegates to `cli.RunGenerate` **before** `config.Load()` (no IGDB credentials needed).
  - Responsibilities (fetch / default path):
    - Loads configuration via `config.Load()`.
    - Builds a logger from `-verbose` flag and/or `Config.Verbose`.
    - When `-fetch` is **empty** and no `generate` subcommand:
      - Validates config and prints:
        - `"Hello from vargames-name-gen (IGDB config loaded, base URL: %s)\n"`.
    - When `-fetch` is **set**:
      - Accepts a **comma-separated list** of entities:
        - Example: `-fetch=games,genres,alternative_names`.
      - Valid entity names (must match IGDB endpoints and the `igdb` package):
        - `games`, `characters`, `genres`, `platforms`, `collections`, `companies`, `alternative_names`.
      - CLI flags (also see `config` env vars):
        - `-fetch-limit`, `-fetch-concurrent` — page size and parallel pages per entity.
        - `-partial` — continue on page/entity failures; write `data/<entity>.partial.json` when some pages succeed; exit `1` if any entity failed.
        - `-fetch-profile` — Apicalypse field profile: `full`, `minimal` (default), or `checksum`.
        - `-incremental` — checksum scan + selective ID pulls for changed/new rows (requires existing `data/<entity>.json` for updates; otherwise full fetch).
      - For each entity:
        - Spawns a goroutine that:
          - Creates a dedicated `igdb.Metrics` and `igdb.Client` (with logger + metrics).
          - Resolves fetch profile via `igdb.ProfileFor`.
          - Creates an `igdb.Fetcher` with profile `QueryPrefix`.
          - Calls `FetchAllResult` or `FetchIncrementalResult` (when `-incremental`).
          - Writes:
            - Raw data: `data/<entity>.json` (or `.partial.json` on partial failure).
            - Metadata: `data/<entity>-meta.json` (counts, profile, incremental stats).
            - Checksums: `data/<entity>-checksums.json` (after incremental runs).
            - HTML report: `data/<entity>-report.html`.
      - Waits for all goroutines to finish:
        - Exits `1` if any entity fetch failed (unless `-partial` wrote partial data).
        - Logs per-entity summaries and `"fetched N entities in …"`.

- `src/config/config.go`
  - Package: `config`
  - Provides:
    - **Types**
      - `type Config struct` includes:
        - `ClientID`: Twitch app client ID (IGDB OAuth).
        - `ClientSecret`: Twitch app client secret (IGDB OAuth).
        - `BaseURL`: IGDB API base URL, defaults to `https://api.igdb.com/v4`.
        - `AccessToken`: optional pre-generated token; if set, the client skips Twitch token fetching.
        - `MaxLimit`: default IGDB per-page `limit` (page size); defaults to 500.
        - `MaxConcurrent`: parallel page workers per entity fetch; defaults to 4.
        - `FetchProfile`: default Apicalypse profile name (`minimal`); see `igdb` profiles.
        - `Verbose`: enables default verbose logging.
    - **Constants**
      - Env var names:
        - `EnvClientID     = "IGDB_CLIENT_ID"`
        - `EnvClientSecret = "IGDB_CLIENT_SECRET"`
        - `EnvBaseURL      = "IGDB_BASE_URL"`
        - `EnvAccessToken  = "IGDB_ACCESS_TOKEN"`
        - `EnvMaxLimit     = "IGDB_MAX_LIMIT"`
        - `EnvMaxConcurrent = "IGDB_MAX_CONCURRENT"`
        - `EnvFetchProfile  = "IGDB_FETCH_PROFILE"`
        - `EnvVerbose      = "IGDB_VERBOSE"`
        - `EnvVerboseAlt   = "VARGAMES_VERBOSE"`
      - Default:
        - `DefaultBaseURL = "https://api.igdb.com/v4"`
    - **Functions**
      - `LoadEnv() error`
        - Reads `.env` from the current working directory.
        - Each non-empty, non-comment line must be `KEY=VALUE`.
        - Behavior:
          - If `.env` does **not** exist: returns `nil`.
          - If `.env` exists:
            - Uses `bufio.Scanner` to process line by line.
            - Tracks `lineNum` and:
              - Returns an error if there is **no `=`** in a non-comment line:
                - `"<file>:<line>: invalid line (missing KEY=VALUE)"`
              - Returns an error if the key is empty:
                - `"<file>:<line>: empty key"`
            - Trims whitespace around key and value.
            - Strips **optional surrounding single or double quotes** on values.
            - Only calls `os.Setenv(key, value)` when that env var is **currently unset**.
              - If `Setenv` fails, returns `fmt.Errorf("setenv %s: %w", key, err)`.
            - On scanner error, wraps as `fmt.Errorf("scan %s: %w", envFile, err)`.
      - `Load() (Config, error)`
        - First calls `LoadEnv()`; on error returns `Config{}` with `fmt.Errorf("load env: %w", err)`.
        - Reads from environment:
          - `IGDB_CLIENT_ID`
          - `IGDB_CLIENT_SECRET`
          - Optional `IGDB_BASE_URL`
          - Optional `IGDB_ACCESS_TOKEN` (pre-generated IGDB access token; if present, skips Twitch OAuth).
          - Optional `IGDB_MAX_LIMIT` (int; default `500`, IGDB hard max is 500 per page).
          - Optional `IGDB_MAX_CONCURRENT` (int; default `4`).
          - Optional `IGDB_FETCH_PROFILE` (`full`, `minimal`, `checksum`; default `minimal`).
          - Optional `IGDB_VERBOSE` / `VARGAMES_VERBOSE` (enable verbose logging; accepts `1`, `true`, `on`, `yes`).
  - Optional `VARGAMES_DATA_DIR` (corpus path for `generate` / `cmd/forge`; not read by `config.Load()` — see `src/cli/flags.go`).
        - Fails if required vars missing:
          - `missing required environment variable IGDB_CLIENT_ID`
          - `missing required environment variable IGDB_CLIENT_SECRET`
        - Uses `DefaultBaseURL` if `IGDB_BASE_URL` is empty.
        - Uses `DefaultMaxLimit` if `IGDB_MAX_LIMIT` is unset or invalid.
        - Returns a **value** `Config` (not `*Config`), which is treated as immutable.
        - `Config.Verbose` controls default verbose logging for the client and fetchers.

- `.env.example`
  - Documents expected env vars:
    - `IGDB_CLIENT_ID=`
    - `IGDB_CLIENT_SECRET=`
    - Optional override:
      - `# IGDB_BASE_URL=https://api.igdb.com/v4`
    - Optional:
      - `# IGDB_ACCESS_TOKEN=`
      - `# IGDB_MAX_LIMIT=500`
      - `# IGDB_MAX_CONCURRENT=4`
      - `# IGDB_FETCH_PROFILE=minimal`
      - `# IGDB_VERBOSE=1` (or set `VARGAMES_VERBOSE=1`)
  - Comments emphasize **do not commit real credentials**, only examples.

- `go.mod`
  - Module: `vargames-name-gen`
  - Go version: `1.26`

**Bruno collections**

- Several Bruno files exist under `bruno/` (e.g., `games.yml`, `genres.yml`, `platforms.yml`, `collections.yml`, etc.).
- These appear to be or will be **API test collections** for hitting IGDB endpoints.
- They’re useful as examples of how requests should look (e.g., endpoints, headers, query bodies).

---

### 3. Design and Style Decisions (Guidance for Future Work)

**Config and Environment**

- Always use `config.Load()` in entrypoints to initialize IGDB credentials.
- Treat `Config` as **immutable** once created; pass by value into constructors (e.g., future `igdb.Client`).
- `LoadEnv()`:
  - Is intentionally **strict** about malformed `.env` lines:
    - This encourages fail-fast behavior in local/dev setups.
  - Will not override already-set environment variables (so container/orchestration config wins over `.env`).

**Secrets**

- **Never** log or print:
  - `ClientID`
  - `ClientSecret`
- It is acceptable to:
  - Log or print `BaseURL`.
  - Log the fact that config is loaded.
- Any future logging around IGDB:
  - Should log HTTP method, path, status code, and high-level error info.
  - Must **not** log raw tokens, secrets, or full responses that might contain private data.

**Go Style (Effective Go influenced)**

- Prefer **small interfaces** with semantic names (e.g., `TokenSource`, `GameFetcher`) instead of large, generic ones.
- Use **value receivers** where types are small and logically immutable; use **pointer receivers** when:
  - The method mutates the receiver.
  - The type is large or copying would be expensive.
- Errors:
  - Return `error` as the **final return value**.
  - Wrap errors using `fmt.Errorf("context: %w", err)` to preserve context.
  - Use human-readable, origin-identifying messages (`"igdb: get games: %w"`).
- Package naming:
  - Lowercase, no underscores; short but evocative:
    - Examples: `config`, `igdb`, `forge`, `cli`, `httpapi`.

---

### 4. IGDB Client, Fetcher, and Metrics (Current Implementation)

- `src/igdb/client.go`
  - `type Client`:
    - Holds:
      - `config.Config` (credentials, base URL, max limit, verbose flag).
      - `*http.Client` with a 30s timeout.
      - Cached OAuth token + expiry.
      - Retry/backoff configuration.
      - Optional `Logger` and `*Metrics`.
    - Authentication:
      - If `Config.AccessToken` is non-empty:
        - Uses it directly as `Authorization: Bearer <token>`.
      - Otherwise:
        - Fetches an access token from Twitch using client-credentials:
          - `https://id.twitch.tv/oauth2/token?client_id=...&client_secret=...&grant_type=client_credentials`.
        - Caches token and expiry; refreshes slightly early.
    - `Post(ctx, endpoint, body)`:
      - Sends `POST` to `Config.BaseURL + "/" + endpoint` with:
        - `Client-Id: <ClientID>`.
        - `Authorization: Bearer <access token>`.
        - `Content-Type: text/plain`.
      - Retry semantics:
        - Retries **up to 10** times on retriable errors (HTTP 429 or `5xx`).
        - Exponential backoff:
          - `DefaultRetryMinBackoff` (currently ~seconds) doubling each attempt.
          - Capped at `DefaultRetryMaxBackoff` (5 minutes).
        - Total time spent in backoff is capped by `DefaultMaxRetryDuration` (1 hour).
        - Non-retriable errors (e.g. 400/401/403) fail immediately.
      - Logging:
        - When verbose, logs token refresh events, retries, and successful responses (endpoint + bytes).
      - Metrics:
        - If a `*Metrics` is attached via `WithMetrics`, records:
          - Endpoint.
          - Retries used.
          - Total duration for that call.

- `src/igdb/fetcher.go`
  - `type Fetcher`:
    - Created via `NewFetcher(client FetcherClient, entity Entity, opts FetcherOptions)`.
    - `FetcherOptions`:
      - `Limit` (per-page limit; default uses `client.MaxLimit()` which reflects `Config.MaxLimit`, default 500).
      - `MaxConcurrent` (parallel page workers; used by concurrent `FetchAllResult`).
      - `QueryPrefix`:
        - Apicalypse query fragment inserted before paging clauses (`limit`/`offset`).
        - Typically from fetch profile via `ProfileFor` (`minimal` default; `full` uses `fields *;`).
      - `Logger` (optional, for progress logs).
    - `FetchAllResult(ctx, allowPartial)`:
      - Uses `GET /<entity>/count` when available to plan concurrent page fetches; falls back to sequential paging.
      - Stops when all pages complete or on error (partial results when `allowPartial`).
      - Returns `FetchResult` with items, counts, and optional error.
    - `FetchIncrementalResult(ctx, existing, dataProfile, allowPartial)`:
      - Checksum scan (`fields id,checksum;`) → diff vs existing corpus → batch `where id = (...);` pulls with minimal profile → merge.
      - Writes checksum map via `main` to `data/<entity>-checksums.json`.
      - No existing corpus → full fetch equivalent.

- `src/igdb/profiles.go`
  - `Profile`, `ProfileFor(entity, name)` registry: `full`, `minimal` (default), `checksum`.
  - Per-entity minimal field sets for all 7 entities (see `planning/FETCH-PROFILES.md`).

- `src/igdb/incremental.go`
  - Incremental diff/merge logic used by `FetchIncrementalResult`.

- `src/igdb/fetch_result.go`
  - `FetchResult`, `IncrementalStats` for fetch outcomes and meta JSON.

- `fetchmeta.go` / `fetchio.go` (package `main`)
  - `writeFetchMeta`, `loadPriorFetchMeta`, `loadExistingEntityJSON`, `writeChecksumsJSON`, entity JSON writers.

- `src/igdb/entities.go`
  - `type Entity string` with constants:
    - `games`, `characters`, `genres`, `platforms`, `collections`, `companies`, `alternative_names`.
  - `AllEntities()` and `ValidEntity(string) bool` helpers.
  - `QueryPrefixForEntity` uses the default **minimal** profile.

- `src/igdb/client.go` (also)
  - `Get(ctx, endpoint)` and `Count(ctx, entity)` share the same retry/backoff path as `Post`.

- `src/igdb/logger.go`
  - `Logger` interface with `Logf`.
  - `NoOpLogger`, `StdLogger`, and `VerboseLogger`.
  - `LoggerFromVerbose(bool)` used by `main` to create a logger from flags/env.

- `src/igdb/metrics.go`
  - `Metrics` to record:
    - Per-request records (`PostRecord`: endpoint, retries, duration).
  - Used by `main` and `report.go` to generate HTML fetch reports.

- `src/forge/` — corpus loader; see `planning/FORGE-PACKAGE.md`
  - `corpus.go` — `LoadFromDir`, `Corpus`, minimal structs for all 7 entities
  - `normalize.go` — `NewRand`, `NormalizeTitle`, `TokenizeTitle`, `RejectTitle`, `SourceTitleSet` (Q0 quality rules)
  - `testutil.go` — `TestCorpusDir(t)` for tests
  - `src/forge/title/` — `Generator`, `GameTitle(opts)`; strategies `pick`, `concat`; `Options.GenreID`, `Options.Strategy`, `Options.Seed`
    - Genre filter via `filterGamePool`; subtitle mutations in `extras.go`
    - **Pending:** `CollectionTitle`, `markov`, platform weighting, strategy registry
  - `src/forge/identity/` — `Generator`, `CharacterName(opts)`; strategies `pick`, `concat`
    - Genre filter via character→game→genre join in `filter.go`
    - **Pending:** `CompanyName`, markov, strategy registry

- `src/cli/` — shared CLI logic (`planning/CLI-STRUCTURE.md`)
  - `generate.go` — `RunGenerate`, `RunForge`, `GenerateTitles`, `GenerateIdentities`
  - `flags.go` — `DefaultDataDir()` → `VARGAMES_DATA_DIR`, else `testdata/corpus` (if present), else `data/`
  - Subcommands: `generate title|game`, `generate character|identity`
  - Title and character flags: `-data-dir`, `-seed`, `-count`, `-strategy` (`pick`|`concat`), `-genre` (IGDB genre ID; 0 = all)
  - **Pending:** `fetch.go`, `serve.go`, `validate.go`, `list.go`

- `cmd/forge/main.go` — standalone binary calling `cli.RunForge`; no IGDB credentials ever

- `src/httpapi/` — **not implemented** (planned: `serve` subcommand, `/health`, `/generate/*`)

- `planning/` — design docs and roadmap; start at `planning/README.md`
- `research/` — APICALYPSE reference, corpus loading benchmarks, storage options

These are **guidelines** meant to keep the project modular and testable.

---

### 5. How to Continue From Here

**Immediate next work (Wave 2):**

1. **Forge polish** — platform weighting (M3b), corpus mtime cache (M7)
2. **NAME-QUALITY** — `forge/quality` subpackage (profanity, batch dedup); Q0 already in `RejectTitle`
3. **CLI-STRUCTURE** — migrate `-fetch` → `fetch` subcommand; add `validate corpus`, `list genres`
4. **TEST-COVERAGE** — close client gaps (`TestPost_TokenFetchedOnce`, etc.); CI vet/build

**Wave 3 (after forge + quality):**

5. **HTTP-API** — `src/httpapi/`, `serve` subcommand, `/generate/game-name` + `/generate/character-name`
6. **IGDB-COMPLIANCE** — attribution in reports/API before public deploy
7. **CI-INFRA** — hardened CI, GitHub Pages for fetch reports

**Mostly done (maintain, don't re-plan):**

- IGDB fetch: concurrent paging, `-partial`, `-incremental`, fetch profiles (`planning/FETCH-ROBUSTNESS.md`, `FETCH-PROFILES.md`)
- Forge M1–M4: load, title/character generation with genre filtering (`planning/FORGE-PACKAGE.md`)

**Deferred until profiling demands it:**

- SQLite embedded index (`planning/EMBEDDED-INDEX.md`)
- Markov strategy (`FORGE-PACKAGE` M5)
- Fetch resume/checkpoints (`planning/FETCH-RESUME.md`)

Keep this file (`AGENTS.md`) updated when you make structural or architectural changes, especially to:
- Configuration behavior.
- IGDB client, fetcher, retry, and metrics design.
- Name generation strategy.

