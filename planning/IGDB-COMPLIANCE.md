# Plan: IGDB Compliance & Attribution

## Goal

Document **legal, ToS, and ethical requirements** for using IGDB/Twitch data and generated names — especially before public HTTP deployment or GitHub Pages publication of fetch reports.

## Current Foundation

- Fetches IGDB via Twitch OAuth ([`src/igdb/client.go`](../src/igdb/client.go))
- Stores raw JSON locally in gitignored `data/`
- Generates derivative names via `forge` (not verbatim IGDB titles in normal operation)
- No attribution text in CLI output, reports, or API responses today
- Rate limiting: client backoff on 429 ([`client.go`](../src/igdb/client.go)); default `MaxConcurrent=4`

## IGDB / Twitch Requirements (summary)

Consult official docs before deploy — this plan is project guidance, not legal advice:

- [IGDB API documentation](https://api-docs.igdb.com/)
- [Twitch Developer Agreement](https://www.twitch.tv/p/legal/developer-agreement/)

Typical expectations:

| Area | Guidance |
|------|----------|
| **Credentials** | Never commit `IGDB_CLIENT_ID` / `IGDB_CLIENT_SECRET`; use env/secrets |
| **Attribution** | Credit IGDB/Twitch when displaying IGDB-sourced data publicly |
| **Rate limits** | ~4 requests/second; use backoff; avoid aggressive CI fetch schedules |
| **Data redistribution** | Do not publish full `games.json` dumps; reports/summaries OK |
| **Commercial use** | Review ToS if monetizing generated content or API access |

## Project Policies

### Fetch pipeline

- `data/` remains **gitignored** — no full corpus in public repos
- GitHub Pages publishes **HTML reports only**, not raw JSON ([CI-INFRA.md](./CI-INFRA.md))
- Scheduled CI fetch: manual or weekly; start with small entities (`genres,platforms`)
- Log rate-limit events; do not echo credentials in CI logs

### Generation output

- `RejectTitle` blocks exact corpus matches (Q0)
- Generated names are **derivative** — still avoid passing off as official IGDB data
- API responses should not include raw IGDB records unless explicitly requested via future `/meta`

### Attribution text (recommended)

Add to fetch HTML reports footer and public API `/meta`:

```
Data sourced from IGDB via Twitch API. IGDB is owned by Twitch/Amazon.
Generated names are procedural derivatives, not official IGDB entries.
```

## Rate Limiting Strategy

| Layer | Today | Planned |
|-------|-------|---------|
| Client backoff | Exponential on 429/5xx | Keep |
| Concurrency | Default 4 workers | Configurable |
| Adaptive throttle | — | Reduce concurrency when 429 rate high (P2) |
| HTTP API | — | Token bucket per IP when public |

Env: `IGDB_MAX_CONCURRENT` (existing). Future: `IGDB_ADAPTIVE_RATE=1`.

## Public Deployment Checklist

Before exposing HTTP API or Pages:

- [ ] Attribution in API `/meta` and report footer
- [ ] No IGDB credentials in client-visible responses
- [ ] Rate limiting on public API
- [ ] CORS restricted to known origins
- [ ] README section linking to IGDB/Twitch terms
- [ ] NAME-QUALITY filters enabled (profanity, blocklist)

## Milestones

| Milestone | Deliverable | Effort | Status |
|-----------|-------------|--------|--------|
| C1 | This compliance doc | — | **Done** (documentation) |
| C2 | Attribution footer in `report.go` HTML | ~0.25 day | Pending |
| C3 | `/meta` attribution field in HTTP API | ~0.1 day | Pending |
| C4 | README compliance section | ~0.1 day | Pending |
| C5 | Adaptive rate limiting research spike | ~0.5 day | Pending |

## Related Plans

- [CI-INFRA.md](./CI-INFRA.md) — fetch workflow secrets, Pages scope
- [HTTP-API.md](./HTTP-API.md) — public deploy checklist
- [FETCH-ROBUSTNESS.md](./FETCH-ROBUSTNESS.md) — concurrency and backoff
- [NAME-QUALITY.md](./NAME-QUALITY.md) — blocklist before public output
