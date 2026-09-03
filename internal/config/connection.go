// Package config holds the shared Glazed "connection" section used by every
// agentforum command that needs a database. Centralizing it here keeps the
// env-var/flag contract (design doc §4) in one place: AGENTFORUM_DB,
// AGENTFORUM_URL, AGENTFORUM_TOKEN, and AGENTFORUM_BACKEND.
package config

import (
	"os"
	"path/filepath"

	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
)

// SectionSlug is the Glazed section name commands decode with
// vals.DecodeSectionInto(SectionSlug, &cfg).
const SectionSlug = "connection"

// Connection is the decoded connection settings for a command. Field names map
// to cobra flags (--db, --url, --token, --backend) and, because the section has
// no prefix and the root uses AppName "agentforum", to env vars
// AGENTFORUM_DB, AGENTFORUM_URL, AGENTFORUM_TOKEN, AGENTFORUM_BACKEND.
type Connection struct {
	DBPath  string `glazed:"db"`
	URL     string `glazed:"url"`
	Token   string `glazed:"token"`
	Backend string `glazed:"backend"`
}

// DefaultDBPath resolves the database path when neither --db nor AGENTFORUM_DB
// is supplied: $AGENTFORUM_DB → --db → $XDG_DATA_HOME/agentforum/agentforum.db
// → ~/.local/share/agentforum/agentforum.db.
func DefaultDBPath() string {
	if v := os.Getenv("AGENTFORUM_DB"); v != "" {
		return v
	}
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return "agentforum.db"
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "agentforum", "agentforum.db")
}

// ResolveDBPath returns the explicit path if non-empty, else DefaultDBPath.
func ResolveDBPath(explicit string) string {
	if explicit != "" {
		return explicit
	}
	return DefaultDBPath()
}

// ConnectionSection returns the reusable Glazed section. It has no prefix so
// flags are --db/--url/--token/--backend and env vars are AGENTFORUM_*.
func ConnectionSection() (*schema.SectionImpl, error) {
	return schema.NewSection(
		SectionSlug, "Connection",
		schema.WithFields(
			fields.New("db", fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Path to the agentforum SQLite database (env AGENTFORUM_DB)")),
			fields.New("url", fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Base URL of a remote agentforum server (env AGENTFORUM_URL); unused in CLI-only/local mode")),
			fields.New("token", fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Bearer token for the authenticated agent (env AGENTFORUM_TOKEN)")),
			fields.New("backend", fields.TypeString,
				fields.WithDefault("local"),
				fields.WithChoices("local", "remote"),
				fields.WithHelp("Backend to use (env AGENTFORUM_BACKEND); remote is a future phase")),
		),
	)
}
