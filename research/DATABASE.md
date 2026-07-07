# Research: Storage Options for Fetched IGDB Data

**Status:** Investigation / decision support (not an implementation plan)

**Date:** 2026-07-07

**Context:** Should vargames-name-gen store fetched IGDB data in JSON files (current approach), a SQL database, a NoSQL store, or something else?

**Related planning docs:**

- [planning/EMBEDDED-INDEX.md](../planning/EMBEDDED-INDEX.md) — optional SQLite corpus store (P3)
- [planning/FETCH-ROBUSTNESS.md](../planning/FETCH-ROBUSTNESS.md) — incremental sync, checksums, metadata
- [planning/HTTP-API.md](../planning/HTTP-API.md) — deployed REST generation service
- [planning/NAMES-PACKAGE.md](../planning/NAMES-PACKAGE.md) — in-memory loading from `data/*.json`

---

## Summary

**For the current phase of the project, JSON files are appropriate and likely the right choice.**

A database becomes worth the complexity when you need **incremental sync**, **filtered queries at runtime**, or a **concurrent HTTP API** — not merely because the dataset is large.

| Phase | Recommended storage |
|-------|---------------------|
| Now (CLI fetch + name-gen experiments) | JSON files in `data/` |
| Soon (checksum-based incremental updates) | SQLite (optional layer) |
| Later (deployed HTTP API, shared access) | PostgreSQL (or hosted SQLite if traffic stays tiny) |
| Unlikely first choice | DynamoDB / general NoSQL |

---

## Current State

The pipeline is intentionally simple:

1. Fetch IGDB pages via `igdb.Fetcher.FetchAll`
2. Aggregate into `[]json.RawMessage`
3. Write `data/<entity>.json` and an HTML report

```go
// main.go — write path today
outPath := filepath.Join(dataDir, entity+".json")
raw, _ := json.Marshal(results)
os.WriteFile(outPath, raw, 0644)
```

- `data/` is gitignored (large fetched corpus).
- JSON files are intended as the **offline corpus** for name-generation experimentation.
- The `names` package and HTTP API are not yet implemented.
- README TODOs already point toward `/count` fanout and `checksum`-based change detection — features that push beyond "one big file per entity."

---

## JSON Files (Current Approach)

### Pros

- **Zero infrastructure** — no server, no cost, no ops; aligns with the "100% free" deployment goal.
- **Matches IGDB shape** — raw API responses are preserved exactly; easy to inspect, diff, and re-fetch.
- **Simple write path** — minimal code; fits the current fetch-and-dump workflow.
- **Good for batch workloads** — "load everything, build a model, generate names" is a natural fit.
- **Snapshot semantics** — each file is a point-in-time export of an entity.

### Cons

- **Full re-fetch replaces the whole file** — incremental updates require custom logic on top.
- **Selective access is awkward** — e.g. "games in genre X" means loading and indexing in memory unless you add another layer.
- **Memory pressure at scale** — large entities (`games`, `alternative_names`) can reach hundreds of MB; loading the full array into RAM may be costly.
- **No built-in upsert** — deduplication or update-by-`id` / `checksum` is manual.
- **Poor fit for concurrent access** — awkward for a live API or parallel writers.

### Verdict

Keep JSON while building the `names` package and experimenting locally. It is the right tradeoff for a CLI tool with an offline corpus.

---

## SQLite (Strong "Next Step" Candidate)

SQLite is not Postgres or DynamoDB, but it addresses many upcoming needs without changing the project's character.

### Pros

- **Still a single local file** — like JSON, no server process required.
- **Supports incremental sync** — upsert by `id`, track `checksum`, skip unchanged rows.
- **Queryable indexes** — filter by `name`, `genre_ids`, etc. for weighted generation without loading everything.
- **Hybrid schema** — `JSON` column for raw IGDB payload plus extracted columns for hot query fields.
- **Excellent Go ecosystem** — `modernc.org/sqlite`, `github.com/mattn/go-sqlite3`, etc.
- **Fits CLI-first design** — can later back a small embedded HTTP server without new infrastructure.

### Cons

- **Schema design required** — even a thin schema is more work than `json.Marshal`.
- **Migrations** — IGDB fields or extraction logic changes need versioning.
- **Limited concurrent writers** — fine for a single fetch job; less ideal for many remote clients hitting the same DB file.

### Verdict

Strong candidate when implementing checksum-based updates or genre-weighted generation without loading all of `games.json` into memory. See [planning/EMBEDDED-INDEX.md](../planning/EMBEDDED-INDEX.md) for a proposed schema and trigger criteria.

---

## PostgreSQL (or Similar SQL Server)

### Pros

- **Rich relational queries** — natural joins across games, genres, platforms, characters.
- **JSONB** — store raw IGDB documents while indexing extracted fields.
- **Full-text / trigram search** — useful for name deduplication, fuzzy matching, and search UX.
- **Multi-client concurrency** — background fetch jobs + live API readers.
- **Mature ops tooling** — migrations, backups, observability.

### Cons

- **Infrastructure overhead** — hosting, credentials, backups; friction for a side project.
- **"Free" has limits** — managed free tiers cap storage/connections; self-hosting adds ops.
- **Overkill for local-only experimentation** — unnecessary before a deployed API exists.
- **Heavier local dev** — Docker compose or remote DB for every contributor.

### Verdict

Appropriate when deploying the HTTP API (`/generate/game-name?genre=...`) with shared, queryable storage and concurrent access. Not needed for the first version of name generation.

---

## DynamoDB / NoSQL (DynamoDB, DocumentDB, etc.)

### Pros

- **Horizontal scale** — if traffic or corpus size grows substantially.
- **Natural key-value access** — IGDB `id` as partition key is straightforward.
- **AWS free tier** — can work for low-traffic APIs if already on AWS.
- **Schema-flexible documents** — store whole IGDB records without upfront normalization.

### Cons

- **Query patterns are the hard part** — genre/platform filters need GSIs, denormalization, or pre-computed indexes.
- **Poor fit for batch model building** — "scan all game names and build a Markov chain" still needs a full export or scan.
- **Cost and design complexity** — RCUs, GSIs, hot partitions; easy to overspend or under-design.
- **Vendor coupling** — local dev is clunkier than SQLite or Postgres.
- **Workload mismatch** — this project is read-heavy batch + occasional sync, not high-velocity OLTP.

### Verdict

Usually not the right first database unless explicitly building on AWS serverless with predominantly key-based access patterns.

---

## Comparison Matrix

| Need | JSON | SQLite | Postgres | DynamoDB |
|------|------|--------|----------|----------|
| Local CLI, minimal setup | ✅ | ✅ | ❌ | ❌ |
| Offline name-gen experiments | ✅ | ✅ | ✅ | ⚠️ |
| Incremental sync via checksum | ❌ | ✅ | ✅ | ✅ |
| Filter by genre/platform at runtime | ⚠️ in-memory | ✅ | ✅ | ⚠️ design-heavy |
| Deployed HTTP API | ⚠️ | ⚠️ | ✅ | ✅ |
| Zero infra / free | ✅ | ✅ | ⚠️ | ⚠️ |
| Full-corpus batch scans | ✅ | ✅ | ✅ | ⚠️ expensive scans |
| Multi-writer concurrency | ❌ | ⚠️ | ✅ | ✅ |

---

## Decision Triggers

Add a queryable store when a **concrete feature** needs it — not preemptively.

| Trigger | Suggested direction |
|---------|---------------------|
| Implementing `names` package v1 | Stay on JSON; load at startup |
| Checksum-based incremental fetch | SQLite upserts (or Postgres if API already exists) |
| `LoadFromDir` > ~5s or high RAM on target hardware | SQLite or embedded index (see EMBEDDED-INDEX plan) |
| Genre/platform filtering without full memory load | SQLite with indexes |
| Deployed HTTP API with concurrent readers | PostgreSQL (JSONB + indexed columns) |
| AWS serverless-only deployment, key lookups dominate | Revisit DynamoDB |

---

## Recommended Progression

### 1. Now — JSON files

- Implement `names` against `data/*.json`.
- Validate generation strategies before adding storage complexity.
- Optionally add `data/<entity>-meta.json` for counts, checksums, and fetch metadata (see FETCH-ROBUSTNESS plan) without introducing a database.

### 2. Soon — optional SQLite layer

- Fetch pipeline upserts rows by `id` + `checksum`.
- Keep JSON export as a debug/snapshot format if desired.
- Defer until profiling or incremental sync work actually starts (EMBEDDED-INDEX trigger criteria).

### 3. Later — PostgreSQL for deployed API

- When the HTTP API needs shared concurrent access and richer queries.
- Consider `JSONB` for raw documents + extracted columns for `name`, `genres`, `platforms`.

### 4. Skip DynamoDB unless

- Already committed to AWS serverless architecture.
- Access pattern is overwhelmingly "get by IGDB id" rather than "filter corpus by metadata."

---

## Hybrid Pattern

A practical long-term shape:

```
IGDB API (source of truth)
    ↓ fetch
Queryable store (SQLite locally, Postgres in production)
    ↓ optional export
JSON snapshots (debugging, fixtures, SAMPLE-CORPUS)
    ↓ load
names.Generator (in-memory indexes for hot paths)
```

Treat IGDB as the source of truth. Use **files for snapshots** and a **queryable store only when a feature demands it**.

---

## Open Questions

- **At what corpus size does in-memory JSON become unacceptable?** Needs profiling on real `games.json` / `alternative_names.json` after a full fetch.
- **JSON meta files vs SQLite for incremental sync?** Meta JSON may be enough for change detection before a full DB layer.
- **Embed SQLite in the binary vs separate `data/corpus.db`?** Separate file is easier to inspect and regenerate.
- **Postgres hosting for "100% free"?** Options include Neon/Supabase free tiers, Railway, or SQLite behind a single-instance API on Fly.io free tier — each has tradeoffs.

---

## Conclusion

JSON files are not a temporary hack for this project — they are a deliberate fit for the current CLI + offline corpus model. The upgrade path is clear: **JSON → SQLite (local/queryable) → Postgres (deployed API)**, with NoSQL only if cloud architecture and access patterns specifically warrant it.

When implementation begins, prefer the existing [EMBEDDED-INDEX](../planning/EMBEDDED-INDEX.md) plan for SQLite specifics rather than duplicating schema design here.
