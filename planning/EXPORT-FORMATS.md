# Plan: Export Formats — Generated Name Output

## Goal

Let users export batches of generated names to **CSV, JSONL, or plain text** files — useful for game jams, content pipelines, and spreadsheet review.

## Current Foundation

- CLI prints names to stdout (`generate title -count=10`)
- No `--output` flag; no structured export
- HTTP API returns JSON arrays (planned) — no file download endpoint

## Proposed Formats

| Format | Extension | Use case |
|--------|-----------|----------|
| Plain text | `.txt` | One name per line; simplest |
| CSV | `.csv` | Spreadsheets; columns: `name,strategy,seed,genre,platform` |
| JSONL | `.jsonl` | Pipelines; one JSON object per line |

### CSV example

```csv
name,strategy,seed,genre_id,type
"Dark Souls: Reborn",concat,42,12,game
"Linkara",pick,42,,character
```

### JSONL example

```jsonl
{"name":"Dark Souls: Reborn","strategy":"concat","seed":"42","genre_id":12,"type":"game"}
{"name":"Linkara","strategy":"pick","seed":"42","type":"character"}
```

## CLI

```bash
vargames-name-gen generate title -count=1000 -seed=jam1 -output=names.csv
vargames-name-gen generate title -count=100 -output=names.jsonl -format=jsonl
forge title -count=50 -output=names.txt
```

| Flag | Default | Description |
|------|---------|-------------|
| `-output` | stdout | File path (extension implies format if `-format` omitted) |
| `-format` | from extension | `txt`, `csv`, `jsonl` |

## HTTP API (optional v1.1)

```
GET /generate/game-name?count=100&format=csv
Content-Disposition: attachment; filename="names.csv"
```

Defer file download until CLI export proves useful.

## Implementation

```
src/cli/
  export.go       # WriteNames(format, path, []GeneratedName)
  export_test.go
```

Shared `GeneratedName` struct:

```go
type GeneratedName struct {
    Name     string
    Strategy string
    Seed     string
    GenreID  *int
    Type     string // "game" | "character"
}
```

## Milestones

| Milestone | Deliverable | Effort |
|-----------|-------------|--------|
| E1 | `-output` txt (one name per line) | ~0.25 day |
| E2 | CSV with metadata columns | ~0.25 day |
| E3 | JSONL export | ~0.25 day |
| E4 | HTTP `format=csv` query param | ~0.25 day |

**Total estimate:** ~1 day.

**Depends on:** CLI-STRUCTURE `generate` subcommand (**done**).

**Status:** Planning doc complete; no `-output` implementation yet.

## Related Plans

- [CLI-STRUCTURE.md](./CLI-STRUCTURE.md) — `generate` flags
- [HTTP-API.md](./HTTP-API.md) — optional download format
- [NAME-QUALITY.md](./NAME-QUALITY.md) — only export filtered names
