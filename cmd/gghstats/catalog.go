package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/hrodrig/gghstats/internal/store"
)

// catalog.go — operator catalog CLI: gghstats repo / gghstats featured
// (v1.1.0, SPEC "repo pins CLI + Featured showcase"). The dashboard shows;
// the CLI stewards. No GitHub token required (add/rm/ls are local SQLite ops).

var nameRe = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

// validCatalogName trims and validates an OWNER/REPO name. Rejects "*", empty
// owner/repo, and FILTER-style syntax.
func validCatalogName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" || name == "*" {
		return "", fmt.Errorf("invalid repo name %q: expected OWNER/REPO", raw)
	}
	if !nameRe.MatchString(name) {
		return "", fmt.Errorf("invalid repo name %q: expected OWNER/REPO (letters, digits, '.', '_', '-')", raw)
	}
	return name, nil
}

// runRepo is the "gghstats repo" command family.
func runRepo(args []string) error {
	if len(args) > 0 && args[0] == "report" {
		return runRepoReport(args[1:])
	}
	return runCatalog("repo", "", args)
}

// runRepoReport is the supported operator interface for changing the reporting
// boundary; it never changes collection or deletes stored data.
func runRepoReport(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("repo report: expected ls|set")
	}
	_, dbPath, rest, err := catalogFlagSet("repo report", args[1:])
	if err != nil {
		return err
	}
	db, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	switch args[0] {
	case "ls":
		if len(rest) != 0 {
			return fmt.Errorf("repo report ls: unexpected argument")
		}
		states, err := db.ListRepoReportStates()
		if err != nil {
			return err
		}
		for _, state := range states {
			fmt.Printf("%s\t%s\t%s\n", state.Name, state.GitHubVisibility, state.ReportPolicy)
		}
		return nil
	case "set":
		if len(rest) != 2 {
			return fmt.Errorf("repo report set: expected OWNER/REPO inherit|include|exclude")
		}
		name, err := validCatalogName(rest[0])
		if err != nil {
			return err
		}
		ok, err := db.SetRepoReportPolicy(name, rest[1])
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("repository not found")
		}
		fmt.Printf("report policy for %s set to %s\n", name, rest[1])
		return nil
	default:
		return fmt.Errorf("repo report: unknown subcommand %q (ls|set)", args[0])
	}
}

// runFeatured is the "gghstats featured" command family.
func runFeatured(args []string) error { return runCatalog("featured", "", args) }

func runCatalog(kind, sub string, args []string) error {
	if sub == "" {
		if len(args) == 0 {
			return fmt.Errorf("%s: expected subcommand add|rm|ls", kind)
		}
		sub = args[0]
		args = args[1:]
	}
	switch sub {
	case "add", "rm":
		return runCatalogMutate(kind, sub, args, os.Stderr)
	case "ls":
		return runCatalogList(kind, args, os.Stdout)
	case "--help", "-h", "help":
		fmt.Fprintln(os.Stderr, catalogUsage(kind))
		return nil
	default:
		return fmt.Errorf("%s: unknown subcommand %q (add|rm|ls)", kind, sub)
	}
}

func catalogUsage(kind string) string {
	if kind == "repo" {
		return `gghstats repo <add|rm|ls|report> [flags]

  gghstats repo report ls
  gghstats repo report set OWNER/REPO inherit|include|exclude

Report policy affects reporting only; collection and SQLite storage continue.`
	}
	return fmt.Sprintf(`gghstats %s <add|rm|ls> [flags]

  %s add OWNER/REPO   Pin/showcase a repo (idempotent)
  %s rm  OWNER/REPO   Remove a repo (error if missing)
  %s ls               List entries

Flags:
  --db PATH   SQLite database path (GGHSTATS_DB or platform default)`, kind, kind, kind, kind)
}

func catalogFlagSet(kind string, args []string) (*flag.FlagSet, string, []string, error) {
	fs := flag.NewFlagSet(kind, flag.ContinueOnError)
	dbPath := envOr("GGHSTATS_DB", defaultDBPath())
	fs.StringVar(&dbPath, "db", dbPath, "SQLite database path")
	if err := fs.Parse(args); err != nil {
		return nil, "", nil, err
	}
	return fs, dbPath, fs.Args(), nil
}

func runCatalogMutate(kind, sub string, args []string, w io.Writer) error {
	_, dbPath, rest, err := catalogFlagSet(kind, args)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return fmt.Errorf("%s %s: expected exactly one OWNER/REPO", kind, sub)
	}
	name, err := validCatalogName(rest[0])
	if err != nil {
		return err
	}

	db, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	if sub == "add" {
		if kind == "repo" {
			if err := db.AddPin(name); err != nil {
				return err
			}
			fmt.Fprintf(w, "pinned %s\n", name)
			return nil
		}
		if err := db.AddFeatured(name); err != nil {
			return err
		}
		fmt.Fprintf(w, "featured %s\n", name)
		return nil
	}

	// rm — missing row is an error (exit 1) per SPEC.
	if kind == "repo" {
		rem, err := db.RemovePin(name)
		if err != nil {
			return err
		}
		if !rem {
			return fmt.Errorf("repo %q is not pinned", name)
		}
		fmt.Fprintf(w, "unpinned %s\n", name)
		return nil
	}
	rem, err := db.RemoveFeatured(name)
	if err != nil {
		return err
	}
	if !rem {
		return fmt.Errorf("featured %q not found", name)
	}
	fmt.Fprintf(w, "unfeatured %s\n", name)
	return nil
}

// runCatalogList prints catalog entries to w.
func runCatalogList(kind string, args []string, w io.Writer) error {
	_, dbPath, rest, err := catalogFlagSet(kind, args)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return fmt.Errorf("%s ls: unexpected argument %q", kind, rest[0])
	}

	db, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	switch kind {
	case "repo":
		pins, err := db.ListPins()
		if err != nil {
			return err
		}
		for _, p := range pins {
			fmt.Fprintln(w, p)
		}
		return nil
	case "featured":
		f, err := db.ListFeatured()
		if err != nil {
			return err
		}
		for _, e := range f {
			fmt.Fprintln(w, e.Name)
		}
		return nil
	}
	return fmt.Errorf("unknown catalog kind %q", kind)
}
