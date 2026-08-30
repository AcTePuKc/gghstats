# Plan — v1.5.0

**Status:** **In progress** — F-fresh + V-vis landing on `develop` (keep
**plan ↔ ROADMAP ↔ SPEC ↔ CHANGELOG ↔ README/api/man** aligned as slices merge).

**Band goal:** Make GitHub traffic reporting honest about freshness and gaps;
separate **collection** from **reporting** with fail-closed defaults; give
operators machine-readable report inventory and a clear chart legend
(null vs confirmed zero); let them download chart-aligned clones/views JSON
from a repo detail page for external visualization tools.

Parent: [ROADMAP.md](../ROADMAP.md) · Spec: [SPEC.md](../SPEC.md)

## Versioning

**1.4.0 → 1.5.0** (minor). 1.4.0 closed UI/SEO + Carlok (#22–#25).

## In scope

| ID | Slice | Item |
|----|-------|------|
| **F-fresh** | Traffic freshness | Persist per-metric (`views` / `clones`) fetch success, latest observed UTC day, and coverage bounds from the GitHub payload; statuses `fresh \| delayed \| missing \| failed \| never`; confirmed zero vs omitted day; detail charts use `null` gaps (never infer zero); independent views/clones fetch so one failure does not discard the other |
| **V-vis** | Report visibility | Persist `github_visibility` + `report_policy`; precedence `exclude > include > inherit`; `GGHSTATS_REPORT_PRIVATE`; CLI `gghstats repo report ls/set`; report surfaces fail closed with indistinguishable 404s; collection filters/pins/`GGHSTATS_INCLUDE_PRIVATE` stay separate from reporting |
| **V-json** | Report CLI JSON | `gghstats repo report ls --json` (machine-readable); optional filters for `unknown` / `exclude` / visibility to script fail-closed post-upgrade checks |
| **U-legend** | Chart legend | On repo detail traffic charts, short i18n legend: gap/`null` = not reported by GitHub; `0` = confirmed zero. Cuts support ambiguity next to F-fresh / X-chart |
| **X-chart** | Repo chart JSON | On `/{owner}/{repo}`, download chart-aligned clones/views JSON (null gaps + freshness) for external viz; additive dense dogfood API; default sparse traffic JSON unchanged |
| **DOC** | Docs | SPEC / README / api / man / CHANGELOG / ROADMAP; operator migration note for fail-closed upgrade |
| **REL** | Release | `VERSION` 1.5.0, `make release-check`, develop→main, tag (explicit user OK) |

### Suggested implementation order

1. **F-fresh + V-vis** (migrations, store, sync, server enforcement, CLI, tests,
   docs that ship with those features).
2. **V-json** (extends `repo report ls`; natural follow-on to V-vis).
3. **U-legend** (small UI/i18n; can land with F-fresh or right after).
4. **X-chart** (depends on coverage/freshness helpers and report scope).
5. **DOC** closure (SPEC dual semantics, ROADMAP band row, CHANGELOG).
6. **REL** when the band is complete.

Work lands on `develop` via pull requests (repo gitflow).

## Design decisions (recorded)

1. **FAIL-CLOSED.** On upgrade, existing repos become `unknown` + `inherit` and
   stay hidden on all report surfaces until a metadata refresh (first
   post-upgrade sync) or an explicit `include`.
2. **Collection ≠ reporting.** `GGHSTATS_INCLUDE_PRIVATE` (collect) and
   `GGHSTATS_REPORT_PRIVATE` (report) are distinct; both default false.
3. **`include` is not a collection bypass.** Reporting-only for already-collected
   repos. Never-collected repos need **`repo add OWNER/REPO`** (pin):
   collection is `GGHSTATS_FILTER ∪ pins`. `featured add` is editorial and
   respects visibility — it does not force it.
4. **Operator alerts see everything.** Per-target rule evaluation may use
   include-private internally; privacy applies to public report surfaces, not
   to the operator’s own alerts.
5. **KNOWN LIMITATION (accepted).** The aggregate index chart does not expose a
   per-repo coverage matrix (Parking lot).
6. **Charts ≠ alerts on missing days.** Detail charts / dense export use `null`
   gaps; H2H / momentum / rolling / `clones_7d|30d` keep **missing-day-as-zero**.
   SPEC §8 must state both.
7. **Coverage window.** Bounds come from dates **present in the GitHub
   payload**, not a fixed `now-13..now` rectangle. Inside that span, absent
   days → `missing`; if latest observed &lt; yesterday UTC → `delayed`; the
   current UTC day is never missing.
8. **Sparse traffic API stays default.** Dense null-gap series are additive
   (X-chart query/route + download). Do not break sparse API consumers.

## X-chart — detail

**Need:** From a repo detail page (e.g. `/hrodrig/pgwd`), export clones and
views chart data as JSON for Grafana, Observable, notebooks, BI.

**Already exists (do not reinvent):**

| Surface | Role | Gap |
|---------|------|-----|
| `GET /api/v1/repos/{o}/{r}/traffic` | Sparse JSON + auth | No UI download; sparse ≠ chart density |
| `gghstats export` CLI | CSV one repo | Not browser; not chart-null JSON |
| `GET /export.jsonl` | Index catalog | Not per-repo traffic charts |

**Shape:**

- UI: download control on the repo page (same button language as index JSONL).
- Dogfood: additive dense route (e.g. `?dense=1` or `/traffic/charts`) matching
  detail chart semantics + freshness metadata.
- Auth / visibility: report-scoped (hidden repos → indistinguishable 404);
  prefer reusing API-token / sessionStorage patterns already used for Sync.
- Filename: `gghstats-{owner}-{repo}-traffic-YYYYMMDD.json` (UTC).

## Frictions to close in SPEC/docs

| ID | Topic | Disposition |
|----|-------|-------------|
| **A** | SPEC §8 "Missing day row → 0" | Keep for alerts/rolling; document chart/UI + dense export exception |
| **B** | `GGHSTATS_REPORT_PRIVATE` + `repo report` | Full doc checklist (README, man, env examples, usage/`--help`) |
| **C** | CHANGELOG / ROADMAP / SPEC header | Band row + `[Unreleased]` + “as of v1.5.0” when semantics match |

## Operator migration (fail-closed upgrade)

1. Pre-upgrade: `cp gghstats.db gghstats.db.pre-1.5.0`.
2. Deploy 1.5.0 → schema migrations for freshness + visibility (restart-safe
   where specified in implementation).
3. First sync populates `github_visibility`.
4. Verify `gghstats repo report ls`.
5. Private repos: `GGHSTATS_REPORT_PRIVATE=true` **or** `repo report set … include`.
6. Not data-loss: SQLite history retained; re-including restores report surfaces.

### What the operator should expect

- Empty/hidden dashboard until first sync after upgrade — expected fail-closed.
- Outside filter → pin with `repo add`; `report set include` alone does not collect.
- Freshness: `failed` / `never` = hard problems; `delayed` = GitHub publish lag;
  `missing` = hole inside observed span; healthy sync with a complete payload →
  expect `fresh`.

## Out of scope

- Carlok #22–#25 (done in 1.4.0).
- Aggregate per-repo coverage matrix on the index chart.
- Changing default sparse `GET .../traffic` to dense.
- Bulk “download all repos’ chart JSON” (JSONL / CLI later if needed).

## Parking lot (after 1.5.0)

1. Fail-closed operator helper (guidance/CLI after upgrade; do not weaken default).
2. Index coverage matrix — only if operators ask after living with per-repo freshness.
3. **Alert rules on uniques** — today traffic alerts evaluate `clones`/`views`
   **`count`** only (SPEC §8 / `DayCount` + range sums). Optional `metric` (or
   field) for GitHub **uniques** (e.g. unique cloners ≥ N) so operators can
   threshold the left side of UI pairs like `uniques/all`, not only totals.
   Must stay explicit in rule JSON and payloads (never ambiguous count vs uniques).

## Exit criteria

1. F-fresh + V-vis on `develop` (tests + migration smoke on a real DB copy).
2. V-json (`repo report ls --json` + useful filters) on `develop`.
3. U-legend on repo traffic charts (all bundled locales).
4. X-chart on `develop` (UI + dense dogfood + tests/i18n/docs).
5. SPEC §8 dual semantics + dense export documented.
6. Operator migration note landed.
7. `make release-check` green (user asks).
8. Merge develop→main + tag `v1.5.0` (user OK).

## Checklist

- [x] Finalize this plan and open a docs PR → `develop` (gitflow) — #38
- [x] Implement F-fresh (migrations, sync, API/UI freshness, chart null gaps)
- [x] Implement V-vis (migrations, report scope, CLI, fail-closed surfaces)
- [x] Local smoke: fail-closed → sync → exclude/restore one repo → freshness
- [x] CHANGELOG `[Unreleased]` for F-fresh + V-vis
- [x] ROADMAP 1.5.0 band row + Next retarget
- [x] SPEC §3.4 / §4.8 freshness + visibility; §8 dual missing-day semantics
- [ ] Implement V-json (`repo report ls --json` + filters)
- [ ] Implement U-legend (i18n: null gap vs confirmed zero)
- [ ] Implement X-chart (dense dogfood + repo-page download)
- [ ] Doc checklist re-verify (`GGHSTATS_REPORT_PRIVATE` + `repo report` on all surfaces post-merge)
- [ ] Operator migration note (README / catalog) polish if gaps remain
- [ ] `VERSION` 1.5.0 + `make release-check` + merge/tag (user OK)
