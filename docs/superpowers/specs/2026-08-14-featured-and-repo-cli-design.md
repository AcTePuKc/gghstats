# Design: repo pins CLI + Featured showcase (v1.1.0)

**Date:** 2026-08-14  
**Status:** Draft — review before weekend implementation  
**Band:** [docs/plan-v1.1.0.md](../../plan-v1.1.0.md)  
**Implementation:** [docs/superpowers/plans/2026-08-14-featured-and-repo-cli.md](../plans/2026-08-14-featured-and-repo-cli.md)

## Goal

Let operators **pin extra repos** and **curate a Featured showcase** in SQLite via CLI, without stuffing long lists into `GGHSTATS_FILTER` and without mixing other people’s excellent forks into the traffic dashboard.

Existing installs (~1.0.x) that never run the new commands stay **behavior-identical**.

## Product intent

gghstats remains a **traffic dashboard** (clones/views beyond GitHub’s 14-day window). Featured is **editorial**: “these other repos are excellent by my criteria.” It is not a second metrics table and not repo groups on `/`.

## Operator stance (why CLI, not a settings UI)

The catalog commands are not only “`.env` is painful.” They are how the operator **lives with** gghstats.

SSH into the VPS. `gghstats repo add`. `featured rm`. `backup`. `restore`. Sync. Upgrade the binary because it is **theirs**. Stay on the console. Nostalgic on purpose: `rc.d` / systemd / a box you still log into — not a SPA admin panel.

Already in that family: `serve`, `fetch`, `report`, `export`, `backup`, `restore`, `alert test`. **E1/E2 extend that loop**, they do not open a second management surface in the browser.

**Implication:** do not add dashboard CRUD for pins/featured in 1.1.0 (or as a “quick win” later) without an explicit product decision. The HTML dashboard **shows**; the CLI **stewards**.

## Versioning (SemVer, current line **1.0.1**)

`MAJOR.MINOR.PATCH`. Patch bumps the **third** digit of the line you are on.

| Form | From 1.0.1 | Meaning | This work |
|------|------------|---------|-----------|
| Patch | `1.0.2`, `1.0.3`, `1.0.4`, … (`1.0.x`) | Fixes only, no features | **Not** this feature |
| Minor | `1.1.0` | Add/remove **without** breaking the 1.x contract | **This release** |
| Major | `2.0.0` | Serious / ROADMAP-expected moment | **Line B** (webhooks), not Featured |

After `1.1.0` is tagged, SemVer patches of *that* minor are `1.1.1`, `1.1.2` (`1.1.x`). Until then, patches stay on **`1.0.x`**. Further features after 1.1.0 wait for `1.2.0`.

## Compatibility contract (installed base)

| Operator | After upgrading to 1.1.0 |
|----------|--------------------------|
| Never pins, never features | Same FILTER, same `/`, same nav (no Featured link), same KPIs |
| Long explicit FILTER in `.env` | FILTER still works; `repo add` is **optional** |
| Curator (many forks, few originals) | `featured add` × N; `/` stays originals; `/featured` is the vitrine |

**Hard rule:** empty `featured` table → **no** Featured nav item. Do not show an empty page in the default chrome.

`GGHSTATS_FILTER` is **not** removed, ignored, or redefined. Wildcards, `!fork`, `!archived`, and exclusions keep current semantics.

## Decisions

| ID | Choice |
|----|--------|
| D1 | Two CLI families: `gghstats repo` (dashboard pins) and `gghstats featured` (vitrine) |
| D2 | FILTER ∪ pins = traffic sync + `/` table. Featured-only repos **not** on `/` |
| D3 | Empty CLI catalog → FILTER-only discovery (today). Pins/featured **add**, they do not replace FILTER |
| D4 | Nav: Repositories → H2H → **Featured** (i18n Destacados). Route `/featured` |
| D5 | Card: title/link = **upstream**; small line = operator fork when `fork=true` |
| D6 | Featured sync = metadata only (`GET /repos/{name}` + parent). **No** clones/views/paths |
| D7 | No Featured JSON API in 1.1.0 (HTML server-rendered). API-only mode has no vitrine until **1.2.0** |
| D8 | Slices: **E1** pins+union first; **E2** featured page. Both in 1.1.0 if time; E1 alone is a valid ship cut |
| D9 | No GSD `.planning/` reboot. Band plan + this spec + ROADMAP Line **E** |
| D10 | Stewardship stays on the CLI (VPS/console). Dashboard does not grow pin/featured editors |

## Surfaces

### Dashboard `/`

Unchanged layout. Row set = FILTER matches ∪ `pins`. KPIs and the clones-over-time chart follow that set only.

### `/featured`

Separate HTML page, same neo-brutalist chrome. Not a clone table. Not KPI cards for 25k clones.

Each card:

- Title + link → `https://github.com/{upstream_full_name}`
- If fork: muted line “Fork of” / “I forked” → `https://github.com/{name}` (operator copy)
- If not a fork: no fork line; title is `{name}`
- Upstream description
- Upstream star count (criterion of “excellent”), not fork clone counts

Before first metadata sync: show `{name}` linking to GitHub; omit stars/parent until `meta_updated_at` is set.

### Nav

Third link, below H2H, above language. Hidden when `COUNT(featured)=0`. Hidden in `GGHSTATS_API_ONLY`. Direct `GET /featured` with empty table: 200 + short empty state **or** 404 — **pick 200 empty state without nav link** so a bookmarked URL after `featured rm` last item does not 404. Nav still hidden.

## Data

Migration **v6** (`PRAGMA user_version`).

```sql
CREATE TABLE IF NOT EXISTS pins (
  name       TEXT PRIMARY KEY,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS featured (
  name                  TEXT PRIMARY KEY,
  created_at            TEXT NOT NULL,
  sort                  INTEGER NOT NULL DEFAULT 0,
  parent_full_name      TEXT NOT NULL DEFAULT '',
  upstream_full_name    TEXT NOT NULL DEFAULT '',
  upstream_description  TEXT NOT NULL DEFAULT '',
  upstream_stars        INTEGER NOT NULL DEFAULT 0,
  fork                  INTEGER NOT NULL DEFAULT 0,
  meta_updated_at       TEXT NOT NULL DEFAULT ''
);
```

- `pins.name` / `featured.name` = operator repo (`hrodrig/awesome-readme`).
- `upstream_full_name` = parent when fork, else `name`.
- `sort` = insertion order (`MAX(sort)+1` on add). `ls` and page use `ORDER BY sort, name`.
- Existing `repos` traffic rows **unchanged**. Featured-only names may exist in `featured` without a `repos` traffic row.

### Name rules

Normalize: trim space. Require exactly one `/` with non-empty owner and repo (`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$` or the project’s existing GitHub name rules if a helper already exists). Reject `*` and FILTER syntax. Do not case-fold; GitHub metadata refresh may rewrite to canonical `FullName`.

## Sync

### Tracked (traffic)

```
tracked = applyFilter(ListRepos, GGHSTATS_FILTER) ∪ pins
```

Pins not in the filtered list are appended as `github.Repo{FullName: pin}` (same as today’s explicit `opts.Repos` path). Then existing `syncRepo` (views, clones, referrers, paths, optional stars).

`opts.Repos` non-empty (fetch/single-repo) **unchanged**: no pin union (explicit list still wins).

### Featured (metadata)

After tracked workers (or a dedicated pass that must not fail the traffic cycle):

For each `featured.name`: `Client.Repo(name)`; if `Parent` set, `Client.Repo(parent)` for stars + description; `UpsertFeaturedMeta`. Per-row failure: log warn, keep previous meta, do not fail `Run`.

Quota: two GETs per featured fork, one GET per non-fork. Thirty featured ≈ 60 extra REST calls, not 30× traffic endpoints.

## CLI

Same DB as `serve` (`--db` / `GGHSTATS_DB` / platform default). **No GitHub token** required for add/rm/ls.

```
gghstats repo add OWNER/REPO
gghstats repo rm  OWNER/REPO
gghstats repo ls

gghstats featured add OWNER/REPO
gghstats featured rm  OWNER/REPO
gghstats featured ls
```

| Case | Behavior |
|------|----------|
| `add` duplicate | Idempotent; stderr note; exit 0 |
| `rm` missing | Error; exit 1 |
| Invalid name | Error; exit 1 |
| `--help` | Subcommand usage; exit 0 |
| `serve` holding SQLite write | CLI waits (`busy_timeout` already 5s); document “retry / avoid during sync” |
| `featured add` of a FILTER original | Allowed: appears on `/` **and** `/featured` |
| `repo add` of a fork while FILTER has `!fork` | Pin **overrides** exclusion for that name (union). That is the point of pins. |

Pins on `/` **do** get traffic. If the operator wanted vitrine-only, they use `featured add`, not `repo add`.

## Errors and ops

- English CLI messages.
- Man page + `gghstats --help` list `repo` and `featured`.
- README: FILTER still discovery; pins/featured in DB; empty featured = no nav.
- `contrib/gghstats.env.example`: comment that FILTER is not the place for 30 one-off names or a showcase list.

## Out of scope (1.1.0)

- Repo groups / tabs on `/`
- UI to add/remove pins or featured (D10: dashboard shows; CLI stewards)
- GitHub topics as group source
- Syncing traffic for all account forks
- Featured JSON `/api/v1/featured` (→ 1.2.0 if needed)
- Replacing FILTER
- Unique cloners as a first-class index/repo metric (already stored as `clone_uniques`; → 1.2.0)
- VERSION bump in the same commit as features (dedicated bump commit before release, per `AGENTS.md`)

## Testing (normative)

Feature tests (CRUD, union, `/featured`) are not enough. **Regression is the 1.0.x contract.** ~4k installs that never run `repo`/`featured` must not see a behavior change. If any **R** test is missing or red, the slice does not ship.

### Existing suite that must stay green

Run after **every** E1 and E2 task, not only at the end:

```
make test
go test ./internal/sync/ ./internal/server/ ./internal/i18n/ ./cmd/gghstats/ -count=1
```

Must still pass (names as of 1.0.1 — do not delete or weaken):

| Existing test | Guards |
|---------------|--------|
| `TestFilterExcludeFork` | `!fork` still drops forks when no pin |
| `TestAPIOnlySkipsHTMLAndSEO` | API-only has no dashboard HTML |
| `TestDogfoodContract_APIOnly` | `/api/repos`, repo, H2H JSON unchanged for dogfood |
| i18n missing-key tests | all locales have every EN key |
| `TestVersionedMigrations` | bump expect **6**; old DBs migrate |
| `cmd/gghstats` CLI dispatch tests | `fetch` / `backup` / `serve` still exist |

Coverage floor **≥80%** (`make cover`) still applies before tag (`SPEC` §6.1).

### New regression IDs (must exist as tests)

**Empty catalog (the 4k path)** — no pins, no featured:

| ID | Assert |
|----|--------|
| **R1** | `GET /` HTML contains Repositories + H2H; **no** `href="/featured"` |
| **R2** | `FILTER=*,!fork` discovery set **equals** pre-1.1.0 (forks absent). Pin union **not** applied when `pins` empty |
| **R3** | `Run` with empty featured does **not** call `GET /repos/{name}` extra times for a vitrine pass that no-ops; traffic set unchanged |
| **R4** | Index KPIs / `ListRepos` row count unchanged by empty `featured` table |
| **R5** | `GGHSTATS_API_ONLY=true`: `GET /featured` is not an HTML dashboard page (404 or same as `/h2h` under API-only) |
| **R6** | `GET /h2h` still 200 with HTML UI |

**With catalog (must not leak into `/`):**

| ID | Assert |
|----|--------|
| **R7** | `featured add` fork + `!fork` → that name **absent** from tracked/traffic/`ListRepos` KPIs |
| **R8** | `repo add` of the same fork → **present** in tracked (union). R7 vs R8 proves the two families |
| **R9** | After `featured add`, `GET /` **has** `href="/featured"`; after `featured rm` last row, nav **gone** (R1 again) |
| **R10** | `Options.Repos` explicit list: pins in DB are **not** merged (fetch one repo still one repo) |
| **R11** | Featured meta path never hits `/traffic/views` or `/traffic/clones` for featured-only names |
| **R12** | Opening a 1.0.x SQLite file (user_version 5) migrates to 6; empty `pins`/`featured`; R1–R4 still hold |

### Gate

- E1 done only if `make test` + **R2, R8, R10, R12** green.
- E2 done only if `make test` + **R1, R3–R7, R9, R11** green.
- Tag 1.1.0 only if **all R** + `make release-check` (user asks).

## Example

```
GGHSTATS_FILTER="hrodrig/*,!fork"
gghstats featured add hrodrig/awesome-readme
```

`/` → ~26 originals, ~25k clones. `/featured` → card linking to `matiassingers/awesome-readme`, fork line to `hrodrig/awesome-readme`.
