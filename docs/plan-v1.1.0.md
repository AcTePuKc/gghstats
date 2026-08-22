# Plan — v1.1.0

**Status:** Planned — implement on `develop`; tag only after user OK.  
**Band goal:** Additive **catalog CLI** (pins) + **Featured** showcase. Operator stewards the box from the console (SSH/VPS: add, rm, sync, backup) — not a browser admin UI. Installed 1.0.x operators who do nothing stay identical.

Parent: [ROADMAP.md](../ROADMAP.md) · Spec: [SPEC.md](../SPEC.md) · Design: [superpowers/specs/2026-08-14-featured-and-repo-cli-design.md](superpowers/specs/2026-08-14-featured-and-repo-cli-design.md) · Tasks: [superpowers/plans/2026-08-14-featured-and-repo-cli.md](superpowers/plans/2026-08-14-featured-and-repo-cli.md)

## Versioning (SemVer)

Current **1.0.1**. Fixes until this ships: **`1.0.2`, `1.0.3`, `1.0.4`, … (`1.0.x`)**. Not `1.1.x`.

| Tag | Allowed |
|-----|---------|
| **1.0.x** | Patches on 1.0.1 only (no Line E) |
| **1.1.0** | E1 + E2 (or E1 only if E2 slips — then E2 waits for **1.2.0**, not a patch) |
| **1.1.x** | Patches **after** 1.1.0 is tagged (`1.1.1`, …) |
| **2.0.0** | Not this. ROADMAP Line B (webhooks / serious expected work) |

Do **not** ship Line E as `1.0.2`. Do **not** treat `1.1.x` as a feature dump.

## Prerequisites

| From | Requirement |
|------|-------------|
| **1.0.0 / 1.0.1** | SPEC freeze; FILTER + HTML `/` + `/h2h` stable |
| **Users** | Empty `pins` / `featured` → bit-identical chrome and discovery |

## In scope

| ID | Slice | Item | Notes |
|----|-------|------|--------|
| **E1** | Pins | `gghstats repo add\|rm\|ls` + SQLite `pins` | Union with `GGHSTATS_FILTER` for traffic sync and `/` |
| **E1s** | Sync | `tracked = filter(ListRepos) ∪ pins` | Explicit `opts.Repos` still wins (no union) |
| **E2** | Featured CLI | `gghstats featured add\|rm\|ls` + table `featured` | No token on add |
| **E2s** | Featured sync | Metadata only (`Client.Repo` + parent) | No clones/views/paths |
| **E2u** | UI | `GET /featured` + nav below H2H | Hide nav when count=0; i18n EN/ES/DE/FR/PT |
| **DOC** | Docs | SPEC, README, man, env.example, CHANGELOG | FILTER semantics documented as union |
| **REL** | Release | Dedicated `VERSION` 1.1.0 commit, then `release-check` | After features land; user must approve merge/tag |

## Out of scope

- Groups/tabs on `/`
- UI editors for pins/featured (D10: console stewards; HTML shows)
- Featured JSON API (1.2.0+)
- Dropping or ignoring `GGHSTATS_FILTER`
- Line B webhooks / Line C leaderboards
- Syncing every fork on the GitHub account
- GSD `.planning/` workflow

## Exit criteria

1. Design decisions D1–D9 implemented or explicitly deferred in CHANGELOG.
2. Upgrade with empty catalog: `/` and nav match 1.0.1 (no Featured link).
3. Pin of a `!fork`-excluded repo appears on `/` and in traffic sync.
4. `featured add` of a fork does **not** change `/` KPIs; `/featured` shows upstream link + fork line after meta sync.
5. `make test` green after E1 and after E2. All regression IDs **R1–R12** in the design spec have tests and pass.
6. Man page + `--help` list new commands. English artifacts.
7. Before tag: `make release-check` (cover ≥80%). User asks.

## Regression (cannot skip)

Empty catalog = 1.0.1 chrome and discovery. Table **R1–R12**: [design spec Testing](superpowers/specs/2026-08-14-featured-and-repo-cli-design.md). Red R = no ship.

## Checklist

- [ ] E1 store migration v6 + pin CRUD tests
- [ ] E1 CLI `repo add|rm|ls`
- [ ] E1s sync union + tests (`!fork` + pin)
- [ ] E2 featured CRUD tests + CLI
- [ ] E2s metadata sync (no traffic) + tests
- [ ] E2u `/featured` + nav hide-when-empty + i18n
- [ ] API-only does not mount Featured HTML
- [ ] Regression **R1–R12** tests exist and pass (empty catalog + no KPI leak)
- [ ] Existing `TestFilterExcludeFork`, dogfood API-only, i18n keys still green
- [ ] SPEC + README + man + env.example
- [ ] CHANGELOG `[Unreleased]` → `[1.1.0]` at VERSION bump
- [ ] `VERSION` 1.1.0 dedicated commit (user OK)
- [ ] `make release-check` (user asks)
- [ ] Merge `develop` → `main`, tag `v1.1.0` (user OK)

## After 1.1.0

- **1.1.x:** SemVer patches of 1.1.0 (`1.1.1`, …). No new features.
- **1.2.0 candidates:** Featured JSON dogfood; optional sitemap `/featured`; **unique cloners visible (index + repo)** — data already in `clone_uniques` (`SUM` of GitHub daily uniques, not lifetime machines). Index: column + KPI. Repo: uniques as primary number, clone events secondary. Label so 1466 is not hidden behind “1466 / 4304”.
- **2.0.0:** Line B (webhooks / rate-limit relief), not more vitrine scope.
