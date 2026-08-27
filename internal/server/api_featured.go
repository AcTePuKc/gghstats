package server

import (
	"net/http"

	"github.com/hrodrig/gghstats/internal/store"
)

// handleAPIFeatured serves GET /api/v1/featured — dogfood for the Featured HTML
// showcase (metadata only; no traffic). Query params mirror /featured.
func handleAPIFeatured(cfg Config) http.HandlerFunc {
	db := cfg.Store
	return func(w http.ResponseWriter, r *http.Request) {
		sort, dir, query, page, perPage := parseFeaturedQueryParams(r)
		items, total, err := db.FilterReportFeatured(cfg.ReportVisibility, query, sort, dir, page, perPage)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "database error")
			return
		}
		if items == nil {
			items = []store.Featured{}
		}
		totalPages := indexTotalPages(total, perPage)
		page = clampIndexPage(page, totalPages)
		writeAPIJSON(w, r, cfg, map[string]interface{}{
			"total_count": total,
			"items":       items,
			"sort":        sort,
			"dir":         dir,
			"q":           query,
			"page":        page,
			"per_page":    perPage,
			"total_pages": totalPages,
		})
	}
}
