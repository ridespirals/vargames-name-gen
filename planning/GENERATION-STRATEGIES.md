# Plan: Generation Strategies — Registry & Plugins

## Goal

Define a **pluggable strategy interface** for name generation so built-in strategies (`pick`, `concat`, `markov`) and future third-party strategies share one contract.

## Current Foundation

- `pick` and `concat` implemented inline in [`src/forge/title`](../src/forge/title/) and [`src/forge/identity`](../src/forge/identity/)
- Strategy selected via `-strategy` CLI flag and `Options.Strategy` string
- No registry; adding a strategy requires editing generator code
- Markov planned in FORGE M5; plugin registry mentioned in planning README out-of-scope list

## Strategy Interface

```go
package title // identity mirrors

type GenerateContext struct {
    Corpus   *forge.Corpus
    Pool     []forge.Game   // pre-filtered by genre/platform
    Rand     *rand.Rand
    Options  Options
}

type Strategy interface {
    Name() string
    Generate(ctx GenerateContext) (string, error)
}
```

## Built-in Registry (v1)

```go
var defaultStrategies = map[string]Strategy{
    "pick":   PickStrategy{},
    "concat": ConcatStrategy{},
    // "markov": MarkovStrategy{},  // M5
}

func StrategyFor(name string) (Strategy, error)
func ListStrategies() []string
```

CLI and HTTP `/strategies` endpoint call `ListStrategies()`.

## Plugin Extension (v2 — optional)

If external strategies are needed:

1. **Compile-time registration** — `init()` calls `RegisterStrategy("custom", CustomStrategy{})`
2. **Go plugin** — `.so` loading (platform limitations; defer)
3. **WASM** — sandboxed strategies (long-term; INTERACTIVE-UI)

**Recommendation:** Compile-time registration only for v1–v2.

## Integration Points

| Consumer | Usage |
|----------|-------|
| `title.Generator.GameTitle` | `StrategyFor(opts.Strategy)` → `Generate` → `RejectTitle` → quality chain |
| `identity.Generator.CharacterName` | Same pattern |
| CLI `-strategy` | Validated against `ListStrategies()` |
| HTTP `?strategy=` | 400 if unknown |
| OPENAPI | Enum from `ListStrategies()` |

## Markov Strategy (M5)

```
src/forge/title/markov.go
```

- Build bigram/trigram models per pool at generator init (or lazy)
- Reject output matching exact corpus titles (Q0)
- Additional Markov-specific quality rules (NAME-QUALITY)

## Milestones

| Milestone | Deliverable | Effort | Status |
|-----------|-------------|--------|--------|
| S1 | `Strategy` interface + registry in `title` | ~0.5 day | Pending |
| S2 | Refactor `pick`/`concat` to registry | ~0.5 day | Pending |
| S3 | Mirror registry in `identity` | ~0.25 day | Pending |
| S4 | `ListStrategies()` for CLI + HTTP | ~0.1 day | Pending |
| S5 | Markov strategy | ~1–2 days | Pending (FORGE M5) |
| S6 | Compile-time `RegisterStrategy` for extensions | ~0.25 day | Pending |

**Today:** `pick` and `concat` are implemented inline in `title/concat.go` and `identity/concat.go` — no registry yet.

## Open Decisions

1. **Per-type strategy lists** — game strategies may differ from character strategies?
2. **Strategy options** — JSON config per strategy (e.g. Markov order)?

**Recommendation:** Separate lists per generator family; strategy-specific options as nested struct on `Options`.

## Related Plans

- [FORGE-PACKAGE.md](./FORGE-PACKAGE.md) — M5 Markov, M9 registry
- [HTTP-API.md](./HTTP-API.md) — `/strategies` endpoint
- [CLI-STRUCTURE.md](./CLI-STRUCTURE.md) — `-strategy` validation
- [NAME-QUALITY.md](./NAME-QUALITY.md) — post-strategy filters
