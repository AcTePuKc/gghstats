# Catalog & Featured showcase

> Applies to **gghstats v1.1.0+**. This page is the operator reference for the
> two local SQLite catalogs introduced in 1.1.0: **pins** (`gghstats repo`) and
> **featured** (`gghstats featured`). The `README.md` links here so the front
> page stays short; this is the detailed guide.

The dashboard **shows**; the CLI **stewards**. There is no browser admin UI.
You manage pins and the showcase from the console — typically over SSH on the
box that runs `gghstats serve`.

---

## The two catalogs, in one sentence

| Catalog | CLI | What it does | Appears on `/`? |
|---|---|---|---|
| **Pins** | `gghstats repo` | Extends `GGHSTATS_FILTER` discovery with extra repos | **Yes** — row + traffic sync |
| **Featured** | `gghstats featured` | Editorial showcase of repos worth highlighting | **No** — only on `/featured` |

They are two independent tables (`pins` and `featured`) in the same SQLite
database. Adding to one never affects the other.

---

## Pins — `gghstats repo`

A **pin** adds a repository to the traffic set regardless of what
`GGHSTATS_FILTER` matches. The effective discovery set is:

```
tracked = result_of(FILTER) ∪ pins
```

That single rule is the whole feature. Because it is a *union*, a pin wins
over exclusions: pin a fork while `FILTER='!fork'`, and that fork still gets
traffic sync and a row on `/` — that is the point of the feature.

```bash
gghstats repo add hrodrig/extra-one    # pin (idempotent — "already pinned" is fine)
gghstats repo rm  hrodrig/extra-one    # unpin (error if it was not pinned)
gghstats repo ls                        # one OWNER/REPO per line, in insertion order
gghstats repo --help
```

- **No GitHub token** is required — `add`/`rm`/`ls` are local SQLite writes.
- `repo add` is **idempotent**: adding an already-pinned repo returns success.
- `repo rm` on a repo that is **not** pinned is an error (exit 1), so you
  cannot "succeed" at removing something that was not there.
- Names must be `OWNER/REPO` (letters, digits, `.`, `_`, `-`). `*` and
  FILTER-style syntax are rejected.

### When to pin (vs. extending FILTER)

Stuffing dozens of one-off names into `GGHSTATS_FILTER` is an anti-pattern:
the filter is the **discovery baseline**, not a hit-list. Keep `FILTER` as the
broad `owner/*, !fork, !archived` shape, and use `repo add` for the
exceptions that matter:

- a specific fork you track despite the `!fork` baseline,
- a repo you archived in GitHub but still want on the dashboard,
- a high-value project outside your usual filter scope.

---

## Featured — `gghstats featured`

A **featured** entry curates a repository into the `/featured` showcase page —
editorial "here's work worth looking at", independent of traffic tracking.

```bash
gghstats featured add hrodrig/awesome-readme    # showcase (idempotent)
gghstats featured rm  hrodrig/awesome-readme    # remove (error if absent)
gghstats featured ls                             # one OWNER/REPO per line
gghstats featured --help
```

- **No GitHub token** on `add`/`rm`/`ls` (local writes, same as pins).
- `featured add` is **idempotent**; `featured rm` errors if the entry is absent.
- Featured repos do **not** appear on `/` and do **not** change `/` KPIs.

### What `/featured` shows

Each entry renders as a card with:

| Element | Source |
|---|---|
| Title (links to GitHub) | `upstream_full_name`, else `name` |
| "Fork of X" badge | shown when the entry is a fork |
| Description | `upstream_description` from GitHub metadata |
| Star count | `upstream_stars` from GitHub metadata |
| "View on GitHub" button | same GitHub URL as the title |
| Last-updated timestamp | `meta_updated_at` |

The card title always links to `https://github.com/<upstream>`, **not** to an
internal dashboard route. (An earlier draft linked to an internal path that
404'd — this is now covered by a regression test.)

### Metadata is populated by sync, not by `add`

`featured add` only records the `OWNER/REPO`. The description, stars, fork
status, and upstream name come from a lightweight metadata refresh that runs
at the end of a sync cycle: one `Client.Repo(name)` call (plus the parent for
forks), and **no traffic calls**. This is deliberate — featured-only repos
must not generate `/traffic/clones` or `/traffic/views` requests.

Until that first refresh runs, the card shows **name + link only**, with no
stars or description. After a sync (or a manual `POST /api/v1/sync`), the
card fills in.

### Nav hide-when-empty

The sidebar link **Featured** (between **H2H** and the rest) only renders when
the `featured` table has at least one row. An empty catalog keeps the default
1.0.x chrome — no dead link, no empty page. This is regression R1.

---

## Shared mechanics

### `--db` and `GGHSTATS_DB`

Both commands read the same database as `serve`. Omit `--db` and they use
`GGHSTATS_DB`, falling back to the platform default:

| OS | Default path |
|---|---|
| macOS | `~/Library/Application Support/gghstats/gghstats.db` |
| Linux | `~/.config/gghstats/gghstats.db` |

```bash
gghstats featured ls --db /data/gghstats/gghstats.db
GGHSTATS_DB=/data/gghstats/gghstats.db gghstats repo ls
```

The CLI and `serve` **must** point at the same file, or changes made in the
console will not show on the dashboard.

### Typical SSH flow

```bash
# on the box running gghstats serve
gghstats repo add     my-org/the-important-fork
gghstats featured add matiassingers/awesome-readme
gghstats featured ls
# next sync cycle fetches metadata; or trigger now:
curl -X POST http://127.0.0.1:8080/api/v1/sync   # if API token is set
```

### Troubleshooting

| Symptom | Cause / fix |
|---|---|
| `repo rm` / `featured rm` returns "not found" / "not pinned" | The row was never added, or you are pointing at a different DB (`--db`). |
| `invalid repo name: expected OWNER/REPO` | You passed a filter expression, `*`, or a bare name. Use `owner/repo`. |
| Featured card shows no stars / description | Metadata sync has not run yet (or sync is disabled in `--demo`). |
| Added a repo but `/` or `/featured` does not change | CLI `--db` and `serve` DB differ, or `serve` was not restarted to pick up a schema change. |

---

## See also

- **[SPEC.md](../SPEC.md)** — normative contract (acceptance R1–R12, sync/API).
- design spec for the record — `docs/superpowers/specs/2026-08-14-featured-and-repo-cli-design.md`
- **[README.md](../README.md) → Catalog** — the short pointer version.
- **[plan-v1.1.0.md](plan-v1.1.0.md)** — band scope, exit criteria, checklist.
