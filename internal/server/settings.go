package server

import (
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hrodrig/gghstats/internal/i18n"
	"github.com/hrodrig/gghstats/internal/version"
)

// SettingsSnapshot contains only safe, redacted values suitable for rendering
// on the unauthenticated dashboard. Never add tokens, paths, raw filters, or
// arbitrary proxy/alert configuration here.
type SettingsSnapshot struct {
	RuntimeMode           string
	Host                  string
	Port                  string
	DatabaseReady         bool
	GithubTokenConfigured bool
	APITokenConfigured    bool
	FilterAll             bool
	IncludePrivate        bool
	ReportPrivate         bool
	SyncInterval          string
	SyncOnStartup         bool
	SyncWorkers           int
	SyncAvailable         bool
	BadgePublic           bool
	MetricsEnabled        bool
	RateLimitEnabled      bool
	APIOnly               bool
	CompactNumbers        bool
	DefaultLocale         string
	EnabledLocales        []string
	AdminProtection       bool
	Editable              bool
	Persisted             bool
	LocalOnly             bool
}

// EditableSettings is deliberately limited to presentation preferences. It
// must not grow to include credentials, paths, networking, collection scope,
// privacy policy, or sync policy.
type EditableSettings struct {
	DefaultLocale  string
	CompactNumbers bool
}

func effectiveEditableSettings(cfg Config) (EditableSettings, bool) {
	if cfg.SettingsManager != nil {
		return cfg.SettingsManager.Snapshot(), true
	}
	return EditableSettings{
		DefaultLocale:  cfg.DefaultLocale,
		CompactNumbers: cfg.CompactNumbers,
	}, false
}

// SettingsUpdate is the allow-listed JSON shape accepted by the settings API.
// Pointer fields let an update change one preference without resetting the
// other one.
type SettingsUpdate struct {
	DefaultLocale  *string `json:"default_locale"`
	CompactNumbers *bool   `json:"compact_numbers"`
}

type persistedSettings struct {
	DefaultLocale  string `json:"default_locale,omitempty"`
	CompactNumbers *bool  `json:"compact_numbers,omitempty"`
}

// SettingsManager owns the small, safe, persisted overlay for UI preferences.
// The file contains no secrets and is never rendered directly.
type SettingsManager struct {
	mu             sync.RWMutex
	path           string
	enabledLocales map[string]struct{}
	values         EditableSettings
}

// NewSettingsManager loads the persisted presentation overlay, if present.
// An empty path is supported for tests and means in-memory settings only.
func NewSettingsManager(path string, defaults EditableSettings, enabledLocales []string) (*SettingsManager, error) {
	enabled := make(map[string]struct{}, len(enabledLocales))
	for _, locale := range enabledLocales {
		locale = i18n.NormalizeLocale(locale)
		if locale != "" {
			enabled[locale] = struct{}{}
		}
	}
	if len(enabled) == 0 {
		enabled[i18n.DefaultLocale] = struct{}{}
	}
	defaults.DefaultLocale = normalizeEditableLocale(defaults.DefaultLocale, enabled)

	m := &SettingsManager{
		path:           path,
		enabledLocales: enabled,
		values:         defaults,
	}
	if path == "" {
		return m, nil
	}

	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return m, nil
	}
	if err != nil {
		return nil, err
	}
	var saved persistedSettings
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&saved); err != nil {
		return nil, err
	}
	if saved.DefaultLocale != "" {
		m.values.DefaultLocale = normalizeEditableLocale(saved.DefaultLocale, enabled)
	}
	if saved.CompactNumbers != nil {
		m.values.CompactNumbers = *saved.CompactNumbers
	}
	return m, nil
}

func normalizeEditableLocale(locale string, enabled map[string]struct{}) string {
	locale = i18n.NormalizeLocale(locale)
	if _, ok := enabled[locale]; ok {
		return locale
	}
	if _, ok := enabled[i18n.DefaultLocale]; ok {
		return i18n.DefaultLocale
	}
	for candidate := range enabled {
		return candidate
	}
	return i18n.DefaultLocale
}

func (m *SettingsManager) Snapshot() EditableSettings {
	if m == nil {
		return EditableSettings{}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.values
}

func (m *SettingsManager) Persisted() bool {
	return m != nil && m.path != ""
}

// Update validates and persists an allow-listed update before publishing it.
func (m *SettingsManager) Update(update SettingsUpdate) (EditableSettings, error) {
	if m == nil {
		return EditableSettings{}, errors.New("settings manager is unavailable")
	}
	if update.DefaultLocale == nil && update.CompactNumbers == nil {
		return EditableSettings{}, errors.New("no editable settings supplied")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	next := m.values
	if update.DefaultLocale != nil {
		locale := i18n.NormalizeLocale(*update.DefaultLocale)
		if _, ok := m.enabledLocales[locale]; !ok {
			return EditableSettings{}, errors.New("default_locale is not enabled")
		}
		next.DefaultLocale = locale
	}
	if update.CompactNumbers != nil {
		next.CompactNumbers = *update.CompactNumbers
	}
	if err := m.saveLocked(next); err != nil {
		return EditableSettings{}, err
	}
	m.values = next
	return next, nil
}

func (m *SettingsManager) saveLocked(values EditableSettings) error {
	if m.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(m.path), 0o700); err != nil {
		return err
	}
	compact := values.CompactNumbers
	saved := persistedSettings{DefaultLocale: values.DefaultLocale, CompactNumbers: &compact}
	raw, err := json.MarshalIndent(saved, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(m.path, raw, 0o600)
}

type settingsPageData struct {
	localeBinder
	Settings SettingsSnapshot
}

func handleSettingsPage(cfg Config, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lb := bindPageLocale(r, cfg)
		settings := cfg.Settings
		if cfg.SettingsManager != nil {
			editable := cfg.SettingsManager.Snapshot()
			settings.DefaultLocale = editable.DefaultLocale
			settings.CompactNumbers = editable.CompactNumbers
			settings.Persisted = cfg.SettingsManager.Persisted()
		}
		settings.LocalOnly = cfg.LocalOnlySettings && cfg.APIToken == ""
		settings.AdminProtection = cfg.APIToken != "" || settings.LocalOnly
		settings.Editable = settings.AdminProtection && cfg.SettingsManager != nil
		content := executeTemplate(tmpl, "settings", settingsPageData{
			localeBinder: lb,
			Settings:     settings,
		})
		renderLayout(w, r, tmpl, cfg, layoutData{
			Title:       lb.T("settings.title"),
			PageID:      "settings",
			Version:     version.Version,
			Breadcrumbs: []breadcrumb{{Label: lb.T("nav.home"), URL: "/"}, {Label: lb.T("settings.title"), URL: ""}},
			Content:     content,
		})
	}
}

func settingsMiddleware(cfg Config, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.APIToken != "" {
			if r.Header.Get("x-api-token") != cfg.APIToken {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			next(w, r)
			return
		}
		if cfg.LocalOnlySettings {
			next(w, r)
			return
		}
		http.NotFound(w, r)
	}
}

func handleSettingsUpdate(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.SettingsManager == nil {
			http.Error(w, `{"error":"settings_unavailable"}`, http.StatusServiceUnavailable)
			return
		}
		defer r.Body.Close()
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
		dec.DisallowUnknownFields()
		var update SettingsUpdate
		if err := dec.Decode(&update); err != nil {
			http.Error(w, `{"error":"invalid_settings"}`, http.StatusBadRequest)
			return
		}
		var extra interface{}
		if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
			http.Error(w, `{"error":"invalid_settings"}`, http.StatusBadRequest)
			return
		}
		values, err := cfg.SettingsManager.Update(update)
		if err != nil {
			http.Error(w, `{"error":"invalid_settings"}`, http.StatusBadRequest)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"default_locale":  values.DefaultLocale,
			"compact_numbers": values.CompactNumbers,
		})
	}
}
