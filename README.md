## Vargames Name Gen

Go-based tooling for ingesting data from the Internet Game Database (IGDB) and using it as training/seed data for generating game- and character-related names.

For deeper architectural notes and agent-facing guidance, see [`AGENTS.md`](AGENTS.md).

---

### TODO / Future

- Use `/count` endpoints to enable fanout
  - We'd be able to determine how many "fetchers" we could use to pull all the data in parallel
  - Could also possibly optimize for number of requests or time or fetchers or whatever
  - Could be a basic change detection feature (if we store the count we found when we last ran, we could at least detect if there are new entities. We'd need another mechanism for modified data, though)

---

### Features

- **Config + env loading**
  - Loads IGDB/Twitch credentials from environment or `.env` (see `.env.example`).
  - Supports:
    - `IGDB_CLIENT_ID`, `IGDB_CLIENT_SECRET`
    - `IGDB_BASE_URL` (default: `https://api.igdb.com/v4`)
    - `IGDB_ACCESS_TOKEN` (optional, pre-generated token; otherwise client-credentials flow is used)
    - `IGDB_MAX_LIMIT` (page size for IGDB queries; IGDB max and default here is `500`)
    - `IGDB_VERBOSE` / `VARGAMES_VERBOSE` (enable verbose logging)

- **IGDB client**
  - Handles Twitch OAuth client‑credentials flow (or uses `IGDB_ACCESS_TOKEN` if set).
  - Adds required headers: `Client-Id` and `Authorization: Bearer <token>`.
  - Implements **retry logic with exponential backoff** for retriable errors (HTTP 429 / 5xx):
    - Configurable min/max backoff and max total retry duration.
    - Max retry count capped (defaults described in `AGENTS.md`).

- **Fetchers**
  - `igdb.Fetcher`:
    - Pages through IGDB endpoints (e.g. `/games`, `/genres`, `/alternative_names`) using Apicalypse bodies:
      - `fields *; limit <N>; offset <page * N>;`
    - Walks pages **sequentially** for a given entity:
      - Starts at offset `0`.
      - Increments offset by `limit` each page.
      - Stops when a page is empty or shorter than `limit`.
    - Aggregates results into a single JSON array.

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

- Go **1.23** or newer (see `go.mod`).
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

#### Fetching entities

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

  - Creates its own IGDB client + metrics + fetcher (per-entity concurrency).
  - Fetches pages **sequentially** for that entity; entities are processed in parallel.
  - Writes:
    - `data/<entity>.json`
    - `data/<entity>-report.html`

  The process exits non‑zero if any entity fetch fails.

- **Verbose logging**

  ```bash
  go run . -fetch=games,alternative_names -verbose
  ```

  or set `IGDB_VERBOSE=1` / `VARGAMES_VERBOSE=1` in your environment.

  This turns on:

  - Token refresh logs.
  - Retry logs with backoff durations.
  - Per-page progress logs from fetchers.

---

### Data & reports

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
  - `data/` is git‑ignored.

These JSON files are intended to be the **offline corpus** for later name-generation experimentation (e.g. in a future `names` package).

---

### Tests

- **Unit tests** for the IGDB client and fetcher:

  ```bash
  go test ./src/igdb/... -v
  ```

- **Integration fetch test** (requires real IGDB credentials, may be slow / large):

  ```bash
  go test -v -run TestFetchAlternativeNames
  ```

  This test:

  - Fetches all `alternative_names`.
  - Asserts an expected count (based on known snapshot).
  - Writes to `data/alternative_names.json`.

Use `-short` to skip this integration test:

```bash
go test -short ./...
```

---

### Architecture overview

High‑level flow (for fetches):

1. `main.go`
   - Parses flags and config.
   - Determines which entities to fetch.
   - For each entity:
     - Creates `Metrics` + `Client` + `Fetcher`.
     - Runs `FetchAll` and writes JSON + HTML report.

2. `config.Config`
   - Encapsulates all environment‑driven configuration.
   - Acts as a simple immutable value passed into the IGDB client.

3. `igdb.Client`
   - Handles auth, retries, backoff, and low‑level HTTP transport.

4. `igdb.Fetcher`
   - Implements paging for specific IGDB entities (`games`, `genres`, etc.).

5. `igdb.Metrics` + `report.go`
   - Records and visualizes performance and reliability characteristics.

For more detail on types, options, and future plans, see **`AGENTS.md`**.

