## Vargames Name Gen

Go-based tooling for ingesting data from the Internet Game Database (IGDB) and using it as training/seed data for generating game- and character-related names.

| Doc | Purpose |
|-----|---------|
| [`AGENTS.md`](AGENTS.md) | Agent/developer guide — **start here for implementation state** |
| [`planning/README.md`](planning/README.md) | Roadmap, plan index, execution waves |
| [`planning/CORPUS-LIFECYCLE.md`](planning/CORPUS-LIFECYCLE.md) | End-to-end fetch → load → generate flow |

### What's implemented

- **IGDB fetch** — concurrent paging, `-partial`, `-incremental`, per-entity fetch profiles (`minimal` default)
- **Forge generation** — `pick` and `concat` strategies for game titles and character names; genre weighting
- **CLI** — `generate title|character` (no IGDB creds); `cmd/forge` standalone binary; flat `-fetch=` flags for IGDB pulls
- **Offline corpus** — `testdata/corpus/` for dev/CI without credentials

### What's next

See [planning/README.md](planning/README.md). Top items: HTTP API, `fetch` subcommand migration, NAME-QUALITY filters, Markov strategy.

---

### Features

- **Config + env loading**
  - Loads IGDB/Twitch credentials from environment or `.env` (see `.env.example`).
  - Supports:
    - `IGDB_CLIENT_ID`, `IGDB_CLIENT_SECRET`
    - `IGDB_BASE_URL` (default: `https://api.igdb.com/v4`)
    - `IGDB_ACCESS_TOKEN` (optional, pre-generated token; otherwise client-credentials flow is used)
    - `IGDB_MAX_LIMIT` (page size for IGDB queries; IGDB max and default here is `500`)
    - `IGDB_MAX_CONCURRENT` (parallel pages per entity; default `4`)
    - `IGDB_FETCH_PROFILE` (default `minimal`; also `full`, `checksum`)
    - `IGDB_VERBOSE` / `VARGAMES_VERBOSE` (enable verbose logging)
    - `VARGAMES_DATA_DIR` (corpus directory for generation; see `src/cli/flags.go`)

- **IGDB client**
  - Handles Twitch OAuth client‑credentials flow (or uses `IGDB_ACCESS_TOKEN` if set).
  - Adds required headers: `Client-Id` and `Authorization: Bearer <token>`.
  - Implements **retry logic with exponential backoff** for retriable errors (HTTP 429 / 5xx):
    - Configurable min/max backoff and max total retry duration.
    - Max retry count capped (defaults described in `AGENTS.md`).

- **Fetchers**
  - `igdb.Fetcher`:
    - Pages through IGDB endpoints using Apicalypse bodies with **fetch profiles** (default `minimal` — lean field sets per entity; see `planning/FETCH-PROFILES.md`).
    - Uses `GET /<entity>/count` when available for concurrent page planning; falls back to sequential scan.
    - Aggregates results into a single JSON array.
  - **Incremental re-fetch** (`-incremental`): checksum scan + selective `where id = (...)` pulls; writes `data/<entity>-checksums.json`.
  - **Partial failure** (`-partial`): continue other entities/pages; write `data/<entity>.partial.json` when needed.
  - **Metadata**: `data/<entity>-meta.json` after each run (counts, profile, incremental stats).

- **Name generation (`forge`)**
  - Loads IGDB-shaped JSON from `data/` or `testdata/corpus/`
  - **Game titles:** `pick` (reservoir sample) or `concat` (splice/mutate fragments); optional genre filter (`-genre`)
  - **Character names:** same strategies; genre filter via character→game→genre join (`-genre`)
  - Basic quality rejects: length, charset, exact corpus match (`forge.RejectTitle`)
  - Env: `VARGAMES_DATA_DIR` overrides default corpus directory

- **Metrics and reports**
  - Per-request metrics:
    - Endpoint, number of retries, duration.
  - Per-run HTML report per entity:
    - Summary (total items, pages, retries, wall clock).
    - Retry distribution (how many requests had 0,1,2,… retries).
    - Duration distribution (0–500ms, 500ms–1s, etc.).
    - Sample of individual requests.
  - Reports are written to `data/<entity>-report.html`.

---

### Requirements

- Go **1.26** or newer (see `go.mod`).
- IGDB/Twitch API credentials:
  - Create or use an existing application in the Twitch Developer Console.
  - Copy `Client ID` and `Client Secret` into a `.env` (see below).

---

### Setup

1. **Clone the repo**

   ```bash
   git clone <your-fork-or-origin-url> vargames-name-gen
   cd vargames-name-gen
   ```

2. **Create `.env` from `.env.example`**

   ```bash
   cp .env.example .env
   ```

   Edit `.env` and fill in:

   - `IGDB_CLIENT_ID=<your-twitch-client-id>`
   - `IGDB_CLIENT_SECRET=<your-twitch-client-secret>`

   Optional overrides:

   - `IGDB_BASE_URL=https://api.igdb.com/v4`
   - `IGDB_ACCESS_TOKEN=<pre-generated IGDB token>`
   - `IGDB_MAX_LIMIT=500` (IGDB supports up to 500 per page; this is also our default)
   - `IGDB_VERBOSE=1` (or `true`, `on`, `yes`) to enable logging, or use `-verbose` flag.

3. **Build**

   ```bash
   go build .
   ```

---

### CLI usage

Run without arguments to simply validate configuration:

```bash
go run .
```

You should see:

```text
Hello from vargames-name-gen (IGDB config loaded, base URL: https://api.igdb.com/v4)
```

#### Generating names (no IGDB credentials)

Uses committed sample corpus by default (`testdata/corpus/`). Override with `-data-dir=data` after fetching.

```bash
# Game titles
go run . generate title -seed=42 -count=5
go run . generate title -strategy=concat -count=3

# Character names (-genre uses IGDB genre ID from genres.json)
go run . generate character -seed=42 -count=3 -genre=12

# Title names also accept -genre
go run . generate title -seed=42 -count=3 -genre=12

# Standalone forge binary (same commands, no "generate" prefix)
go run ./cmd/forge title -genre=12 -seed=42 -count=5
go run ./cmd/forge character -genre=12 -count=3

# Build forge binary
go build -o forge ./cmd/forge
./forge title -count=3
```

#### Fetching entities (requires IGDB credentials)

Fetch one or more IGDB entities and write them to the `data/` directory.

- **Single entity**

  ```bash
  go run . -fetch=alternative_names
  ```

  This will:

  - Call `POST https://api.igdb.com/v4/alternative_names` with appropriate Apicalypse queries.
  - Page until it exhausts the entity (empty/short page).
  - Write:
    - Raw data: `data/alternative_names.json`
    - Report:   `data/alternative_names-report.html`

- **Multiple entities (parallel)**

  ```bash
  go run . -fetch=games,genres,platforms
  ```

  For each of `games`, `genres`, `platforms`:

  - Creates its own IGDB client + metrics + fetcher (entities run in parallel).
  - Pages within each entity use concurrent workers when `/count` is available (see `-fetch-concurrent`).
  - Writes:
    - `data/<entity>.json`
    - `data/<entity>-meta.json`
    - `data/<entity>-report.html`

  The process exits non‑zero if any entity fetch fails (use `-partial` to keep partial results).

- **Verbose logging**

  ```bash
  go run . -fetch=games,alternative_names -verbose
  ```

  or set `IGDB_VERBOSE=1` / `VARGAMES_VERBOSE=1` in your environment.

  This turns on:

  - Token refresh logs.
  - Retry logs with backoff durations.
  - Per-page progress logs from fetchers.

- **Fetch profiles and incremental sync**

  Default fetch uses the **minimal** profile (only fields needed for forge + checksums). Use `full` for archival `fields *;`:

  ```bash
  go run . -fetch=games -fetch-profile=minimal
  go run . -fetch=games -fetch-profile=full
  ```

  Re-fetch only changed rows (requires existing `data/games.json`):

  ```bash
  go run . -fetch=games -incremental
  ```

  Combine with `-partial` to tolerate page failures during large pulls.

---

- **Data directory**: `data/`
  - Raw IGDB responses:
    - `data/games.json`
    - `data/genres.json`
    - `data/alternative_names.json`
    - etc.
  - Per-entity HTML reports:
    - `data/games-report.html`
    - `data/genres-report.html`
    - `data/alternative_names-report.html`
  - Machine-readable metadata:
    - `data/<entity>-meta.json`
    - `data/<entity>-checksums.json` (after `-incremental` runs)
  - `data/` is git‑ignored.

These JSON files are the **offline corpus** for procedural generation in the [`forge`](src/forge/) package.

---

### Tests

CI (`.github/workflows/ci-tests.yml`) runs `go fix ./...` (must be clean) then `go test -v ./...` on push/PR.

```bash
go fix ./...
go test ./...
```

| Package | Command |
|---------|---------|
| All | `go test ./...` |
| Forge + CLI | `go test ./src/forge/... ./src/cli/... -v` |
| IGDB client/fetcher | `go test ./src/igdb/... -v` |
| Main helpers | `go test . -v` |

Client test gaps and coverage targets: [planning/TEST-COVERAGE.md](planning/TEST-COVERAGE.md).

No live IGDB network calls in automated tests. Corpus benchmarks: `go test -bench=BenchmarkLoadFromDir ./src/forge/...`

---

### Architecture overview

```text
IGDB API ──fetch (main.go -fetch=…)──► data/*.json
                                           │
testdata/corpus ──────────────────────────┤
                                           ▼
                              forge.LoadFromDir → Corpus
                                           │
                    ┌──────────────────────┴──────────────────────┐
                    ▼                                              ▼
            forge/title.GameTitle                      forge/identity.CharacterName
                    │                                              │
                    └──────────► CLI: generate / cmd/forge ◄──────┘
                                           │
                                    (planned: httpapi serve)
```

**Fetch path:** `main.go` → `igdb.Client` + `igdb.Fetcher` → JSON + meta + HTML reports.

**Generate path:** `src/cli/generate.go` → `forge.LoadFromDir` → `title.New` / `identity.New` → stdout.

Details: [`AGENTS.md`](AGENTS.md), [`planning/CORPUS-LIFECYCLE.md`](planning/CORPUS-LIFECYCLE.md).

