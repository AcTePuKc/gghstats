# Plan — v1.4.0

**Status:** **In progress** — implement on `develop`; **do not** bump `VERSION` / tag until user OK after Carlok UI issues (#22–#25) land (or are explicitly deferred).

**Band goal:** Close leftover **1.1 / unique-cloners** product debt: clearer unique-cloner UX on repo (and index if needed), **Featured JSON** dogfood for API-only, and SEO **`/featured`** in the sitemap. Patches from Carlok may ship in the same release.

Parent: [ROADMAP.md](../ROADMAP.md) · Spec: [SPEC.md](../SPEC.md) · Featured design: [superpowers/specs/2026-08-14-featured-and-repo-cli-design.md](superpowers/specs/2026-08-14-featured-and-repo-cli-design.md)

## Versioning

Current **1.3.0**. This band → **1.4.0** (minor). No `1.3.x` feature dump: Carlok items are 1.3 polish that may ride in **1.4.0** Unreleased.

| Tag | Allowed |
|-----|---------|
| **1.3.x** | Optional patches if Carlok ships alone before 1.4 features |
| **1.4.0** | U1 + F1 + S1 (below); optional Carlok #22–#25 |
| **2.0.0** | Line B — not this band |

## In scope

| ID | Slice | Item | Notes |
|----|-------|------|--------|
| **S1** | Sitemap | Include `GET /featured` in `/sitemap.xml` when showcase non-empty | Same indexability rules as `/` / `/h2h`; API-only still omits SEO |
| **U1** | Uniques UX | Repo (and index as needed): uniques primary, events secondary; honest labels | Data already in `clone_uniques`; avoid implying lifetime machines |
| **F1** | Featured JSON | `GET /api/v1/featured` dogfood (list + query parity with HTML where practical) | Auth like other API; no traffic fields; SPEC + `docs/api.md` + contract test |
| **DOC** | Docs | SPEC / README / api.md / CHANGELOG as slices land | English |
| **REL** | Release | Dedicated `VERSION` 1.4.0 after features + Carlok gate | User must approve `release-check` / main / tag |

## Out of scope

- Line B webhooks / GraphQL
- Featured HTML editors (CLI stewardship stays)
- Groups/tabs on `/`
- Changing meaning of `clone_uniques` (still SUM of daily GitHub uniques)

## Exit criteria

1. S1/U1/F1 done or explicitly deferred in CHANGELOG with user OK.
2. Empty featured catalog → sitemap unchanged (no `/featured` URL).
3. API-only + token can list featured via documented JSON (F1).
4. `make test` green; dogfood / SEO tests cover new paths.
5. Before tag: Carlok #22–#25 closed **or** user defers them out of 1.4.0.
6. `make release-check` (user asks); merge `develop` → `main`, tag `v1.4.0` (user OK).

## Checklist

- [x] S1 sitemap `/featured` + tests
- [x] U1 repo/index unique-cloner labels / hierarchy + i18n
- [x] F1 `GET /api/v1/featured` + SPEC + api.md + tests
- [x] CHANGELOG `[Unreleased]` notes per slice
- [ ] Wait / integrate Carlok #22–#25 (or defer note)
- [ ] `VERSION` 1.4.0 dedicated commit (user OK)
- [ ] `make release-check` (user asks)
- [ ] Merge `develop` → `main`, tag `v1.4.0` (user OK)

## Slice order

1. **S1** (this PR stream) — smallest, unblocks SEO
2. **U1** — UX / i18n
3. **F1** — API contract
4. Carlok + REL
