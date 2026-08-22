package sync

import (
	"log/slog"

	"github.com/hrodrig/gghstats/internal/github"
	"github.com/hrodrig/gghstats/internal/store"
)

// resolveTracked resolves the tracked repo set for a sync run:
//
//   - opts.Repos non-empty (explicit list) → returned verbatim, pins are NOT
//     merged (R10).
//   - otherwise FILTER ∪ pins: the filter-mapped ListRepos result, plus any
//     pinned repos not already present (R2 union of empty pins = no-op; R8 a
//     pin survives a `!fork` exclusion).
//
// Pinned repos carry only FullName here; metadata is fetched per-repo by
// ensureRepoMetadata during sync.
func resolveTracked(gh *github.Client, db *store.Store, opts Options) ([]github.Repo, error) {
	repos, err := resolveRepos(gh, opts)
	if err != nil {
		return nil, err
	}
	if len(opts.Repos) > 0 {
		return repos, nil
	}
	pins, err := db.ListPins()
	if err != nil {
		return nil, err
	}
	if len(pins) == 0 {
		return repos, nil
	}
	seen := make(map[string]bool, len(repos))
	for _, r := range repos {
		seen[r.FullName] = true
	}
	for _, p := range pins {
		if seen[p] {
			continue
		}
		seen[p] = true
		repos = append(repos, github.Repo{FullName: p})
	}
	return repos, nil
}

// syncFeaturedMeta refreshes stored metadata for every featured showcase
// entry using only the repo endpoint (no traffic). For a fork, the parent is
// resolved for upstream description + stars; for a plain repo the repo itself
// is the upstream. Per-row failures are logged and skipped — they never fail
// the sync cycle (R11).
func syncFeaturedMeta(gh *github.Client, db *store.Store) error {
	featured, err := db.ListFeatured()
	if err != nil {
		return err
	}
	for _, f := range featured {
		meta, err := gh.Repo(f.Name)
		if err != nil {
			slog.Warn("featured metadata failed", "repo", f.Name, "error", err)
			continue
		}
		upstream := meta
		parentName := ""
		if meta.Parent != nil && meta.Parent.FullName != "" {
			parentName = meta.Parent.FullName
			up, err := gh.Repo(parentName)
			if err != nil {
				slog.Warn("featured parent metadata failed", "repo", f.Name, "parent", parentName, "error", err)
				upstream = meta
			} else {
				upstream = up
			}
		}
		if err := db.UpsertFeaturedMeta(
			f.Name,
			parentName,
			upstream.FullName,
			upstream.DescriptionOrEmpty(),
			upstream.StargazersCount,
			meta.Fork,
		); err != nil {
			slog.Warn("featured metadata upsert failed", "repo", f.Name, "error", err)
		}
	}
	return nil
}
