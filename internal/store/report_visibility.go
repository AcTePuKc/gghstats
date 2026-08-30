package store

import "database/sql"

// ReportVisibility configures which inherited GitHub visibility classes may be
// shown on reporting surfaces. Unknown is never inherited: it must be refreshed
// from GitHub or explicitly included by an operator.
type ReportVisibility struct {
	IncludePrivate bool
}

const (
	VisibilityPublic  = "public"
	VisibilityPrivate = "private"
	VisibilityUnknown = "unknown"
	ReportInherit     = "inherit"
	ReportInclude     = "include"
	ReportExclude     = "exclude"
)

// NormalizeGitHubVisibility turns the GitHub REST fields into our persisted
// representation. GitHub's visibility field is preferred; private is retained
// for older API responses.
func NormalizeGitHubVisibility(visibility string, private bool) string {
	switch visibility {
	case VisibilityPublic, VisibilityPrivate:
		return visibility
	}
	if visibility != "" {
		// GitHub Enterprise can return values such as "internal". Treat an
		// unmodelled class as unknown rather than accidentally making it public.
		return VisibilityUnknown
	}
	if private {
		return VisibilityPrivate
	}
	return VisibilityPublic
}

func validReportPolicy(policy string) bool {
	return policy == ReportInherit || policy == ReportInclude || policy == ReportExclude
}

// SetRepoReportPolicy changes only the reporting decision; collection and local
// persistence remain unaffected. It returns false when the repository is absent.
func (s *Store) SetRepoReportPolicy(name, policy string) (bool, error) {
	if !validReportPolicy(policy) {
		return false, ErrInvalidReportPolicy(policy)
	}
	res, err := s.db.Exec(`UPDATE repos SET report_policy=? WHERE name=?`, policy, name)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// RepoReportState is the operator-facing stored decision input.
type RepoReportState struct {
	Name             string `json:"name"`
	GitHubVisibility string `json:"github_visibility"`
	ReportPolicy     string `json:"report_policy"`
}

func (s *Store) ListRepoReportStates() ([]RepoReportState, error) {
	rows, err := s.db.Query(`SELECT name, github_visibility, report_policy FROM repos ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RepoReportState
	for rows.Next() {
		var state RepoReportState
		if err := rows.Scan(&state.Name, &state.GitHubVisibility, &state.ReportPolicy); err != nil {
			return nil, err
		}
		out = append(out, state)
	}
	return out, rows.Err()
}

type invalidReportPolicy string

func (e invalidReportPolicy) Error() string      { return "invalid report policy " + string(e) }
func ErrInvalidReportPolicy(policy string) error { return invalidReportPolicy(policy) }

// UpsertRepoWithVisibility atomically updates repository metadata and its GitHub
// visibility, avoiding a window where a private repository could be reportable.
func (s *Store) UpsertRepoWithVisibility(name, description string, stars, forks, watchers, issues, prs int, fork, archived bool, parentFullName, visibility string) error {
	if visibility != VisibilityPublic && visibility != VisibilityPrivate {
		visibility = VisibilityUnknown
	}
	if !fork {
		parentFullName = ""
	}
	_, err := s.db.Exec(`INSERT INTO repos (name, description, stars, forks, watchers, issues, prs, fork, archived, parent_full_name, github_visibility, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT (name) DO UPDATE SET
		description=excluded.description, stars=MAX(repos.stars, excluded.stars), forks=MAX(repos.forks, excluded.forks), watchers=MAX(repos.watchers, excluded.watchers),
		issues=excluded.issues, prs=excluded.prs, fork=excluded.fork, archived=excluded.archived, parent_full_name=excluded.parent_full_name,
		github_visibility=excluded.github_visibility, hidden=0, updated_at=excluded.updated_at`,
		name, description, stars, forks, watchers, issues, prs, boolToInt(fork), boolToInt(archived), parentFullName, visibility)
	return err
}

func (s *Store) reportVisible(name string, scope ReportVisibility) (bool, error) {
	var visibility, policy string
	err := s.db.QueryRow(`SELECT github_visibility, report_policy FROM repos WHERE name=? AND hidden=0`, name).Scan(&visibility, &policy)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if policy == ReportExclude {
		return false, nil
	}
	if policy == ReportInclude {
		return true, nil
	}
	return visibility == VisibilityPublic || (visibility == VisibilityPrivate && scope.IncludePrivate), nil
}

// ReportRepoByName returns nil for non-reportable repositories, intentionally
// making excluded direct requests indistinguishable from ordinary not-found.
func (s *Store) ReportRepoByName(scope ReportVisibility, name string) (*RepoSummary, error) {
	ok, err := s.reportVisible(name, scope)
	if err != nil || !ok {
		return nil, err
	}
	return s.RepoByName(name)
}

func (s *Store) ListReportRepos(scope ReportVisibility, sort, direction string) ([]RepoSummary, error) {
	repos, err := s.ListRepos(sort, direction)
	if err != nil {
		return nil, err
	}
	out := make([]RepoSummary, 0, len(repos))
	for _, repo := range repos {
		ok, err := s.reportVisible(repo.Name, scope)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, repo)
		}
	}
	return out, nil
}

func (s *Store) ReportRepoCount(scope ReportVisibility) (int, error) {
	repos, err := s.ListReportRepos(scope, "name", "asc")
	return len(repos), err
}

func (s *Store) SumReportClonesAll(scope ReportVisibility) (int, error) {
	repos, err := s.ListReportRepos(scope, "name", "asc")
	if err != nil {
		return 0, err
	}
	total := 0
	for _, repo := range repos {
		total += repo.TotalClones
	}
	return total, nil
}
func (s *Store) SumReportViewsAll(scope ReportVisibility) (int, error) {
	repos, err := s.ListReportRepos(scope, "name", "asc")
	if err != nil {
		return 0, err
	}
	total := 0
	for _, repo := range repos {
		total += repo.TotalViews
	}
	return total, nil
}
