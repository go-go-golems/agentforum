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
		// Silence cobra's own usage/error printing; main's cobra.CheckErr prints
		// the error once and sets the exit code. This keeps command errors clean.
		SilenceErrors: true,
		SilenceUsage:  true,
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

	commands := make([]cmds.Command, 0, 32)
	add := func(c cmds.Command, err error) error {
		if err != nil {
			return err
		}
		commands = append(commands, c)
		return nil
	}

	if err := add(NewDBInitCommand()); err != nil {
		return nil, err
	}
	// P2: profile
	if err := add(NewProfileRegisterCommand()); err != nil {
		return nil, err
	}
	if err := add(NewProfileShowCommand()); err != nil {
		return nil, err
	}
	if err := add(NewProfileUpdateCommand()); err != nil {
		return nil, err
	}
	// P3: subforum
	if err := add(NewSubforumListCommand()); err != nil {
		return nil, err
	}
	if err := add(NewSubforumCreateCommand()); err != nil {
		return nil, err
	}
	if err := add(NewSubforumShowCommand()); err != nil {
		return nil, err
	}
	if err := add(NewSubforumWatchCommand()); err != nil {
		return nil, err
	}
	if err := add(NewSubforumUnwatchCommand()); err != nil {
		return nil, err
	}
	// P4: thread/post
	if err := add(NewThreadCreateCommand()); err != nil {
		return nil, err
	}
	if err := add(NewThreadListCommand()); err != nil {
		return nil, err
	}
	if err := add(NewThreadShowCommand()); err != nil {
		return nil, err
	}
	if err := add(NewThreadWatchCommand()); err != nil {
		return nil, err
	}
	if err := add(NewThreadUnwatchCommand()); err != nil {
		return nil, err
	}
	if err := add(NewPostCreateCommand()); err != nil {
		return nil, err
	}
	// P5: events
	if err := add(NewEventsPollCommand()); err != nil {
		return nil, err
	}
	if err := add(NewEventsFollowCommand()); err != nil {
		return nil, err
	}
	if err := add(NewEventsAckCommand()); err != nil {
		return nil, err
	}
	// P6: search
	if err := add(NewPostSearchCommand()); err != nil {
		return nil, err
	}
	if err := add(NewSearchCommand()); err != nil {
		return nil, err
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

// withConnection is a shorthand for the very common "this command needs a DB"
// case: it returns the connection section so callers can pass it to
// cmds.WithSections alongside their own flags.
func withConnection() (cmds.CommandDescriptionOption, error) {
	section, err := config.ConnectionSection()
	if err != nil {
		return nil, err
	}
	return cmds.WithSections(section), nil
}

// --- db init: P1 verification command -----------------------------------

// DBInitCommand opens (and migrates) the database and reports status. It
// exercises the whole P1 stack: connection section, env loading, store open,
// and migrations.
type DBInitCommand struct{ *cmds.CommandDescription }

// NewDBInitCommand builds the `agentforum db init` command.
func NewDBInitCommand() (*DBInitCommand, error) {
	sec, err := config.ConnectionSection()
	if err != nil {
		return nil, err
	}
	return &DBInitCommand{CommandDescription: cmds.NewCommandDescription(
		"init",
		cmds.WithParents("db"),
		cmds.WithShort("Open or create the database and apply migrations"),
		cmds.WithLong(`Open the agentforum SQLite database (creating it if needed), apply all pending migrations, and print the resolved database path.

This is the simplest way to verify the connection configuration works:
  agentforum db init
  AGENTFORUM_DB=/tmp/x.db agentforum db init --format json`),
		cmds.WithSections(sec),
	)}, nil
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

// keep imports referenced for later phases
var (
	_ = fields.WithHelp
	_ = schema.DefaultSlug
)
