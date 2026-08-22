# Repo pins CLI + Featured showcase (v1.1.0) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship **gghstats v1.1.0** Line E: SQLite-backed `gghstats repo` pins (union with `GGHSTATS_FILTER`) and `gghstats featured` + `/featured` vitrine, with empty catalog identical to 1.0.x.

**Architecture:** New migration v6 tables `pins` and `featured`. Sync tracked set = filtered `ListRepos` ∪ pins. Featured refresh uses `github.Client.Repo` only (no traffic). HTML `/featured` is server-rendered; nav link hidden when `COUNT(featured)=0`.

**Tech Stack:** Go, SQLite (`modernc.org/sqlite`), existing `cmd/gghstats` stdlib flags, `web/templates`, `internal/i18n/locales`.

**Spec:** [docs/superpowers/specs/2026-08-14-featured-and-repo-cli-design.md](../specs/2026-08-14-featured-and-repo-cli-design.md)  
**Band:** [docs/plan-v1.1.0.md](../../plan-v1.1.0.md)

## Global Constraints

- Run commands from the **gghstats repository root**.
- English only for project artifacts (UI strings via i18n JSON; CLI stderr English).
- Work on `develop`; never merge/tag `main` without explicit user approval in the same turn.
- Before each `git commit`, show the full message and wait for user approval.
- Do not delete files without explicit approval.
- Keep `golang.org/x/net v0.57.0` pin (`make check-x-net-pin`).
- Do **not** bump `VERSION` / man `.TH` / CHANGELOG version header until a dedicated 1.1.0 bump commit (user OK).
- Do **not** add Featured JSON API in this plan (1.2.0).
- Do **not** add HTML forms to add/remove pins or featured (D10: CLI stewardship / console nostalgia).
- Empty `featured` → no nav link. FILTER semantics unchanged except union with pins for discovery.

## File map

| File | Role |
|------|------|
| `internal/store/store.go` | `migrateV6`; pin/featured CRUD + `CountFeatured` |
| `internal/store/store_test.go` | Migration v6; CRUD; sort; invalid names |
| `internal/store/reponame.go` | `NormalizeRepoName` (shared by store + CLI) |
| `internal/sync/sync.go` | Union pins after `resolveRepos`; `syncFeaturedMeta` |
| `internal/sync/sync_test.go` | `!fork` + pin included; featured-only excluded from traffic |
| `cmd/gghstats/catalog.go` | `runRepo` / `runFeatured` add\|rm\|ls |
| `cmd/gghstats/catalog_test.go` | CLI against temp DB |
| `cmd/gghstats/main.go` | Register commands; usage text |
| `internal/server/server.go` | Mount `/featured`; `ShowFeatured` on layout |
| `internal/server/featured.go` | Handler |
| `web/templates/featured.html` | Cards |
| `web/templates/layout.html` | Nav link below H2H |
| `internal/i18n/locales/{en,es,de,fr,pt-br}.json` | `nav.featured`, `featured.*`, `meta.featured` |
| `internal/server/*_test.go` | Nav hidden/shown; API-only skips HTML |
| `SPEC.md` | §4.2 union; new CLI; `/featured` |
| `README.md` | FILTER + pins + featured |
| `contrib/man/man1/gghstats.1` | Commands (not `.TH` version until bump) |
| `contrib/gghstats.env.example` | Comment: do not list 30 names / showcase in FILTER |
| `CHANGELOG.md` | `[Unreleased]` notes until VERSION bump |

E1 = Tasks 1–3. E2 = Tasks 4–7. Docs = Task 8. Stop after Task 3 if cutting E2 to 1.2.0.

---

### Task 1: Migration v6 + pin/featured store

**Files:**
- Create: `internal/store/reponame.go`
- Modify: `internal/store/store.go` (`migrate` slice + CRUD)
- Modify: `internal/store/store_test.go` (`TestVersionedMigrations` want **6**)
- Test: `go test ./internal/store/ -count=1 -run 'TestVersionedMigrations|TestPins|TestFeatured|TestNormalizeRepoName'`

**Interfaces:**
- Produces:
  - `func NormalizeRepoName(s string) (string, error)`
  - `func (s *Store) AddPin(name string) (added bool, err error)`
  - `func (s *Store) RemovePin(name string) (removed bool, err error)`
  - `func (s *Store) ListPins() ([]string, error)`
  - `func (s *Store) AddFeatured(name string) (added bool, err error)`
  - `func (s *Store) RemoveFeatured(name string) (removed bool, err error)`
  - `func (s *Store) ListFeatured() ([]FeaturedRow, error)` — `ORDER BY sort, name`
  - `func (s *Store) CountFeatured() (int, error)`
  - `func (s *Store) UpsertFeaturedMeta(name, parent, upstream, desc string, stars int, fork bool) error`

```go
type FeaturedRow struct {
	Name                 string
	CreatedAt            string
	Sort                 int
	ParentFullName       string
	UpstreamFullName     string
	UpstreamDescription  string
	UpstreamStars        int
	Fork                 bool
	MetaUpdatedAt        string
}
```

- [ ] **Step 1: Write failing tests**

```go
func TestNormalizeRepoName(t *testing.T) {
	got, err := NormalizeRepoName("  hrodrig/awesome-readme  ")
	if err != nil || got != "hrodrig/awesome-readme" {
		t.Fatalf("got %q %v", got, err)
	}
	for _, bad := range []string{"", "nope", "a/b/c", "*", "hrodrig/*", "a /b"} {
		if _, err := NormalizeRepoName(bad); err == nil {
			t.Errorf("NormalizeRepoName(%q) want error", bad)
		}
	}
}

func TestPinsCRUD(t *testing.T) {
	s := tempDB(t)
	added, err := s.AddPin("hrodrig/pgwd")
	if err != nil || !added {
		t.Fatalf("AddPin: added=%v err=%v", added, err)
	}
	added, err = s.AddPin("hrodrig/pgwd")
	if err != nil || added {
		t.Fatalf("idempotent AddPin: added=%v err=%v", added, err)
	}
	pins, err := s.ListPins()
	if err != nil || len(pins) != 1 || pins[0] != "hrodrig/pgwd" {
		t.Fatalf("ListPins: %v %v", pins, err)
	}
	removed, err := s.RemovePin("hrodrig/pgwd")
	if err != nil || !removed {
		t.Fatalf("RemovePin: %v %v", removed, err)
	}
	removed, err = s.RemovePin("hrodrig/pgwd")
	if err != nil || removed {
		t.Fatalf("RemovePin missing: removed=%v err=%v", removed, err)
	}
}

func TestFeaturedSortAndMeta(t *testing.T) {
	s := tempDB(t)
	if _, err := s.AddFeatured("hrodrig/b"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddFeatured("hrodrig/a"); err != nil {
		t.Fatal(err)
	}
	rows, err := s.ListFeatured()
	if err != nil || len(rows) != 2 || rows[0].Name != "hrodrig/b" || rows[1].Name != "hrodrig/a" {
		t.Fatalf("sort by insert, got %+v err=%v", rows, err)
	}
	if err := s.UpsertFeaturedMeta("hrodrig/b", "up/b", "up/b", "desc", 99, true); err != nil {
		t.Fatal(err)
	}
	rows, _ = s.ListFeatured()
	if rows[0].UpstreamStars != 99 || !rows[0].Fork || rows[0].UpstreamFullName != "up/b" {
		t.Fatalf("meta: %+v", rows[0])
	}
	n, err := s.CountFeatured()
	if err != nil || n != 2 {
		t.Fatalf("CountFeatured=%d %v", n, err)
	}
}
```

Update `TestVersionedMigrations`: `want 6`; assert `pins` and `featured` tables exist.

- [ ] **Step 2: Run tests — expect FAIL** (missing types/migrate)

Run: `go test ./internal/store/ -count=1 -run 'TestVersionedMigrations|TestPins|TestFeatured|TestNormalizeRepoName' -v`

Expected: FAIL compile or `user_version = 5, want 6`.

- [ ] **Step 3: Implement**

`NormalizeRepoName`: trim; regexp `^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`.

`migrateV6`: SQL from the design spec. Append to `migrations` slice.

`AddPin` / `AddFeatured`: normalize; `INSERT OR IGNORE`; return `RowsAffected()==1`. Featured `sort` = `COALESCE(MAX(sort),-1)+1`. `created_at` = UTC RFC3339.

`Remove*`: `DELETE`; `RowsAffected()==1`.

`UpsertFeaturedMeta`: `UPDATE featured SET parent_full_name=?, upstream_full_name=?, ... meta_updated_at=? WHERE name=?`.

- [ ] **Step 4: Run tests — expect PASS**

Run: `go test ./internal/store/ -count=1`

Expected: PASS.

- [ ] **Step 5: Commit** (after user approves message)

```
feat(store): pins and featured tables (migration v6)
```

---

### Task 2: CLI `repo` and `featured`

**Files:**
- Create: `cmd/gghstats/catalog.go`
- Create: `cmd/gghstats/catalog_test.go`
- Modify: `cmd/gghstats/main.go` (`cliCommands`, `usage`)
- Test: `go test ./cmd/gghstats/ -count=1 -run 'TestRunRepo|TestRunFeatured|TestRunCLI'`

**Interfaces:**
- Consumes: store CRUD from Task 1
- Produces: `runRepo(args []string) error`, `runFeatured(args []string) error`

- [ ] **Step 1: Failing CLI tests**

```go
func TestRunRepoAddLsRm(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "t.db")
	if err := runRepo([]string{"add", "hrodrig/pgwd", "--db", dbPath}); err != nil {
		t.Fatal(err)
	}
	if err := runRepo([]string{"add", "hrodrig/pgwd", "--db", dbPath}); err != nil {
		t.Fatal(err) // idempotent
	}
	if err := runRepo([]string{"ls", "--db", dbPath}); err != nil {
		t.Fatal(err)
	}
	if err := runRepo([]string{"rm", "hrodrig/pgwd", "--db", dbPath}); err != nil {
		t.Fatal(err)
	}
	if err := runRepo([]string{"rm", "hrodrig/pgwd", "--db", dbPath}); err == nil {
		t.Fatal("rm missing want error")
	}
	if err := runRepo([]string{"add", "not-a-repo", "--db", dbPath}); err == nil {
		t.Fatal("invalid name want error")
	}
}
```

Mirror for `runFeatured`. Add `TestRunCLI` unknown subcommand already covers dispatch once registered.

- [ ] **Step 2: Run — expect FAIL** (undefined `runRepo`)

- [ ] **Step 3: Implement**

Pattern: `cmd/gghstats/alert.go` (subcommand switch) + `backup.go` (`--db` default).

```
gghstats repo add|rm|ls [OWNER/REPO] [--db PATH]
gghstats featured add|rm|ls [OWNER/REPO] [--db PATH]
```

`add`/`rm` require positional `OWNER/REPO`. `ls` prints one name per line (`featured ls` may print `name\tupstream` when meta set). Duplicate add: stderr `already pinned` / `already featured`, exit 0. Missing rm: `fmt.Errorf("%s is not pinned", name)`.

Register in `cliCommands`. Extend `usage` Commands list.

- [ ] **Step 4:** `go test ./cmd/gghstats/ -count=1` PASS

- [ ] **Step 5: Commit** (user OK)

```
feat(cli): gghstats repo and featured add/rm/ls
```

---

### Task 3: Sync union pins (E1 complete)

**Files:**
- Modify: `internal/sync/sync.go`
- Modify: `internal/sync/sync_test.go`
- Test: `go test ./internal/sync/ -count=1 -run 'TestResolve|TestUnion|TestRun'`

**Interfaces:**
- Consumes: `ListPins()`
- Produces: `func unionPinRepos(discovered []github.Repo, pins []string) []github.Repo`

- [ ] **Step 1: Tests**

Reuse httptest `ListRepos` from `TestFilterExcludeFork`. After filter `*,!fork`, discovered lacks forks. Call `unionPinRepos` with pin `a/2` (fork) → result includes `a/2`.

`TestUnionPinsDoesNotDedupeWrong`: pin already in discovered → still one entry.

`TestRunExplicitReposSkipsPins`: `Options.Repos` non-empty → do not merge pins (call `Run` with httptest that 404s if `/user/repos` hit; only explicit names sync). Need a store with a pin that must **not** be fetched.

- [ ] **Step 2: FAIL then implement**

In `Run`, after `resolveRepos`:

```go
	if len(opts.Repos) == 0 {
		pins, err := db.ListPins()
		if err != nil {
			return result, fmt.Errorf("list pins: %w", err)
		}
		repos = unionPinRepos(repos, pins)
	}
```

`unionPinRepos`: index by `FullName`; append `{FullName: pin}` for missing pins (preserve pin string).

- [ ] **Step 3:** `go test ./internal/sync/ ./internal/store/ -count=1` PASS

- [ ] **Step 4: Commit** (user OK)

```
feat(sync): union GGHSTATS_FILTER discovery with pinned repos
```

**E1 ship cut lives here.** `/` already lists whatever is in `repos` after sync. No HTML change required for pins.

---

### Task 4: Featured metadata sync (E2s)

**Files:**
- Modify: `internal/sync/sync.go`
- Modify: `internal/sync/sync_test.go`
- Test: `go test ./internal/sync/ -count=1 -run 'TestSyncFeatured'`

**Interfaces:**
- Consumes: `ListFeatured`, `UpsertFeaturedMeta`, `github.Client.Repo`
- Produces: `func syncFeaturedMeta(gh *github.Client, db *store.Store)` (void; log per-row errors)

- [ ] **Step 1: httptest**

Server: `GET /repos/hrodrig/awesome-readme` → fork, parent `matiassingers/awesome-readme`. `GET /repos/matiassingers/awesome-readme` → stars 100, description. `GET /repos/.../traffic/*` must **not** be called for featured-only (t.Error if hit).

Seed featured row; `FILTER` that excludes forks; `Run`; assert `ListFeatured()[0].UpstreamStars==100` and no clones rows for that name.

- [ ] **Step 2: Implement**

Call `syncFeaturedMeta` at end of `Run` (after deltas). Per name: `gh.Repo(name)`; if parent non-empty, `gh.Repo(parent)` for description+stars; else use the repo itself as upstream. Failures: `slog.Warn`, continue.

- [ ] **Step 3:** tests PASS

- [ ] **Step 4: Commit** (user OK)

```
feat(sync): refresh featured upstream metadata without traffic
```

---

### Task 5: `/featured` HTML + nav

**Files:**
- Create: `internal/server/featured.go`
- Create: `web/templates/featured.html`
- Modify: `web/templates/layout.html`
- Modify: `internal/server/server.go` (`layoutData.ShowFeatured`, mount route, `renderLayoutStatus` fills flag)
- Create or modify: `internal/server/featured_test.go`, `internal/server/api_only_test.go`
- Test: `go test ./internal/server/ -count=1 -run 'TestFeatured|TestAPIOnly'`

**Interfaces:**
- Consumes: `CountFeatured`, `ListFeatured`
- Produces: `GET /featured` HTML; `layoutData.ShowFeatured bool`; `PageID == "featured"`

- [ ] **Step 1: Tests**

Empty store: `GET /` body does not contain `href="/featured"`. `GET /featured` status 200.

After `AddFeatured`: `GET /` contains Featured nav href; `GET /featured` contains repo name.

`APIOnly: true`: `GET /featured` 404 (or not the HTML page — same as `/h2h` today).

- [ ] **Step 2: FAIL (i18n keys missing is OK until Task 6 if tests search href only)**

- [ ] **Step 3: Implement**

`renderLayoutStatus`: if `cfg.Store != nil`, `n, _ := cfg.Store.CountFeatured(); data.ShowFeatured = n > 0`.

layout.html after H2H:

```html
{{if .ShowFeatured}}
<a class="nav-link app-brutal-nav px-3 py-2 text-uppercase fw-semibold small{{if eq .PageID "featured"}} active{{end}}" href="/featured" title="{{call .T "nav.featured"}}">{{call .T "nav.featured"}}</a>
{{end}}
```

Mount next to `/h2h` inside `!cfg.APIOnly`. Template cards: link `https://github.com/{{.UpstreamFullName}}` (fallback `.Name` if upstream empty); fork line if `.Fork`.

- [ ] **Step 4:** server tests PASS (may fail on missing i18n until Task 6 — if `T` returns key, tests should still find href)

- [ ] **Step 5: Commit** (user OK)

```
feat(ui): /featured page and nav link below H2H
```

---

### Task 6: i18n

**Files:**
- Modify: `internal/i18n/locales/en.json`, `es.json`, `de.json`, `fr.json`, `pt-br.json`
- Modify: any i18n completeness test if one exists (`grep featured` after)

English source:

| Key | EN |
|-----|-----|
| `nav.featured` | Featured |
| `featured.title` | Featured |
| `featured.intro` | Repositories I recommend — originals linked first; my fork noted when I keep a copy. |
| `featured.fork_of` | Forked from |
| `featured.empty` | No featured repositories yet. |
| `meta.featured` | Curated repositories recommended by this gghstats operator. |

ES: Destacados / Fork de … (match existing `repo.fork_of` tone). DE/FR/PT-BR: complete, not English leftovers.

- [ ] **Step 1:** add keys all five files
- [ ] **Step 2:** `go test ./internal/i18n/ ./internal/server/ -count=1` PASS
- [ ] **Step 3: Commit** (user OK)

```
feat(i18n): Featured nav and page strings
```

---

### Task 7: SPEC + operator docs (no VERSION bump)

**Files:**
- Modify: `SPEC.md` §4.2 (union pins; featured metadata pass); §5 CLI table; HTML route list
- Modify: `README.md` configuration / exclude-forks section
- Modify: `contrib/man/man1/gghstats.1` Commands (not `.TH` date/version)
- Modify: `contrib/gghstats.env.example` FILTER comment
- Modify: `CHANGELOG.md` under `[Unreleased]`
- Modify: `docs/plan-v1.1.0.md` checklist boxes as items land
- Test: `go test ./...` (no `-race` unless usual `make test`)

SPEC bullets:

```markdown
- Empty explicit repo list → `ListRepos` then `GGHSTATS_FILTER`, then **union** SQLite `pins`.
- `featured` rows are not in the tracked traffic set unless also discovered or pinned.
- After traffic workers: metadata refresh for `featured` via `GET /repos/{name}` (and parent). Failures do not fail the cycle.
```

CLI:

```
gghstats repo add|rm|ls
gghstats featured add|rm|ls
```

CHANGELOG `[Unreleased]`:

```markdown
### Added
- **Repo pins:** `gghstats repo add|rm|ls` stores extras in SQLite; sync unions them with `GGHSTATS_FILTER`.
- **Featured:** `gghstats featured add|rm|ls` and `/featured` showcase (upstream link + fork note). Nav hidden when empty.
```

- [ ] **Step 1:** edit docs
- [ ] **Step 2:** `go test ./... -count=1` PASS
- [ ] **Step 3: Commit** (user OK)

```
docs: pins, Featured, FILTER union for 1.1.0
```

---

### Task 8: Lint sanity (no tag)

- [ ] `gofmt -s -l .` clean; `go vet ./...`
- [ ] `make check-x-net-pin` if that target exists
- [ ] Do **not** run `make release-check` / merge `main` / tag unless the user asks

---

### Task 9: Regression gate (mandatory — 4k installs)

**Files:** tests added in Tasks 1–5 must include IDs **R1–R12** from the design spec. This task is the **checklist**, not new product code.

**Spec:** [Testing (normative)](../specs/2026-08-14-featured-and-repo-cli-design.md)

- [ ] **Step 1: Empty catalog (R1–R6, R12)**

`go test ./internal/server/ ./internal/sync/ ./internal/store/ -count=1 -run 'TestEmptyCatalog|TestFilterExcludeFork|TestAPIOnly|TestVersionedMigrations|TestH2H' -v`

Expect: `GET /` has no `href="/featured"`; `!fork` still drops forks; API-only has no Featured HTML; user_version 6 from v5 DB; `/h2h` 200.

- [ ] **Step 2: Catalog isolation (R7–R11)**

`go test ./internal/sync/ ./internal/server/ -count=1 -run 'TestFeaturedOnlyNotTracked|TestPinOverridesForkFilter|TestExplicitReposSkipsPins|TestFeaturedMetaNoTraffic|TestFeaturedNavToggle' -v`

Expect: featured-only absent from KPIs; pin of fork present; explicit `Repos` ignores pins; no `/traffic/clones` for featured-only; nav appears then disappears after last `rm`.

- [ ] **Step 3: Full suite**

Run: `make test`

Expected: PASS (race). Do not ship a slice if this is red.

- [ ] **Step 4: Commit only if R1–R12 were missing names/comments** (user OK). If tests already landed in Tasks 1–5, no extra commit — still **run** this gate before calling E1 or E2 done.

---

## Spec coverage

| Spec item | Task |
|-----------|------|
| migrate v6 pins/featured | 1 |
| Normalize names | 1 |
| CLI two families | 2 |
| FILTER ∪ pins; explicit Repos wins | 3 |
| Featured-only not on traffic | 4 |
| Metadata GET only | 4 |
| `/featured` + nav hide | 5 |
| i18n | 6 |
| FILTER unchanged docs | 7 |
| No JSON API / no VERSION bump | 7–8 (omitted) |
| Regression R1–R12 | 9 (plus tests in 1–5) |

## Weekend order

1. Tasks 1–3 (E1) with café. Dashboard already useful (`repo add` vs `.env` list).
2. **Task 9 empty-catalog half (R2, R8, R10, R12)** before calling E1 done.
3. Tasks 4–7 (E2) with bachata.
4. **Task 9 full (R1–R12) + `make test`**. Red = no ship.
5. VERSION 1.1.0 **later**, dedicated commit, user OK.
