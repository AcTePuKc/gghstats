# Plan — v1.1.0

**Status:** **Closed** — **v1.1.0** shipped (2026-08-22): Line E pins CLI ∪ FILTER + Featured HTML/CLI. Next: [ROADMAP.md](../ROADMAP.md) (1.2.0 / 1.3.0 already tagged; remaining product work listed under **Next**).

**Band goal:** Additive **catalog CLI** (pins) + **Featured** showcase. Operator stewards the box from the console (SSH/VPS: add, rm, sync, backup) — not a browser admin UI. Installed 1.0.x operators who do nothing stay identical.

Parent: [ROADMAP.md](../ROADMAP.md) · Spec: [SPEC.md](../SPEC.md) · Design: [superpowers/specs/2026-08-14-featured-and-repo-cli-design.md](superpowers/specs/2026-08-14-featured-and-repo-cli-design.md) · Tasks: [superpowers/plans/2026-08-14-featured-and-repo-cli.md](superpowers/plans/2026-08-14-featured-and-repo-cli.md)

## Versioning (SemVer)

Shipped as **1.1.0**. Current tree may be a later **1.x** (see root **`VERSION`**).

| Tag | Allowed |
|-----|---------|
| **1.0.x** | Patches on 1.0.1 only (no Line E) — historical |
| **1.1.0** | E1 + E2 (this band) |
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

- [x] E1 store migration v6 + pin CRUD tests
- [x] E1 CLI `repo add|rm|ls`
- [x] E1s sync union + tests (`!fork` + pin)
- [x] E2 featured CRUD tests + CLI
- [x] E2s metadata sync (no traffic) + tests
- [x] E2u `/featured` + nav hide-when-empty + i18n
- [x] API-only does not mount Featured HTML
- [x] Regression **R1–R12** tests exist and pass (empty catalog + no KPI leak)
- [x] Existing `TestFilterExcludeFork`, dogfood API-only, i18n keys still green
- [x] SPEC + README + man + env.example
- [x] CHANGELOG `[Unreleased]` → `[1.1.0]` at VERSION bump
- [x] `VERSION` 1.1.0 dedicated commit (user OK)
- [x] `make release-check` (user asks)
- [x] Merge `develop` → `main`, tag `v1.1.0` (user OK)

## After 1.1.0

- **1.1.x:** SemVer patches of 1.1.0 (`1.1.1`, …). No new features.
- **Shipped later:** **1.2.0** — Featured pagination/search/sort + compact numbers; **1.3.0** — index unique-cloners visibility, `# - %` rank/share, JSONL export (see CHANGELOG).
- **Still open (next minor):** repo-page unique cloners; Featured JSON dogfood; optional sitemap `/featured`. UI polish tracked in GitHub issues after 1.3.0.
- **2.0.0:** Line B (webhooks / rate-limit relief), not more vitrine scope.
