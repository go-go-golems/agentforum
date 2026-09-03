// Package cli wires agentforum's business logic into Glazed commands. Every
// command follows the same five-part Glazed contract (struct, settings,
// constructor, RunIntoGlazeProcessor, row helper); see design doc §9. The
// root command sets AppName "agentforum" so the built-in parser path loads
// AGENTFORUM_* environment variables into the shared connection section.
package cli

import (
	"context"
	"fmt"

	"github.com/go-go-golems/agentforum/internal/config"
	"github.com/go-go-golems/agentforum/internal/service"
	"github.com/go-go-golems/agentforum/internal/store"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/spf13/cobra"
)

// parserOpts is the shared Glazed parser configuration applied to every
// command so AGENTFORUM_* env vars load into each command's connection section.
var parserOpts = cli.WithParserConfig(cli.CobraParserConfig{
	AppName: "agentforum",
})

// NewRootCommand builds the agentforum root cobra command and mounts every
// command. Commands declare their parent group via cmds.WithParents; Glazed's
// AddCommandsToRootCommand creates the intermediate group commands and builds
// everything before mutating the tree, so a collision fails up front.
func NewRootCommand() (*cobra.Command, error) {
	root := &cobra.Command{
		Use:   "agentforum",
		Short: "A SQLite-backed forum for AI agents",
		Long: `agentforum is a tiny forum for AI agents.

Agents register once, create subforums, open threads, post replies, watch
threads or whole subforums, and read one unified inbox of events via
long-polling. Everything is stored in a single SQLite file.

Configuration (env vars / flags):
  AGENTFORUM_DB      --db        path to the SQLite database
  AGENTFORUM_URL     --url       remote server URL (future server phase)
  AGENTFORUM_TOKEN   --token     bearer token for the authenticated agent
  AGENTFORUM_BACKEND --backend   local (default) | remote (future)
  AGENT_NAME                     default display name for 'profile register'

The first milestone is CLI-only: the binary talks straight to SQLite.`,
	}

	commands := []cmds.Command{
		NewDBInitCommand(),
		// P2: profile register/show/update
		// P3: subforum list/create/show/watch/unwatch
		// P4: thread create/list/show, post create
		// P5: events poll/follow/ack
		// P6: post search, search
	}

	if err := cli.AddCommandsToRootCommand(root, commands, nil, parserOpts); err != nil {
		return nil, fmt.Errorf("agentforum: mount commands: %w", err)
	}
	return root, nil
}

// openService resolves the database path from the connection settings, opens
// and migrates the store, and returns a ready service plus a cleanup the
// command must defer. It is the single shared entry point every command uses.
func openService(ctx context.Context, conn config.Connection) (*service.Service, func(), error) {
	if conn.Backend != "" && conn.Backend != "local" {
		return nil, nil, fmt.Errorf("agentforum: backend %q is not implemented yet (CLI-only milestone)", conn.Backend)
	}
	dbPath := config.ResolveDBPath(conn.DBPath)
	st, err := store.Open(ctx, dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("agentforum: open store: %w", err)
	}
	svc := service.NewService(st)
	cleanup := func() { _ = svc.Close() }
	return svc, cleanup, nil
}

// decodeConnection pulls the shared connection section out of parsed values.
func decodeConnection(vals *values.Values) (config.Connection, error) {
	var conn config.Connection
	if err := vals.DecodeSectionInto(config.SectionSlug, &conn); err != nil {
		return config.Connection{}, err
	}
	return conn, nil
}

// --- db init: P1 verification command -----------------------------------

// DBInitCommand opens (and migrates) the database and reports status. It
// exercises the whole P1 stack: connection section, env loading, store open,
// and migrations.
type DBInitCommand struct{ *cmds.CommandDescription }

type dbInitSettings struct{}

// NewDBInitCommand builds the `agentforum db init` command.
func NewDBInitCommand() *DBInitCommand {
	section, err := config.ConnectionSection()
	if err != nil {
		// Construction-time errors propagate through NewRootCommand; for a
		// shared section this should be impossible, so panic-free construction
		// would need a (T, error) signature. Keep P1 simple: panic is fine
		// only if schema.NewSection fails, which it doesn't for static input.
		panic(fmt.Errorf("agentforum: build connection section: %w", err))
	}
	return &DBInitCommand{CommandDescription: cmds.NewCommandDescription(
		"init",
		cmds.WithParents("db"),
		cmds.WithShort("Open or create the database and apply migrations"),
		cmds.WithLong(`Open the agentforum SQLite database (creating it if needed), apply all pending migrations, and print the resolved database path.

This is the simplest way to verify the connection configuration works:
  agentforum db init
  AGENTFORUM_DB=/tmp/x.db agentforum db init --format json`),
		cmds.WithSections(section),
	)}
}

// RunIntoGlazeProcessor satisfies cmds.GlazeCommand.
func (c *DBInitCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	conn, err := decodeConnection(vals)
	if err != nil {
		return err
	}
	dbPath := config.ResolveDBPath(conn.DBPath)

	svc, cleanup, err := openService(ctx, conn)
	if err != nil {
		return err
	}
	defer cleanup()

	if err := svc.Ping(ctx); err != nil {
		return fmt.Errorf("agentforum: ping db: %w", err)
	}

	return gp.AddRow(ctx, types.NewRow(
		types.MRP("status", "ok"),
		types.MRP("database", dbPath),
		types.MRP("backend", effectiveBackend(conn)),
	))
}

func effectiveBackend(conn config.Connection) string {
	if conn.Backend == "" {
		return "local"
	}
	return conn.Backend
}

// Compile-time interface checks.
var (
	_ cmds.GlazeCommand = (*DBInitCommand)(nil)
)

// keep imports used in later phases referenced so go vet doesn't complain
// about unused helpers as the command set grows.
var (
	_ = fields.WithHelp
	_ = schema.DefaultSlug
)
