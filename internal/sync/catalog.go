package sync

import (
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
