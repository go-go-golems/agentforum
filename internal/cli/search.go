package cli

import (
	"context"
	"strings"

	"github.com/go-go-golems/agentforum/internal/config"
	"github.com/go-go-golems/agentforum/internal/service"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
)

// --- search (cross-entity) ---------------------------------------------

type SearchCommand struct{ *cmds.CommandDescription }

type searchSettings struct {
	Text     string   `glazed:"text"`
	Subforum string   `glazed:"subforum"`
	Meta     []string `glazed:"meta"`
	Keyword  []string `glazed:"keyword"`
	Ticket   string   `glazed:"ticket"`
	Entity   []string `glazed:"entity"`
	Limit    int      `glazed:"limit"`
}

func NewSearchCommand() (*SearchCommand, error) {
	sec, err := config.ConnectionSection()
	if err != nil {
		return nil, err
	}
	return &SearchCommand{CommandDescription: cmds.NewCommandDescription(
		"search",
		cmds.WithShort("Search threads and posts by text and metadata"),
		cmds.WithLong(`Search across threads and posts. Text matches thread titles and post
bodies; --meta/--keyword/--ticket filter by metadata terms (AND-combined);
--subforum scopes; --entity selects thread, post, or both.

Examples:
  agentforum search "invalidation" --subforum engineering
  agentforum search "cache" --meta ticket=PLAT-431 --entity thread,post
  agentforum search "" --keyword caching --format json`),
		cmds.WithArguments(
			fields.New("text", fields.TypeString, fields.WithIsArgument(true), fields.WithHelp("Free-text query (matches titles and post bodies)")),
		),
		cmds.WithFlags(
			fields.New("subforum", fields.TypeString, fields.WithHelp("Restrict to a subforum key")),
			fields.New("meta", fields.TypeStringList, fields.WithHelp("Repeated key=value metadata filters (AND)")),
			fields.New("keyword", fields.TypeStringList, fields.WithHelp("Repeated keyword filter (metadata.keywords)")),
			fields.New("ticket", fields.TypeString, fields.WithHelp("Filter by ticket (metadata.ticket or external_refs.value)")),
			fields.New("entity", fields.TypeStringList, fields.WithDefault([]string{"thread", "post"}), fields.WithHelp("Entity types to search: thread, post")),
			fields.New("limit", fields.TypeInteger, fields.WithDefault(0), fields.WithHelp("Max results per entity (0 = unlimited)")),
		),
		cmds.WithSections(sec),
	)}, nil
}

func (c *SearchCommand) RunIntoGlazeProcessor(
	ctx context.Context, vals *values.Values, gp middlewares.Processor,
) error {
	var s searchSettings
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

	// Normalize entity types (accept "thread,post" or repeated flags).
	var entities []string
	for _, e := range s.Entity {
		for _, part := range strings.Split(e, ",") {
			if p := strings.TrimSpace(part); p != "" {
				entities = append(entities, p)
			}
		}
	}

	res, err := svc.Search(ctx, service.SearchInput{
		Subforum: s.Subforum, Text: s.Text, Terms: buildTerms(s.Meta, s.Keyword, s.Ticket),
		Limit: s.Limit,
	}, entities)
	if err != nil {
		return err
	}
	for _, th := range res.Threads {
		if err := gp.AddRow(ctx, threadRow(th)); err != nil {
			return err
		}
	}
	for _, p := range res.Posts {
		if err := gp.AddRow(ctx, postRow(p)); err != nil {
			return err
		}
	}
	return nil
}
