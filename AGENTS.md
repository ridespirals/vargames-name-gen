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

**Language / Tooling**

- Go module: `vargames-name-gen`
- Go version: `go 1.23`
- Follows recommendations from **Effective Go** (`https://go.dev/doc/effective_go`), especially:
  - Clear naming, small interfaces, value types where appropriate.
  - Explicit error handling with wrapped errors.
  - Avoid logging secrets; only log safe metadata.

**Key Files (as of now)**

- `main.go`
  - Entrypoint for the CLI.
  - Responsibilities:
    - Loads configuration via `config.Load()`.
    - Builds a logger from `-verbose` flag and/or `Config.Verbose`.
    - When `-fetch` is **empty**:
      - Just validates config and prints:
        - `"Hello from vargames-name-gen (IGDB config loaded, base URL: %s)\n"`.
    - When `-fetch` is **set**:
      - Accepts a **comma-separated list** of entities:
        - Example: `-fetch=games,genres,alternative_names`.
      - Valid entity names (must match IGDB endpoints and the `igdb` package):
        - `games`, `characters`, `genres`, `platforms`, `collections`, `companies`, `alternative_names`.
      - For each entity:
        - Spawns a goroutine that:
          - Creates a dedicated `igdb.Metrics` and `igdb.Client` (with logger + metrics).
          - Creates an `igdb.Fetcher` for that entity.
          - Calls `FetchAll` to page through **up to 500 pages** with limit = `Config.MaxLimit` (default 500).
          - Writes the raw JSON array to:
            - `data/<entity>.json` (e.g. `data/alternative_names.json`).
          - Writes an HTML report to:
            - `data/<entity>-report.html` (e.g. `data/alternative_names-report.html`).
            - Report includes total duration, request counts, retry distribution, and per-request samples.
      - Waits for all goroutines to finish:
        - Fails fast if any entity fetch fails.
        - Logs a summary such as:
          - `"fetched 3 entities in 1m23s"`.

- `src/config/config.go`
  - Package: `config`
  - Provides:
    - **Types**
      - `type Config struct { ClientID, ClientSecret, BaseURL string }`
        - `ClientID` and `ClientSecret`: Twitch app credentials for IGDB OAuth.
        - `BaseURL`: IGDB API base URL, defaults to `https://api.igdb.com/v4`.
    - **Constants**
      - Env var names:
        - `EnvClientID     = "IGDB_CLIENT_ID"`
        - `EnvClientSecret = "IGDB_CLIENT_SECRET"`
        - `EnvBaseURL      = "IGDB_BASE_URL"`
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
          - Optional `IGDB_MAX_LIMIT` (int; default `500`).
          - Optional `IGDB_VERBOSE` / `VARGAMES_VERBOSE` (enable verbose logging; accepts `1`, `true`, `on`, `yes`).
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
  - Comments emphasize **do not commit real credentials**, only examples.

- `go.mod`
  - Module: `vargames-name-gen`
  - Go version: `1.23`

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
    - Examples: `config`, `igdb`, `names`, `cli`, `httpapi`.

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
    - Created via `NewFetcher(client *Client, entity Entity, opts FetcherOptions)`.
    - `FetcherOptions`:
      - `Limit` (per-page limit; default uses `client.MaxLimit()` which reflects `Config.MaxLimit`).
      - `MaxPages` (default 20; in main we use 500 for full exports).
      - `MaxConcurrent` (default 4; concurrency for paging).
      - `Logger` (optional, for progress logs).
    - `FetchAll(ctx)`:
      - Spawns up to `MaxPages` goroutines, each requesting:
        - `fields *; limit <Limit>; offset <page * Limit>;`.
      - Uses a semaphore (`MaxConcurrent`) to avoid unbounded parallelism.
      - Aggregates JSON responses into `[]json.RawMessage`.
      - Logs progress per page and final item count when a logger is set.
      - Returns combined results or the first error encountered.

- `src/igdb/entities.go`
  - `type Entity string` with constants:
    - `games`, `characters`, `genres`, `platforms`, `collections`, `companies`, `alternative_names`.
  - `AllEntities()` and `ValidEntity(string) bool` helpers.

- `src/igdb/logger.go`
  - `Logger` interface with `Logf`.
  - `NoOpLogger`, `StdLogger`, and `VerboseLogger`.
  - `LoggerFromVerbose(bool)` used by `main` to create a logger from flags/env.

- `src/igdb/metrics.go`
  - `Metrics` to record:
    - Per-request records (`PostRecord`: endpoint, retries, duration).
  - Used by `main` and `report.go` to generate HTML fetch reports.

- `src/names/` (not yet implemented)
  - Logic for **name generation** using IGDB data:
    - Simple baselines:
      - Concatenate and mutate real titles.
      - Markov-ish chains over real titles.
    - Weighted by genre or other metadata.

- `src/httpapi/` or `src/server/` (not yet implemented)
  - HTTP handlers exposing:
    - `/generate/game-name` (with optional `genre`, `platform`, `seed` params).
    - `/generate/character-name`, etc.

- `src/cli/` (partially covered by current `main.go`)
  - Commands like:
    - `fetch-games`
    - `generate-game-name`
  - Could be wired through subcommands or flags to the `main` binary.

These are **guidelines** meant to keep the project modular and testable.

---

### 5. How to Continue From Here

Good next steps for any developer or agent:

1. **Leverage fetched data for name generation**
   - Implement a `names` package that:
     - Consumes `data/games.json`, `data/alternative_names.json`, etc.
     - Produces game/character name suggestions using:
       - Concatenation and simple mutation strategies.
       - Markov-style models over existing names.
     - Supports weighting by genre/platform using the fetched metadata.

2. **Add a small HTTP API**
   - Expose endpoints such as:
     - `GET /generate/game-name?genre=<id>&seed=<string>`.
     - `GET /generate/character-name?...`.
   - Internally:
     - Load pre-fetched JSON (or a DB/embedded store) at startup.
     - Delegate generation to the `names` package.

3. **Improve fetch robustness and configurability**
   - Make per-entity `MaxPages`, `MaxConcurrent`, and `MaxLimit` configurable via flags/env.
   - Allow partial results even when some pages or entities fail (with clear reporting).
   - Consider persisting fetch metadata (e.g. last run time, last successful page) in `data/`.

4. **Enhance reporting and observability**
   - Add per-entity summary JSON alongside the HTML reports (for machine consumption).
   - Capture and display IGDB status codes/error messages in the report.
   - Optionally, add Prometheus metrics or structured logs for fetch runs.

Keep this file (`AGENTS.md`) updated when you make structural or architectural changes, especially to:
- Configuration behavior.
- IGDB client, fetcher, retry, and metrics design.
- Name generation strategy.

