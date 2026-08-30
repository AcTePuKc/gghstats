# Plan — v1.4.0

**Status:** **Closed** — code on `develop` (PR #33 + U1/F1/S1). `VERSION` **1.4.0** bump in this release commit. Tag / merge to `main` only after user OK (`release-check`).

**Band goal:** Close leftover **1.1 / unique-cloners** product debt: clearer unique-cloner UX on repo (and index if needed), **Featured JSON** dogfood for API-only, and SEO **`/featured`** in the sitemap. Carlok UI issues (#22–#25) shipped in the same release.

Parent: [ROADMAP.md](../ROADMAP.md) · Spec: [SPEC.md](../SPEC.md) · Featured design: [superpowers/specs/2026-08-14-featured-and-repo-cli-design.md](superpowers/specs/2026-08-14-featured-and-repo-cli-design.md)

## Versioning

Band → **1.4.0** (minor).

| Tag | Allowed |
|-----|---------|
| **1.4.0** | U1 + F1 + S1 + Carlok #22–#25 |
| **2.0.0** | Line B — not this band |

## In scope

| ID | Slice | Item | Notes |
|----|-------|------|--------|
| **S1** | Sitemap | Include `GET /featured` in `/sitemap.xml` when showcase non-empty | Same indexability rules as `/` / `/h2h`; API-only still omits SEO |
| **U1** | Uniques UX | Repo (and index as needed): uniques primary, events secondary; honest labels | Data already in `clone_uniques`; avoid implying lifetime machines |
| **F1** | Featured JSON | `GET /api/v1/featured` dogfood (list + query parity with HTML where practical) | Auth like other API; no traffic fields; SPEC + `docs/api.md` + contract test |
| **C22–C25** | Carlok UI | Locale chart/stats (#22), JSONL UTC filename (#23), stats help tooltips (#24), Rank header (#25) | Merged via PR #33 |
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
- [x] Carlok #22–#25 on `develop` (PR #33)
- [x] `VERSION` 1.4.0 dedicated commit
- [ ] `make release-check` (user asks)
- [ ] Merge `develop` → `main`, tag `v1.4.0` (user OK) — closes #22–#25 on default branch

## Slice order

1. **S1** — smallest, unblocks SEO
2. **U1** — UX / i18n
3. **F1** — API contract
4. Carlok + REL
