# Plan: `forge` Package — Procedural Generation from IGDB Data

## Goal

Turn fetched JSON in `data/` into a reusable library that forges **game-world text artifacts** from IGDB corpora: titles, character names, collection names, company labels, and related patterns — with optional genre weighting and reproducible output via seeds.

## Current Foundation

- **Fetch output:** `data/<entity>.json` — arrays of raw IGDB objects
- **Entities loaded:** `games`, `characters`, `alternative_names`, `collections`, `companies`, `genres`, `platforms`
- **`src/forge/`** — corpus loader (`LoadFromDir`, minimal structs)
- **`src/forge/title/`** — title-family generator shell (games, collections, subtitle extras)
- **`src/forge/identity/`** — identity-family generator shell (characters, companies)
- **`main.go`** — fetch; `generate title` subcommand (no IGDB creds)
- **`cmd/forge`** — standalone forge CLI (dev loop, no IGDB creds)
- **`src/cli/`** — shared `GenerateTitles`, `RunGenerate`, `RunForge`

## Proposed Architecture

```mermaid
flowchart TB
  subgraph load [forge — load]
    JSON[data/*.json]
    Corpus[Corpus]
    JSON --> Corpus
  end

  subgraph title_pkg [forge/title]
    Corpus --> TitleGen[Generator]
    TitleGen --> GameTitle[GameTitle]
    TitleGen --> CollectionTitle[CollectionTitle]
    AltExtras[alternative_names extras]
    SubExtras[subtitle patterns]
    AltExtras --> TitleGen
    SubExtras --> TitleGen
  end

  subgraph identity_pkg [forge/identity]
    Corpus --> IdentityGen[Generator]
    IdentityGen --> CharName[CharacterName]
    IdentityGen --> CompanyName[CompanyName]
    GameLink[game linkage for weighting]
    GameLink --> IdentityGen
  end

  subgraph shared [forge — shared utilities]
    Normalize[normalize]
    Seed[seed]
    Corpus --> Normalize
  end
```

## Package Layout

```
src/forge/
  doc.go
  corpus.go          # LoadFromDir, Corpus, minimal entity structs
  corpus_test.go
  testutil.go        # TestCorpusDir
  normalize.go       # shared tokenization (M2+)
  seed.go            # deterministic PRNG (M2+)

src/forge/title/
  doc.go
  generator.go       # title.Generator
  options.go         # title-specific Options (M2+)
  concat.go          # splice/mutate game titles (M2)
  extras.go          # subtitles, alt-name tokens (M2+)
  markov.go          # title Markov models (M5)

src/forge/identity/
  doc.go
  generator.go       # identity.Generator
  options.go         # identity-specific Options (M4+)
  generate.go        # character / company generation (M4)
  extras.go          # epithets, game-link weighting (M4+)
```

## Data Model (Minimal Structs)

Parse only fields needed for generation and weighting. Do not mirror full IGDB schemas.

| Entity | Package consumer | Fields |
|--------|------------------|--------|
| `games` | `title`, `identity` (linkage) | `id`, `name`, `genres[]`, `platforms[]` |
| `characters` | `identity` | `id`, `name`, `games[]` |
| `companies` | `identity` | `id`, `name` |
| `collections` | `title` | `id`, `name` |
| `alternative_names` | `title` (mutation tokens) | `name`, `game` |
| `genres` | both (weighting) | `id`, `name` |
| `platforms` | reference metadata (loaded; not used for title filtering) | `id`, `name` |

## Public API (v1)

```go
// forge — load once, share across generators
corpus, err := forge.LoadFromDir("data")

titles := title.New(corpus)
identities := identity.New(corpus)

// title.Options — genre filter, strategy, seed
name, err := titles.GameTitle(title.Options{GenreID: &rpgID, Seed: 42})

// identity.Options — game linkage, seed
char, err := identities.CharacterName(identity.Options{Seed: 42})
```

Each subpackage owns its `Options` and extras; shared concerns (`normalize`, `seed`) live in `forge`.

## Generation Strategies (Phased)

### Phase 1 — Baseline (`forge/title`, ship first)

1. **Reservoir sampling** from a genre-filtered pool (fallback to all games when no matches)
2. **Concat / mutate:**
   - Split titles on spaces, `:`, `-`
   - Combine 2 fragments from different titles
   - Light mutations: subtitle patterns (`: Reborn`, `II`, `Remastered`)
3. **Reject rules:** too short/long, duplicate of source, all-caps noise, empty after normalize

### Phase 2 — Markov (`forge/title`)

- Character bigram/trigram models per pool (all games, or per-genre)
- Filter against exact existing titles in corpus

### Phase 3 — Weighting polish

- Soft weighting: 80% matching genre, 20% global pool
- `alternative_names` as mutation tokens in `title`, not direct output

### Phase 4 — Identity (`forge/identity`)

- Character names from `characters` pool
- Company labels from `companies`
- Genre weighting via `characters.games[]` → `games.genres[]`

## CLI Integration

Shared logic lives in [`src/cli/generate.go`](../src/cli/generate.go). Two entry points:

```bash
# Main binary — generate subcommand skips IGDB config
go run . generate title -seed=42 -count=5

# Forge binary — no IGDB credentials required (recommended for dev)
go run ./cmd/forge title -seed=42 -count=5
go run ./cmd/forge title -data-dir=data -strategy=concat
```

Default corpus: `testdata/corpus` when present, else `data/` (override with `-data-dir` or `VARGAMES_DATA_DIR`).

Defer HTTP API until CLI proves the API (see `HTTP-API.md`).

## Testing Strategy

| Test | What it verifies |
|------|------------------|
| `testdata/corpus/` | Committed IGDB-shaped fixtures |
| `forge.LoadFromDir` | Parses all entities, handles missing optional files |
| `title` / `identity` | Determinism, weighting, normalize per subpackage |
| `forge.TestCorpusDir(t)` | Shared fixture path helper |

No live IGDB calls in tests.

## Milestones

| Milestone | Deliverable | Package | Effort |
|-----------|-------------|---------|--------|
| M1 | `corpus.go` + `LoadFromDir` + subpackage shells | `forge`, `title`, `identity` | ~1 day |
| M2 | `GameTitle` with reservoir + concat | `forge/title` | ~1 day |
| M3 | Genre filtering | `forge/title` | ~0.5 day |
| M4 | `CharacterName` (+ company later) | `forge/identity` | ~0.5 day |
| M5 | Markov strategy | `forge/title` | ~1–2 days |
| M6 | CLI (`generate` + `cmd/forge`) | `main`, `cmd/forge`, `src/cli` | ~0.5 day | **partial** (title only) |

**Total estimate:** ~4–6 days for v1 (M1–M4 + CLI).

## Open Decisions

1. **Minimum corpus size** — `games.json` required for `title`; `characters.json` sufficient for `identity` alone?
2. **Character genre weighting** — join through `games` in `identity` (Phase 4) or skip for v1?
3. **Output uniqueness** — guarantee "not in corpus" or allow low-probability collisions?

**Recommendation:** Require `games.json` for title generation; allow character-only corpus for identity. Defer character↔genre joins to M4+.

## Dependencies

- Requires fetched data in `data/` (produced by `-fetch` CLI) or `testdata/corpus`
- HTTP API (`HTTP-API.md`) depends on `forge/title` M2+

## Related Plans

- [FETCH-ROBUSTNESS.md](./FETCH-ROBUSTNESS.md) — faster/more reliable corpus refresh
- [SAMPLE-CORPUS.md](./SAMPLE-CORPUS.md) — offline fixtures
- [HTTP-API.md](./HTTP-API.md) — expose generation over HTTP
- [NAME-QUALITY.md](./NAME-QUALITY.md) — post-generation filters (may live in `forge/quality` later)

## Supersedes

Renamed from `FORGE-PACKAGE.md` — the `names` package name was too narrow for collections, companies, and artifact-specific extras.
