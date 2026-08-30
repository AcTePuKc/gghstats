package store

import (
	"database/sql"
	"fmt"
	"strings"
)

// catalog.go — operator catalog: dashboard pins and the Featured showcase
// (v1.1.0, SPEC "repo pins CLI + Featured showcase").

// --- Pins ---

// AddPin idempotently records a dashboard pin.
func (s *Store) AddPin(name string) error {
	_, err := s.db.Exec(
		`INSERT INTO pins (name, created_at) VALUES (?, datetime('now'))
		 ON CONFLICT(name) DO NOTHING`,
		name,
	)
	return err
}

// RemovePin removes a pin and reports whether a row was actually removed.
func (s *Store) RemovePin(name string) (bool, error) {
	res, err := s.db.Exec(`DELETE FROM pins WHERE name=?`, name)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// ListPins returns pinned names in insertion order.
func (s *Store) ListPins() ([]string, error) {
	rows, err := s.db.Query(`SELECT name FROM pins ORDER BY rowid`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// Featured is one showcase entry as stored.
type Featured struct {
	Name                string `json:"name"`
	Sort                int    `json:"sort"`
	ParentFullName      string `json:"parent_full_name,omitempty"`
	UpstreamFullName    string `json:"upstream_full_name"`
	UpstreamDescription string `json:"upstream_description"`
	UpstreamStars       int    `json:"upstream_stars"`
	Fork                bool   `json:"fork"`
	MetaUpdatedAt       string `json:"meta_updated_at,omitempty"`
}

// AddFeatured idempotently records a showcase entry, assigning insertion order.
func (s *Store) AddFeatured(name string) error {
	_, err := s.db.Exec(
		`INSERT INTO featured (name, created_at, sort)
		 VALUES (?, datetime('now'), COALESCE((SELECT MAX(sort)+1 FROM featured), 0))
		 ON CONFLICT(name) DO NOTHING`,
		name,
	)
	return err
}

// RemoveFeatured removes a showcase entry and reports whether a row was actually removed.
func (s *Store) RemoveFeatured(name string) (bool, error) {
	res, err := s.db.Exec(`DELETE FROM featured WHERE name=?`, name)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

const featuredCols = `
	name, sort, parent_full_name, upstream_full_name,
	upstream_description, upstream_stars, fork, meta_updated_at`

const featuredSelect = `
	SELECT` + featuredCols + `
	FROM featured ORDER BY sort, name`

// ListFeatured returns showcase entries ordered by sort then name.
func (s *Store) ListFeatured() ([]Featured, error) {
	rows, err := s.db.Query(featuredSelect)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Featured
	for rows.Next() {
		var f Featured
		var fork int
		if err := rows.Scan(
			&f.Name, &f.Sort, &f.ParentFullName, &f.UpstreamFullName,
			&f.UpstreamDescription, &f.UpstreamStars, &fork, &f.MetaUpdatedAt,
		); err != nil {
			return nil, err
		}
		f.Fork = fork != 0
		out = append(out, f)
	}
	return out, rows.Err()
}

// FeaturedCount returns the number of showcase rows (used to hide nav when 0).
func (s *Store) FeaturedCount() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM featured`).Scan(&n)
	return n, err
}

// ReportFeaturedCount and FilterReportFeatured apply the same server-side
// visibility decision as every other report surface. A featured catalog entry
// without a reportable repository is intentionally invisible.
func (s *Store) ReportFeaturedCount(scope ReportVisibility) (int, error) {
	entries, _, err := s.FilterReportFeatured(scope, "", "sort", "asc", 1, 1_000_000)
	return len(entries), err
}

func (s *Store) FilterReportFeatured(scope ReportVisibility, query, sort, dir string, page, perPage int) ([]Featured, int, error) {
	entries, _, err := s.FilterFeatured(query, sort, dir, 1, 1_000_000)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Featured, 0, len(entries))
	for _, entry := range entries {
		// A catalog entry is not reportable until it has a collected repository
		// record. This keeps a manually entered name from bypassing the boundary.
		var exists int
		err := s.db.QueryRow(`SELECT 1 FROM repos WHERE name=?`, entry.Name).Scan(&exists)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return nil, 0, err
		}
		ok, err := s.reportVisible(entry.Name, scope)
		if err != nil {
			return nil, 0, err
		}
		if ok {
			out = append(out, entry)
		}
	}
	total := len(out)
	if page < 1 {
		page = 1
	}
	if perPage <= 0 {
		perPage = total
	}
	start := (page - 1) * perPage
	if start >= total {
		return []Featured{}, total, nil
	}
	end := start + perPage
	if end > total {
		end = total
	}
	return out[start:end], total, nil
}

// featuredSortCol maps an allowed sort key to a SQL column (defence in depth).
func featuredSortCol(sort string) string {
	switch sort {
	case "name":
		return "name"
	case "stars":
		return "upstream_stars"
	default:
		return "sort" // display/insertion order
	}
}

// FilterFeatured returns a filtered, sorted, paginated slice of showcase
// entries plus the total number of matching rows (before pagination).
// sort is one of "sort" (default display order), "name", or "stars" (unknown
// values fall back to "sort"); dir is "asc" or "desc" (anything else -> asc).
// query is a case-insensitive substring match against name and
// upstream_full_name. page is 1-based (callers must clamp; values < 1 are
// treated as 1). perPage is expected to be already clamped by the caller.
func (s *Store) FilterFeatured(query, sort, dir string, page, perPage int) ([]Featured, int, error) {
	col := featuredSortCol(sort)
	order := "asc"
	if dir == "desc" {
		order = "desc"
	}

	where := ""
	var args []any
	if q := strings.TrimSpace(query); q != "" {
		where = ` WHERE name LIKE ? OR upstream_full_name LIKE ?`
		args = append(args, "%"+q+"%", "%"+q+"%")
	}

	var total int
	countSQL := `SELECT COUNT(*) FROM featured` + where
	if err := s.db.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}

	selectSQL := `SELECT` + featuredCols + `
	FROM featured` + where +
		fmt.Sprintf(` ORDER BY %s %s, name LIMIT ? OFFSET ?`, col, order)
	args = append(args, perPage, (page-1)*perPage)

	rows, err := s.db.Query(selectSQL, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Featured
	for rows.Next() {
		var f Featured
		var fork int
		if err := rows.Scan(
			&f.Name, &f.Sort, &f.ParentFullName, &f.UpstreamFullName,
			&f.UpstreamDescription, &f.UpstreamStars, &fork, &f.MetaUpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		f.Fork = fork != 0
		out = append(out, f)
	}
	return out, total, rows.Err()
}

// UpsertFeaturedMeta stores refreshed metadata for a featured entry. The row
// must already exist (created via AddFeatured); metadata is updated in place.
// upstreamFullName is the parent when fork, else name.
func (s *Store) UpsertFeaturedMeta(name, parentFullName, upstreamFullName, description string, stars int, fork bool) error {
	res, err := s.db.Exec(
		`UPDATE featured SET
		   parent_full_name=?, upstream_full_name=?, upstream_description=?,
		   upstream_stars=?, fork=?, meta_updated_at=datetime('now')
		 WHERE name=?`,
		parentFullName, upstreamFullName, description, stars, boolToInt(fork), name,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("featured %q not found", name)
	}
	return nil
}

// GetFeaturedMeta returns stored metadata for a featured name (zero Featured + false if absent).
func (s *Store) GetFeaturedMeta(name string) (Featured, bool, error) {
	var f Featured
	var fork int
	err := s.db.QueryRow(
		`SELECT name, sort, parent_full_name, upstream_full_name,
		        upstream_description, upstream_stars, fork, meta_updated_at
		 FROM featured WHERE name=?`, name,
	).Scan(
		&f.Name, &f.Sort, &f.ParentFullName, &f.UpstreamFullName,
		&f.UpstreamDescription, &f.UpstreamStars, &fork, &f.MetaUpdatedAt,
	)
	if err == sql.ErrNoRows {
		return Featured{}, false, nil
	}
	if err != nil {
		return Featured{}, false, err
	}
	f.Fork = fork != 0
	return f, true, nil
}
