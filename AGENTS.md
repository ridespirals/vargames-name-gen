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

- `src/main.go`
  - Entrypoint, currently just loads configuration and prints a confirmation:
    - Uses `config.Load()` to obtain `Config` (value, not pointer).
    - On error, logs and exits:
      - `log.Fatalf("config: %v (set %s and %s, e.g. from .env)", ...)`
    - On success, prints:
      - `"Hello from vargames-name-gen (IGDB config loaded, base URL: %s)\n"`
  - Does **not** yet make IGDB HTTP calls; that is the next logical step.

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
        - Fails if required vars missing:
          - `missing required environment variable IGDB_CLIENT_ID`
          - `missing required environment variable IGDB_CLIENT_SECRET`
        - Uses `DefaultBaseURL` if `IGDB_BASE_URL` is empty.
        - Returns a **value** `Config` (not `*Config`), which is treated as immutable.

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

### 4. Anticipated Future Structure

This is not implemented yet, but planned and recommended:

- `src/igdb/`
  - `client.go`: `type Client` with:
    - Auth handling (Client ID/secret → Twitch OAuth token).
    - `BaseURL` from `config.Config`.
    - Methods like:
      - `GetGames(ctx context.Context, query GameQuery) ([]Game, error)`
      - `GetGenres(ctx context.Context, ids []int64) ([]Genre, error)`
  - `models.go`: light structs matching IGDB entities we care about (`Game`, `Genre`, `Platform`, etc.).

- `src/names/`
  - Logic for **name generation** using IGDB data:
    - Simple baselines:
      - Concatenate and mutate real titles.
      - Markov-ish chains over real titles.
    - Weighted by genre or other metadata.

- `src/httpapi/` or `src/server/`
  - HTTP handlers exposing:
    - `/generate/game-name` (with optional `genre`, `platform`, `seed` params).
    - `/generate/character-name`, etc.

- `src/cli/`
  - Commands like:
    - `fetch-games`
    - `generate-game-name`
  - Could be wired through subcommands or flags to the `main` binary.

These are **guidelines** meant to keep the project modular and testable.

---

### 5. How to Continue From Here

Good next steps for any developer or agent:

1. **Implement the IGDB client skeleton**
   - Create `src/igdb/client.go`.
   - Add a `Client` that holds:
     - `http.Client`
     - `Config` (or a smaller `Credentials` struct)
     - Cached access token + expiry.
   - Implement:
     - A method to fetch and cache a Twitch access token.
     - A small method to call one IGDB endpoint (e.g., basic `/games` query).

2. **Add a minimal CLI or HTTP endpoint**
   - Example: `main.go` could:
     - Initialize `Config`.
     - Create `igdb.Client`.
     - Fetch a few games and print their names as a smoke test.

3. **Begin experimenting with name generation**
   - Add a `names` package that:
     - Accepts a slice of IGDB game names.
     - Produces simple generated names.
   - Later, add options for weighting by genre and more involved algorithms.

Keep this file (`AGENTS.md`) updated when you make structural or architectural changes, especially to:
- Configuration behavior.
- IGDB client design.
- Name generation strategy.

