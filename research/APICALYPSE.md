# Apicalypse Cheat Sheet (IGDB)

**Apicalypse** is the query language IGDB uses in **POST request bodies**. It is not JSON — it is plain text, semicolon-terminated statements.

Official references:

- [IGDB API docs](https://api-docs.igdb.com/)
- [Apicalypse syntax](https://apicalypse.io/syntax/)

This project sends queries via [`src/igdb/client.go`](../src/igdb/client.go) (`Content-Type: text/plain`) and builds paging queries in [`src/igdb/fetcher.go`](../src/igdb/fetcher.go). Example Bruno requests live under [`bruno/`](../bruno/).

---

## HTTP basics (IGDB)

| Item | Value |
|------|-------|
| Method | `POST` for data queries (e.g. `/v4/games`) |
| Body | Apicalypse text (not JSON) |
| `Content-Type` | `text/plain` |
| Headers | `Client-Id`, `Authorization: Bearer <token>` |
| Rate limit | ~4 requests/second (429 when exceeded) |
| `limit` max | **500** per request (this repo defaults to 500 via `IGDB_MAX_LIMIT`) |
| Count | `GET /v4/<entity>/count` (no body) — see [bruno/counts/](../bruno/counts/) |

---

## Statement cheat sheet

Statements are separated by `;`. Order is usually flexible; IGDB examples often put `fields` first.

| Statement | Purpose | Example |
|-----------|---------|---------|
| `fields` | Columns / expansions to return | `fields id,name,genres;` |
| `exclude` | Omit fields when using `fields *` | `exclude screenshots;` |
| `where` | Filter (SQL-like) | `where rating > 80;` |
| `search` | Full-text search on the endpoint | `search "zelda";` |
| `sort` | Order results | `sort rating desc;` |
| `limit` | Page size (1–500) | `limit 50;` |
| `offset` | Skip N results (pagination) | `offset 1000;` |

### Shorthand (supported by Apicalypse)

| Long | Short |
|------|-------|
| `fields` | `f` |
| `where` | `w` |
| `limit` | `l` |

Example: `f name,rating; w rating > 80; l 10;`

---

## `fields` — selection & expansion

```apicalypse
fields name;
```

```apicalypse
fields id,name,genres,platforms,checksum;
```

```apicalypse
fields *;
```

**Nested / expanded fields** use dot notation:

```apicalypse
fields name, genres.name;
```

```apicalypse
fields name, genres.*;
```

```apicalypse
fields *, release_dates.*, release_dates.platform.*;
```

Deep expansion (from this repo’s [`bruno/games.yml`](../bruno/games.yml)):

```apicalypse
fields *, release_dates.*, release_dates.platform.*, release_dates.platform.versions.*;
```

**Related record on another entity:**

```apicalypse
fields name, game.name;
```

Use `GET /v4/<entity>/meta` or `fields *` on a single row during exploration to discover field names.

---

## `exclude` — all fields except …

```apicalypse
fields *;
exclude screenshots, videos;
```

Useful when you want almost everything but a few heavy arrays.

---

## `where` — filters

### Comparison operators

| Op | Meaning |
|----|---------|
| `=` | equals |
| `!=` | not equals |
| `>`, `>=`, `<`, `<=` | numeric / date comparisons |
| `&` | AND |
| `\|` | OR |

### Examples

Single ID:

```apicalypse
fields name;
where id = 1942;
```

Multiple IDs (common for incremental re-fetch):

```apicalypse
fields *;
where id = (8, 9, 11);
```

Genre filter:

```apicalypse
fields name, genres;
where genres = 4;
```

Compound:

```apicalypse
fields name, rating;
where rating > 80 & genres = 12;
```

Release activity on a platform:

```apicalypse
fields *;
where game.platforms = 48 & date > 1538129354;
sort date asc;
```

Exclude game editions (DLC / collector’s editions often have `version_parent` set):

```apicalypse
fields name, involved_companies;
search "Assassins Creed";
where version_parent = null;
```

### Array / set operators (Apicalypse)

| Op | Meaning |
|----|---------|
| `[]` | contains **all** of these values |
| `![]` | does not contain all of these |
| `()` | contains **at least one** of these |
| `!()` | does not contain any of these |
| `{}` | contains these values **exclusively** |

---

## `search` — text search

Search applies to the **endpoint you POST to** (`/games`, `/characters`, etc.). Results are ranked by similarity.

```apicalypse
search "zelda";
fields name;
```

```apicalypse
fields *;
search "sonic the hedgehog";
limit 50;
```

```apicalypse
search "Halo";
fields name, release_date.human;
```

Combine with `where` / `sort`:

```apicalypse
fields name, rating;
search "Halo";
sort rating desc;
limit 10;
```

---

## `sort`

```apicalypse
fields name, rating;
sort rating desc;
```

```apicalypse
fields name, first_release_date;
sort first_release_date asc;
limit 20;
```

---

## Pagination — `limit` & `offset`

Default `limit` is **10** if omitted. Maximum is **500**.

Page 1:

```apicalypse
fields id, name, checksum;
limit 500;
offset 0;
```

Page 2:

```apicalypse
fields id, name, checksum;
limit 500;
offset 500;
```

Page N (0-based page index `p`, page size `L`):

```text
offset = p * L
```

From [`bruno/alternative_names.yml`](../bruno/alternative_names.yml):

```apicalypse
fields *, game.name;
offset 195856;
limit 5;
```

### How this repo pages

[`Fetcher.FetchAll`](../src/igdb/fetcher.go) builds each request as:

```text
<QueryPrefix> limit <L>; offset <page * L>;
```

Default `<QueryPrefix>` is `fields *;` (see [`entities.go`](../src/igdb/entities.go)). Planned lean profiles are in [FETCH-PROFILES.md](../planning/FETCH-PROFILES.md).

---

## Entity count (no Apicalypse body)

```http
GET https://api.igdb.com/v4/games/count
Client-Id: <id>
Authorization: Bearer <token>
```

Returns a number — used in [FETCH-ROBUSTNESS.md](../planning/FETCH-ROBUSTNESS.md) for parallel fan-out planning. Bruno examples: [`bruno/counts/`](../bruno/counts/).

---

## Example queries by task

### Minimal fetch for name generation

Planned “minimal” profile ([FETCH-PROFILES.md](../planning/FETCH-PROFILES.md)):

```apicalypse
fields id,name,genres,platforms,checksum;
limit 500;
offset 0;
```

```apicalypse
fields id,name,game,comment,checksum;
limit 500;
offset 0;
```

### Checksum scan (change detection)

Lightweight full table pass ([FETCH-ROBUSTNESS.md](../planning/FETCH-ROBUSTNESS.md) Phase D):

```apicalypse
fields id,checksum;
limit 500;
offset 0;
```

### Re-pull specific rows after checksum diff

```apicalypse
fields id,name,genres,platforms,checksum;
where id = (1020, 4025, 119133);
```

### Top-rated games sample

```apicalypse
fields name, rating, rating_count;
where rating != null;
sort rating desc;
limit 25;
```

### Games on a platform

```apicalypse
fields name, platforms;
where platforms = 48;
sort name asc;
limit 100;
```

### Character lookup

```apicalypse
fields *;
search "link";
limit 10;
```

### Small reference tables (genres, platforms)

```apicalypse
fields *;
```

Genres/platforms are small enough to fetch in one or few pages.

---

## Endpoints used in this project

| Entity | POST path | Notes |
|--------|-----------|-------|
| `games` | `/v4/games` | Largest table; always paginate |
| `characters` | `/v4/characters` | |
| `genres` | `/v4/genres` | Small |
| `platforms` | `/v4/platforms` | Small |
| `collections` | `/v4/collections` | |
| `companies` | `/v4/companies` | |
| `alternative_names` | `/v4/alternative_names` | Very large; paginate |

---

## Tips & gotchas

1. **Always terminate statements with `;`** — this repo normalizes query prefixes to end with `;` before appending `limit`/`offset`.
2. **Prefer explicit `fields`** over `fields *` for large entities — smaller payloads, faster fetches ([FETCH-PROFILES.md](../planning/FETCH-PROFILES.md)).
3. **`checksum` changes** when IGDB updates a record — pair with `id` for incremental sync.
4. **`search` is not a filter** — it ranks by relevance; use `where` for exact constraints.
5. **429 responses** — back off and retry; client implements exponential backoff ([`client.go`](../src/igdb/client.go)).
6. **IDs are integers** — genre/platform filters use numeric IDs; fetch `genres` / `platforms` first to map names → IDs.
7. **Expansions can explode payload size** — `fields *` plus nested `release_dates.platform.versions.*` is convenient in Bruno, expensive in production fetch.

---

## curl template

```bash
curl -X POST "https://api.igdb.com/v4/games" \
  -H "Client-Id: $IGDB_CLIENT_ID" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: text/plain" \
  -d 'fields name,rating; where rating > 90; sort rating desc; limit 10;'
```

Token: Twitch client-credentials flow (see [`bruno/auth.yml`](../bruno/auth.yml)) or set `IGDB_ACCESS_TOKEN` in `.env`.

---

## Related docs in this repo

- [FETCH-PROFILES.md](../planning/FETCH-PROFILES.md) — per-entity field sets
- [FETCH-ROBUSTNESS.md](../planning/FETCH-ROBUSTNESS.md) — paging, count, checksum sync
- [DATABASE.md](./DATABASE.md) — storage implications of fetch shape/size
- [AGENTS.md](../AGENTS.md) — IGDB client and fetcher design
