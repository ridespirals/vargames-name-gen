# Sample Corpus Fixtures

Committed IGDB-shaped JSON for offline development and CI. **Not** a production dataset.

## Contents

| File | Records | Notes |
|------|---------|-------|
| `games.json` | 10 | Recognizable titles (Zelda, Mario, etc.); 3+ genres, 2+ platforms |
| `characters.json` | 10 | Linked to sample games via `games[]` |
| `genres.json` | 5 | Subset of common IGDB genres |
| `platforms.json` | 3 | e.g. NES, SNES, PC |
| `alternative_names.json` | 5 | Linked to sample games; varied `comment` for locale tests |
| `collections.json` | 3 | |
| `companies.json` | 3 | |

Total size: ~2.1 KB.

## Provenance

- Hand-crafted for test stability; fields match `forge` minimal structs
- Names are well-known game titles used for clarity in tests — not a random sample from IGDB
- No secrets or credentials in fixture data

## Refresh process

When IGDB schema or `forge` minimal structs change:

1. Update structs in [`src/forge/corpus.go`](../../src/forge/corpus.go)
2. Adjust fixtures to match required fields
3. Run `go test ./src/forge/...`
4. Optional: truncate local `data/` with `scripts/truncate-corpus.sh` (planned, SAMPLE-CORPUS S7)

## Usage

```bash
go run ./cmd/forge title -seed=1
go test ./src/forge/...
```

Default data dir resolves to `testdata/corpus` when present ([`src/cli/flags.go`](../../src/cli/flags.go)).

## Do not

- Copy real `data/` dumps into this directory (gitignored production data may contain unexpected fields)
- Commit IGDB credentials or Twitch tokens

See [planning/SAMPLE-CORPUS.md](../planning/SAMPLE-CORPUS.md).
