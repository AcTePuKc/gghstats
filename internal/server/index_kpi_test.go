package server

import (
	"testing"

	"github.com/hrodrig/gghstats/internal/store"
)

func TestSumIndexKPIsIncludesUniques(t *testing.T) {
	stars, forks, clones, cloneUniques, views, viewUniques := sumIndexKPIs([]store.RepoSummary{
		{Stars: 1, Forks: 2, TotalClones: 10, CloneUniques: 4, TotalViews: 20, TotalUniques: 7},
		{Stars: 3, Forks: 4, TotalClones: 5, CloneUniques: 2, TotalViews: 8, TotalUniques: 3},
	})
	if stars != 4 || forks != 6 {
		t.Fatalf("stars/forks = %d/%d, want 4/6", stars, forks)
	}
	if clones != 15 || cloneUniques != 6 {
		t.Fatalf("clones/cloneUniques = %d/%d, want 15/6", clones, cloneUniques)
	}
	if views != 28 || viewUniques != 10 {
		t.Fatalf("views/viewUniques = %d/%d, want 28/10", views, viewUniques)
	}
}
