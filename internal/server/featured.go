package server

import (
	"html/template"
	"net/http"

	"github.com/hrodrig/gghstats/internal/store"
	"github.com/hrodrig/gghstats/internal/version"
)

// featuredPageData is the payload for the /featured showcase template.
type featuredPageData struct {
	localeBinder
	Featured []store.Featured
}

func handleFeaturedPage(cfg Config, db *store.Store, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/featured" {
			lb := bindPageLocale(r, cfg)
			writeBrutalistNotFound(w, r, tmpl, cfg, lb.T("not_found.title"), lb.T("not_found.heading"), r.URL.Path, lb.T("not_found.detail"))
			return
		}

		lb := bindPageLocale(r, cfg)
		featured, err := db.ListFeatured()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data := featuredPageData{
			localeBinder: lb,
			Featured:     featured,
		}

		content := executeTemplate(tmpl, "featured", data)
		renderLayout(w, r, tmpl, cfg, layoutData{
			Title:   lb.T("featured.title"),
			PageID:  "featured",
			Version: version.Version,
			Breadcrumbs: []breadcrumb{
				{Label: lb.T("nav.home"), URL: "/"},
				{Label: lb.T("featured.title"), URL: ""},
			},
			Content: content,
		})
	}
}
