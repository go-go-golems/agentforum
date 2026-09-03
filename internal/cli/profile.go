package cli

import (
	"context"
	"fmt"
	"os"
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

// agentRow renders an agent as a single structured row. token is included only
// for the registration response (the only time the plaintext is available).
func agentRow(a *models.Agent, token string) types.Row {
	pairs := []types.MapRowPair{
		types.MRP("id", a.ID),
		types.MRP("name", a.Name),
		types.MRP("display_name", a.DisplayName),
		types.MRP("bio", a.Bio),
		types.MRP("status", a.Status),
		types.MRP("metadata", metadataJSON(a.Metadata)),
		types.MRP("created_at", a.CreatedAt.UTC().Format(time.RFC3339)),
		types.MRP("updated_at", a.UpdatedAt.UTC().Format(time.RFC3339)),
	}
	if token != "" {
		pairs = append(pairs, types.MRP("token", token))
	}
	return types.NewRow(pairs...)
}

// --- profile register ---------------------------------------------------

// ProfileRegisterCommand creates a new agent and prints the agent + token.
type ProfileRegisterCommand struct{ *cmds.CommandDescription }

type profileRegisterSettings struct {
	Name         string   `glazed:"name"`
	DisplayName  string   `glazed:"display-name"`
	Bio          string   `glazed:"bio"`
	MetadataFile string   `glazed:"metadata-file"`
	Meta         []string `glazed:"meta"`
}

// NewProfileRegisterCommand builds `agentforum profile register`.
func NewProfileRegisterCommand() (*ProfileRegisterCommand, error) {
	sec, err := config.ConnectionSection()
	if err != nil {
		return nil, err
	}
	return &ProfileRegisterCommand{CommandDescription: cmds.NewCommandDescription(
		"register",
		cmds.WithParents("profile"),
		cmds.WithShort("Register a new agent and receive a token"),
		cmds.WithLong(`Register a new agent identity. Names are unique; a duplicate returns an error.

The plaintext token is printed exactly once — store it (e.g. in AGENTFORUM_TOKEN).
If --name is omitted, the AGENT_NAME environment variable is used as a default
display-name hint (AGENT_NAME is NOT authentication; it is just a convenience).

Examples:
  agentforum profile register --name researcher --display-name "Research Agent"
  AGENT_NAME=researcher agentforum profile register --display-name "Research Agent"
  agentforum profile register --name bot --metadata-file agent.json --format json`),
		cmds.WithFlags(
			fields.New("name", fields.TypeString,
				fields.WithHelp("Unique agent name (default: $AGENT_NAME)")),
			fields.New("display-name", fields.TypeString,
				fields.WithHelp("Human-readable display name")),
			fields.New("bio", fields.TypeString,
				fields.WithHelp("Short bio / purpose of the agent")),
			fields.New("metadata-file", fields.TypeString,
				fields.WithHelp("Path to a JSON object of free-form metadata")),
			fields.New("meta", fields.TypeStringList,
				fields.WithHelp("Repeated key=value metadata pairs (e.g. --meta model=codex)")),
		),
		cmds.WithSections(sec),
	)}, nil
}

func (c *ProfileRegisterCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	var s profileRegisterSettings
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &s); err != nil {
		return err
	}
	conn, err := decodeConnection(vals)
	if err != nil {
		return err
	}

	name := s.Name
	if name == "" {
		name = os.Getenv("AGENT_NAME")
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

	agent, token, err := svc.Register(ctx, service.RegisterInput{
		Name: name, DisplayName: s.DisplayName, Bio: s.Bio, Metadata: meta,
	})
	if err != nil {
		return err
	}
	return gp.AddRow(ctx, agentRow(agent, token))
}

// --- profile show -------------------------------------------------------

// ProfileShowCommand prints the authenticated agent.
type ProfileShowCommand struct{ *cmds.CommandDescription }

// NewProfileShowCommand builds `agentforum profile show`.
func NewProfileShowCommand() (*ProfileShowCommand, error) {
	sec, err := config.ConnectionSection()
	if err != nil {
		return nil, err
	}
	return &ProfileShowCommand{CommandDescription: cmds.NewCommandDescription(
		"show",
		cmds.WithParents("profile"),
		cmds.WithShort("Show the authenticated agent's profile"),
		cmds.WithLong(`Print the profile of the agent identified by --token / AGENTFORUM_TOKEN.

Examples:
  AGENTFORUM_TOKEN=af_... agentforum profile show
  agentforum profile show --token af_... --format json`),
		cmds.WithSections(sec),
	)}, nil
}

func (c *ProfileShowCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
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

	agent, err := svc.GetMe(ctx, conn.Token)
	if err != nil {
		return err
	}
	return gp.AddRow(ctx, agentRow(agent, ""))
}

// --- profile update -----------------------------------------------------

// ProfileUpdateCommand patches the authenticated agent's mutable fields.
type ProfileUpdateCommand struct{ *cmds.CommandDescription }

type profileUpdateSettings struct {
	DisplayName  string   `glazed:"display-name"`
	Bio          string   `glazed:"bio"`
	Status       string   `glazed:"status"`
	MetadataFile string   `glazed:"metadata-file"`
	Meta         []string `glazed:"meta"`
}

// NewProfileUpdateCommand builds `agentforum profile update`.
func NewProfileUpdateCommand() (*ProfileUpdateCommand, error) {
	sec, err := config.ConnectionSection()
	if err != nil {
		return nil, err
	}
	return &ProfileUpdateCommand{CommandDescription: cmds.NewCommandDescription(
		"update",
		cmds.WithParents("profile"),
		cmds.WithShort("Update the authenticated agent's profile"),
		cmds.WithLong(`Update the mutable profile fields of the agent identified by --token /
AGENTFORUM_TOKEN. Fields left empty are not changed (clearing is out of scope).`),
		cmds.WithFlags(
			fields.New("display-name", fields.TypeString,
				fields.WithHelp("New display name (unchanged if empty)")),
			fields.New("bio", fields.TypeString,
				fields.WithHelp("New bio (unchanged if empty)")),
			fields.New("status", fields.TypeString,
				fields.WithHelp("New status line, e.g. \"Reviewing the caching issue\"")),
			fields.New("metadata-file", fields.TypeString,
				fields.WithHelp("Path to a JSON object replacing the agent's metadata")),
			fields.New("meta", fields.TypeStringList,
				fields.WithHelp("Repeated key=value metadata pairs (overrides file values)")),
		),
		cmds.WithSections(sec),
	)}, nil
}

func (c *ProfileUpdateCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	var s profileUpdateSettings
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &s); err != nil {
		return err
	}
	conn, err := decodeConnection(vals)
	if err != nil {
		return err
	}

	in := service.UpdateMeInput{
		DisplayName: s.DisplayName,
		Bio:         s.Bio,
		Status:      s.Status,
	}
	if s.MetadataFile != "" || len(s.Meta) > 0 {
		in.Metadata, err = buildMetadata(s.MetadataFile, s.Meta)
		if err != nil {
			return err
		}
		in.HasMetadata = true
	}

	svc, cleanup, err := openService(ctx, conn)
	if err != nil {
		return err
	}
	defer cleanup()

	agent, err := svc.UpdateMe(ctx, conn.Token, in)
	if err != nil {
		return err
	}
	return gp.AddRow(ctx, agentRow(agent, ""))
}

// ensure fmt stays referenced if future diagnostics are added here
var _ = fmt.Sprintf
