package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hrodrig/gghstats/internal/store"
)

const (
	defaultTrafficDays = 30
	maxTrafficDays     = 3660
)

type repoTrafficResponse struct {
	Name            string           `json:"name"`
	Days            int              `json:"days"`
	From            string           `json:"from"`
	To              string           `json:"to"`
	Clones          []store.DayRow   `json:"clones"`
	Views           []store.DayRow   `json:"views"`
	ClonesFreshness trafficFreshness `json:"clones_freshness"`
	ViewsFreshness  trafficFreshness `json:"views_freshness"`
}

// repoTrafficDenseResponse is chart-aligned: every UTC day in [from,to] appears;
// omitted GitHub days use null count/uniques (same as detail charts).
type repoTrafficDenseResponse struct {
	Name            string              `json:"name"`
	Days            int                 `json:"days"`
	From            string              `json:"from"`
	To              string              `json:"to"`
	Dense           bool                `json:"dense"`
	Clones          []chartTrafficPoint `json:"clones"`
	Views           []chartTrafficPoint `json:"views"`
	ClonesFreshness trafficFreshness    `json:"clones_freshness"`
	ViewsFreshness  trafficFreshness    `json:"views_freshness"`
}

func repoFullNameFromRequest(r *http.Request) string {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	if owner == "" || repo == "" {
		return ""
	}
	return owner + "/" + repo
}

// parseTrafficDays interprets the days query parameter (default 30). days=0 means all stored history (UTC).
func parseTrafficDays(raw string) (days int, err error) {
	if raw == "" {
		return defaultTrafficDays, nil
	}
	days, err = strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid days")
	}
	if days < 0 || days > maxTrafficDays {
		return 0, fmt.Errorf("invalid days")
	}
	return days, nil
}

func trafficDateRangeUTC(days int, extentMin string, extentOk bool) (from, to string, err error) {
	to = time.Now().UTC().Format("2006-01-02")
	if days == 0 {
		if !extentOk {
			return to, to, nil
		}
		return extentMin, to, nil
	}
	from = time.Now().UTC().AddDate(0, 0, -(days - 1)).Format("2006-01-02")
	return from, to, nil
}

func trafficJSONFilename(fullName string, now time.Time) string {
	safe := strings.ReplaceAll(fullName, "/", "-")
	return fmt.Sprintf("gghstats-%s-traffic-%s.json", safe, now.UTC().Format("20060102"))
}

type repoTrafficWindow struct {
	Name            string
	Days            int
	From            string
	To              string
	Clones          []store.DayRow
	Views           []store.DayRow
	ClonesFreshness trafficFreshness
	ViewsFreshness  trafficFreshness
}

func loadRepoTrafficWindow(db *store.Store, scope store.ReportVisibility, fullName string, days int, now time.Time) (*repoTrafficWindow, int, error) {
	summary, err := db.ReportRepoByName(scope, fullName)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if summary == nil {
		return nil, http.StatusNotFound, nil
	}
	extentMin, _, extentOk, err := db.TrafficDateExtentForRepo(fullName)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	from, to, err := trafficDateRangeUTC(days, extentMin, extentOk)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	clones, err := db.ClonesByRange(fullName, from, to)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	views, err := db.ViewsByRange(fullName, from, to)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	clones, err = trafficRowsWithLatestCoverage(db, fullName, "clones", clones)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	views, err = trafficRowsWithLatestCoverage(db, fullName, "views", views)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	viewsFreshness, clonesFreshness, err := repoTrafficFreshness(db, fullName, now.UTC())
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if clones == nil {
		clones = []store.DayRow{}
	}
	if views == nil {
		views = []store.DayRow{}
	}
	return &repoTrafficWindow{
		Name:            fullName,
		Days:            days,
		From:            from,
		To:              to,
		Clones:          clones,
		Views:           views,
		ClonesFreshness: clonesFreshness,
		ViewsFreshness:  viewsFreshness,
	}, http.StatusOK, nil
}

func denseRepoTrafficResponse(win *repoTrafficWindow) *repoTrafficDenseResponse {
	clones := denseTrafficChart(win.Clones, win.From, win.To)
	views := denseTrafficChart(win.Views, win.From, win.To)
	if clones == nil {
		clones = []chartTrafficPoint{}
	}
	if views == nil {
		views = []chartTrafficPoint{}
	}
	return &repoTrafficDenseResponse{
		Name:            win.Name,
		Days:            win.Days,
		From:            win.From,
		To:              win.To,
		Dense:           true,
		Clones:          clones,
		Views:           views,
		ClonesFreshness: win.ClonesFreshness,
		ViewsFreshness:  win.ViewsFreshness,
	}
}

func writeRepoTrafficJSON(w http.ResponseWriter, r *http.Request, cfg Config, payload any, downloadName string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if downloadName != "" {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, downloadName))
	}
	setAPICORS(w, r, cfg.CORSOrigins)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "encode error")
	}
}

func handleAPIRepoTraffic(cfg Config) http.HandlerFunc {
	db := cfg.Store
	return func(w http.ResponseWriter, r *http.Request) {
		fullName := repoFullNameFromRequest(r)
		if fullName == "" {
			writeJSONError(w, http.StatusBadRequest, "invalid path")
			return
		}
		days, err := parseTrafficDays(strings.TrimSpace(r.URL.Query().Get("days")))
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		dense := strings.TrimSpace(r.URL.Query().Get("dense")) == "1"
		download := strings.TrimSpace(r.URL.Query().Get("download")) == "1"
		if download {
			dense = true
		}

		win, code, err := loadRepoTrafficWindow(db, cfg.ReportVisibility, fullName, days, time.Now().UTC())
		if code == http.StatusNotFound {
			writeJSONNotFound(w)
			return
		}
		if err != nil {
			if code == http.StatusBadRequest {
				writeJSONError(w, code, err.Error())
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		filename := ""
		if download {
			filename = trafficJSONFilename(fullName, time.Now().UTC())
		}
		if dense {
			writeRepoTrafficJSON(w, r, cfg, denseRepoTrafficResponse(win), filename)
			return
		}
		writeRepoTrafficJSON(w, r, cfg, &repoTrafficResponse{
			Name:            win.Name,
			Days:            win.Days,
			From:            win.From,
			To:              win.To,
			Clones:          win.Clones,
			Views:           win.Views,
			ClonesFreshness: win.ClonesFreshness,
			ViewsFreshness:  win.ViewsFreshness,
		}, "")
	}
}

// handleRepoTrafficJSONExport serves chart-aligned traffic JSON as a file download
// on the HTML surface (report-scoped). Auth: optionalAPITokenMiddleware — require
// x-api-token only when GGHSTATS_API_TOKEN is set.
func handleRepoTrafficJSONExport(cfg Config) http.HandlerFunc {
	db := cfg.Store
	return func(w http.ResponseWriter, r *http.Request) {
		fullName := repoFullNameFromRequest(r)
		if fullName == "" {
			http.NotFound(w, r)
			return
		}
		days, err := parseTrafficDays(strings.TrimSpace(r.URL.Query().Get("days")))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		win, code, err := loadRepoTrafficWindow(db, cfg.ReportVisibility, fullName, days, time.Now().UTC())
		if code == http.StatusNotFound || win == nil {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			if code == http.StatusBadRequest {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		payload := denseRepoTrafficResponse(win)
		writeRepoTrafficJSON(w, r, cfg, payload, trafficJSONFilename(fullName, time.Now().UTC()))
	}
}

func writeJSONError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"error":%q}`, msg)
}
