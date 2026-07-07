# Plan: CLI Structure — Subcommands

## Goal

Replace the growing flat-flag interface in [`main.go`](../main.go) with **discoverable subcommands** as fetch, generate, serve, and reporting features accumulate.

## Current Foundation

| Flag / behavior | Today |
|-----------------|-------|
| (none) | Validate config; print hello message |
| `-fetch=entities` | Fetch IGDB entities to `data/` |
| `-verbose` | Enable logging |

Planned additions from other plans:

- `-generate` ([FORGE-PACKAGE.md](./FORGE-PACKAGE.md))
- `-serve` ([HTTP-API.md](./HTTP-API.md))
- `-report-index` ([CI-INFRA.md](./CI-INFRA.md))
- `fetch --resume` ([FETCH-RESUME.md](./FETCH-RESUME.md))
- `index` ([EMBEDDED-INDEX.md](./EMBEDDED-INDEX.md))

Flat flags become unwieldy and hard to document.

## Proposed Architecture

```mermaid
flowchart TB
  Main[main.go] --> Dispatch[subcommand dispatch]
  Dispatch --> Validate[validate / default]
  Dispatch --> Fetch[fetch]
  Dispatch --> Generate[generate]
  Dispatch --> Serve[serve]
  Dispatch --> ReportIndex[report-index]
  Dispatch --> Index[index - future]
```

## Target UX

```bash
# Default: validate config (unchanged behavior)
vargames-name-gen
vargames-name-gen validate

# Fetch
vargames-name-gen fetch games,genres,platforms
vargames-name-gen fetch games --profile=minimal --concurrent=4 --partial

# Generate (after FORGE-PACKAGE)
vargames-name-gen generate game --genre=12 --seed=42 --count=5
vargames-name-gen generate character --count=3

# Serve HTTP API (after HTTP-API)
vargames-name-gen serve --addr=:8080 --data-dir=data

# Reports
vargames-name-gen report-index --data-dir=data

# Future
vargames-name-gen index --data-dir=data    # build SQLite corpus
vargames-name-gen fetch games --resume
```

## Implementation Approach

### Phase 1: stdlib dispatch (recommended v1)

- Parse `os.Args[1]` as subcommand name
- Each subcommand gets its own `flag.FlagSet`
- Shared flags: `-verbose`, `-data-dir` via helper or parent config struct
- **No new dependency**

### Phase 2: cobra (if subcommands exceed ~6)

- Migrate when `validate`, `fetch`, `generate`, `serve`, `report-index`, `index`, `resume` all exist
- Better help text and shell completion
- Document migration in README

## Package Layout Options

**Option A (minimal):** Keep all logic in `main.go` + `cli.go` at repo root

**Option B (cleaner):** `src/cli/` with one file per subcommand

**Recommendation:** Option B once `fetch` moves out of `main.go`; start with `main.go` + `cli_dispatch.go`.

```
src/cli/
  cli.go            # Run(args []string) error
  fetch.go
  generate.go       # later
  serve.go          # later
  flags.go          # shared Verbose, DataDir
```

`main.go` becomes: load config, `cli.Run(os.Args[1:])`.

## Shared Configuration

| Flag | Env | Used by |
|------|-----|---------|
| `-verbose` | `IGDB_VERBOSE`, `VARGAMES_VERBOSE` | all |
| `-data-dir` | `VARGAMES_DATA_DIR` | generate, serve, report-index, index |

IGDB credentials only required for `fetch` subcommand.

## Backward Compatibility

| Old | New | Transition |
|-----|-----|------------|
| `-fetch=games` | `fetch games` | Accept old flag with deprecation warning for 1 release |
| `-verbose` | global `-verbose` on any subcommand | Keep working |

Print warning: `"-fetch is deprecated; use: vargames-name-gen fetch games"`.

## Help Text

```
vargames-name-gen - IGDB corpus tools and name generation

Usage:
  vargames-name-gen [command]

Commands:
  validate      Check configuration and exit
  fetch         Fetch IGDB entities to data/
  generate      Generate game or character names
  serve         Start HTTP API server
  report-index  Build HTML index of fetch reports

Flags:
  -verbose      Enable verbose logging
```

## Milestones

| Milestone | Deliverable | Effort |
|-----------|-------------|--------|
| C1 | Dispatch skeleton + `validate` default | ~0.5 day |
| C2 | `fetch` subcommand (migrate from `-fetch`) | ~0.5 day |
| C3 | Deprecation shim for `-fetch=` | ~0.25 day |
| C4 | `generate` subcommand | ~0.25 day (with FORGE-PACKAGE M6) |
| C5 | `serve` subcommand | ~0.25 day (with HTTP-API H4) |
| C6 | `report-index` subcommand | ~0.25 day |
| C7 | `src/cli/` package extraction | ~0.5 day |

**Total estimate:** ~2–2.5 days across multiple feature landings.

## Open Decisions

1. **Binary name** — keep `vargames-name-gen` or shorten to `vng` via symlink?
2. **cobra now vs later** — stdlib first per YAGNI?
3. **Exit codes** — document per subcommand (fetch partial = 1, etc.)?

**Recommendation:** stdlib dispatch; document exit codes in README.

## Risk Register

| Risk | Mitigation |
|------|------------|
| Breaking existing scripts using `-fetch` | Deprecation period + README migration |
| Duplicated flag parsing | `src/cli/flags.go` shared helper |
| `main_test.go` breakage | Update tests when moving fetch logic |

## Related Plans

- [FORGE-PACKAGE.md](./FORGE-PACKAGE.md) — `generate` subcommand (M6)
- [HTTP-API.md](./HTTP-API.md) — `serve` subcommand (H4)
- [FETCH-ROBUSTNESS.md](./FETCH-ROBUSTNESS.md) — fetch flags move under `fetch`
- [FETCH-RESUME.md](./FETCH-RESUME.md) — `fetch --resume`
- [EMBEDDED-INDEX.md](./EMBEDDED-INDEX.md) — `index` subcommand
- [CI-INFRA.md](./CI-INFRA.md) — `report-index`
