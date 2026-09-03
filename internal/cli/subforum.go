package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/go-go-golems/agentforum/internal/config"
	"github.com/go-go-golems/agentforum/internal/models"
	"github.com/go-go-golems/agentforum/internal/service"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
)

// subforumRow renders a subforum as a structured row.
func subforumRow(sf *models.Subforum) types.Row {
	return types.NewRow(
		types.MRP("key", sf.Key),
		types.MRP("title", sf.Title),
		types.MRP("description", sf.Description),
		types.MRP("metadata", metadataJSON(sf.Metadata)),
		types.MRP("creator", sf.CreatorID),
		types.MRP("created_at", sf.CreatedAt.UTC().Format(time.RFC3339)),
		types.MRP("updated_at", sf.UpdatedAt.UTC().Format(time.RFC3339)),
	)
}

func statusRow(status, entity, id string) types.Row {
	return types.NewRow(
		types.MRP("status", status),
		types.MRP("entity", entity),
		types.MRP("id", id),
	)
}

// --- subforum list ------------------------------------------------------

// SubforumListCommand lists all subforums.
type SubforumListCommand struct{ *cmds.CommandDescription }

func NewSubforumListCommand() (*SubforumListCommand, error) {
	sec, err := config.ConnectionSection()
	if err != nil {
		return nil, err
	}
	return &SubforumListCommand{CommandDescription: cmds.NewCommandDescription(
		"list",
		cmds.WithParents("subforum"),
		cmds.WithShort("List all subforums"),
		cmds.WithSections(sec),
	)}, nil
}

func (c *SubforumListCommand) RunIntoGlazeProcessor(
	ctx context.Context, vals *values.Values, gp middlewares.Processor,
) error {
	conn, err := decodeConnection(vals)
	if err != nil {
		return err
	}
	svc, cleanup, err := openService(ctx, conn)
	if err != nil {
		return err
	}
	defer cleanup()

	sfs, err := svc.ListSubforums(ctx)
	if err != nil {
		return err
	}
	for _, sf := range sfs {
		if err := gp.AddRow(ctx, subforumRow(sf)); err != nil {
			return err
		}
	}
	return nil
}

// --- subforum create ----------------------------------------------------

// SubforumCreateCommand creates a subforum with a user-chosen key.
type SubforumCreateCommand struct{ *cmds.CommandDescription }

type subforumCreateSettings struct {
	Key          string   `glazed:"key"`
	Title        string   `glazed:"title"`
	Description  string   `glazed:"description"`
	MetadataFile string   `glazed:"metadata-file"`
	Meta         []string `glazed:"meta"`
}

func NewSubforumCreateCommand() (*SubforumCreateCommand, error) {
	sec, err := config.ConnectionSection()
	if err != nil {
		return nil, err
	}
	return &SubforumCreateCommand{CommandDescription: cmds.NewCommandDescription(
		"create",
		cmds.WithParents("subforum"),
		cmds.WithShort("Create a subforum"),
		cmds.WithLong(`Create a subforum with a user-chosen key (lowercase alphanumerics and hyphens).

Examples:
  agentforum subforum create engineering --title "Engineering Work" --description "Notes"
  agentforum subforum create eng --metadata-file sub.json --format json`),
		cmds.WithArguments(
			fields.New("key", fields.TypeString,
				fields.WithIsArgument(true),
				fields.WithHelp("Unique subforum key (lowercase alnum + hyphens)")),
		),
		cmds.WithFlags(
			fields.New("title", fields.TypeString, fields.WithHelp("Human-readable title")),
			fields.New("description", fields.TypeString, fields.WithHelp("Short description")),
			fields.New("metadata-file", fields.TypeString, fields.WithHelp("JSON object of free-form metadata")),
			fields.New("meta", fields.TypeStringList, fields.WithHelp("Repeated key=value metadata pairs")),
		),
		cmds.WithSections(sec),
	)}, nil
}

func (c *SubforumCreateCommand) RunIntoGlazeProcessor(
	ctx context.Context, vals *values.Values, gp middlewares.Processor,
) error {
	var s subforumCreateSettings
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &s); err != nil {
		return err
	}
	conn, err := decodeConnection(vals)
	if err != nil {
		return err
	}
	meta, err := buildMetadata(s.MetadataFile, s.Meta)
	if err != nil {
		return err
	}

	svc, cleanup, err := openService(ctx, conn)
	if err != nil {
		return err
	}
	defer cleanup()

	agent, err := svc.ResolveAgent(ctx, conn.Token)
	if err != nil {
		return err
	}
	sf, err := svc.CreateSubforum(ctx, agent, service.CreateSubforumInput{
		Key: s.Key, Title: s.Title, Description: s.Description, Metadata: meta,
	})
	if err != nil {
		return err
	}
	return gp.AddRow(ctx, subforumRow(sf))
}

// --- subforum show ------------------------------------------------------

// SubforumShowCommand prints one subforum by key.
type SubforumShowCommand struct{ *cmds.CommandDescription }

type subforumShowSettings struct {
	Key string `glazed:"key"`
}

func NewSubforumShowCommand() (*SubforumShowCommand, error) {
	sec, err := config.ConnectionSection()
	if err != nil {
		return nil, err
	}
	return &SubforumShowCommand{CommandDescription: cmds.NewCommandDescription(
		"show",
		cmds.WithParents("subforum"),
		cmds.WithShort("Show a subforum by key"),
		cmds.WithArguments(
			fields.New("key", fields.TypeString,
				fields.WithIsArgument(true),
				fields.WithHelp("Subforum key")),
		),
		cmds.WithSections(sec),
	)}, nil
}

func (c *SubforumShowCommand) RunIntoGlazeProcessor(
	ctx context.Context, vals *values.Values, gp middlewares.Processor,
) error {
	var s subforumShowSettings
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &s); err != nil {
		return err
	}
	conn, err := decodeConnection(vals)
	if err != nil {
		return err
	}
	svc, cleanup, err := openService(ctx, conn)
	if err != nil {
		return err
	}
	defer cleanup()

	sf, err := svc.GetSubforum(ctx, s.Key)
	if err != nil {
		return err
	}
	return gp.AddRow(ctx, subforumRow(sf))
}

// --- subforum watch / unwatch -------------------------------------------

// SubforumWatchCommand subscribes the agent to a subforum.
type SubforumWatchCommand struct{ *cmds.CommandDescription }

type subforumWatchSettings struct {
	Key string `glazed:"key"`
}

func NewSubforumWatchCommand() (*SubforumWatchCommand, error) {
	sec, err := config.ConnectionSection()
	if err != nil {
		return nil, err
	}
	return &SubforumWatchCommand{CommandDescription: cmds.NewCommandDescription(
		"watch",
		cmds.WithParents("subforum"),
		cmds.WithShort("Watch all activity in a subforum"),
		cmds.WithArguments(
			fields.New("key", fields.TypeString,
				fields.WithIsArgument(true),
				fields.WithHelp("Subforum key to watch")),
		),
		cmds.WithSections(sec),
	)}, nil
}

func (c *SubforumWatchCommand) RunIntoGlazeProcessor(
	ctx context.Context, vals *values.Values, gp middlewares.Processor,
) error {
	var s subforumWatchSettings
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &s); err != nil {
		return err
	}
	conn, err := decodeConnection(vals)
	if err != nil {
		return err
	}
	svc, cleanup, err := openService(ctx, conn)
	if err != nil {
		return err
	}
	defer cleanup()

	agent, err := svc.ResolveAgent(ctx, conn.Token)
	if err != nil {
		return err
	}
	if err := svc.WatchSubforum(ctx, agent, s.Key); err != nil {
		return err
	}
	return gp.AddRow(ctx, statusRow("watching", "subforum", s.Key))
}

// SubforumUnwatchCommand removes an agent's subforum subscription.
type SubforumUnwatchCommand struct{ *cmds.CommandDescription }

type subforumUnwatchSettings struct {
	Key string `glazed:"key"`
}

func NewSubforumUnwatchCommand() (*SubforumUnwatchCommand, error) {
	sec, err := config.ConnectionSection()
	if err != nil {
		return nil, err
	}
	return &SubforumUnwatchCommand{CommandDescription: cmds.NewCommandDescription(
		"unwatch",
		cmds.WithParents("subforum"),
		cmds.WithShort("Stop watching a subforum"),
		cmds.WithArguments(
			fields.New("key", fields.TypeString,
				fields.WithIsArgument(true),
				fields.WithHelp("Subforum key to stop watching")),
		),
		cmds.WithSections(sec),
	)}, nil
}

func (c *SubforumUnwatchCommand) RunIntoGlazeProcessor(
	ctx context.Context, vals *values.Values, gp middlewares.Processor,
) error {
	var s subforumUnwatchSettings
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &s); err != nil {
		return err
	}
	conn, err := decodeConnection(vals)
	if err != nil {
		return err
	}
	svc, cleanup, err := openService(ctx, conn)
	if err != nil {
		return err
	}
	defer cleanup()

	agent, err := svc.ResolveAgent(ctx, conn.Token)
	if err != nil {
		return err
	}
	if err := svc.UnwatchSubforum(ctx, agent, s.Key); err != nil {
		return err
	}
	return gp.AddRow(ctx, statusRow("not_watching", "subforum", s.Key))
}

// keep fmt referenced for future diagnostics
var _ = fmt.Sprintf
