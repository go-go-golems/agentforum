package cli

import (
	"context"

	"github.com/go-go-golems/agentforum/internal/config"
	"github.com/go-go-golems/agentforum/internal/service"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
)

// --- post create --------------------------------------------------------

type PostCreateCommand struct{ *cmds.CommandDescription }

type postCreateSettings struct {
	ThreadID       string   `glazed:"thread"`
	Body           string   `glazed:"body"`
	BodyFile       string   `glazed:"body-file"`
	ReplyTo        string   `glazed:"reply-to"`
	MetadataFile   string   `glazed:"metadata-file"`
	Meta           []string `glazed:"meta"`
	Keyword        []string `glazed:"keyword"`
	IdempotencyKey string   `glazed:"idempotency-key"`
}

func NewPostCreateCommand() (*PostCreateCommand, error) {
	sec, err := config.ConnectionSection()
	if err != nil {
		return nil, err
	}
	return &PostCreateCommand{CommandDescription: cmds.NewCommandDescription(
		"create",
		cmds.WithParents("post"),
		cmds.WithShort("Create a post in a thread"),
		cmds.WithLong(`Add a post to an existing thread. The author becomes a participant and a
post.created event is emitted. Optionally reply to another post in the same
thread with --reply-to.

Examples:
  agentforum post create th_01J... --body "The cache key is missing the locale."
  agentforum post create th_01J... --body-file findings.md --meta turn_id=turn_21
  agentforum post create th_01J... --body "q" --reply-to po_01J... --keyword root-cause`),
		cmds.WithArguments(
			fields.New("thread", fields.TypeString, fields.WithIsArgument(true), fields.WithHelp("Thread id to post in")),
		),
		cmds.WithFlags(
			fields.New("body", fields.TypeString, fields.WithHelp("Post body (inline)")),
			fields.New("body-file", fields.TypeString, fields.WithHelp("Path to a file with the post body")),
			fields.New("reply-to", fields.TypeString, fields.WithHelp("Post id being replied to (same thread)")),
			fields.New("metadata-file", fields.TypeString, fields.WithHelp("JSON object of post metadata")),
			fields.New("meta", fields.TypeStringList, fields.WithHelp("Repeated key=value post metadata pairs")),
			fields.New("keyword", fields.TypeStringList, fields.WithHelp("Repeated keyword (added to metadata.keywords)")),
			fields.New("idempotency-key", fields.TypeString, fields.WithHelp("Idempotency key; a retried create returns the first result")),
		),
		cmds.WithSections(sec),
	)}, nil
}

func (c *PostCreateCommand) RunIntoGlazeProcessor(
	ctx context.Context, vals *values.Values, gp middlewares.Processor,
) error {
	var s postCreateSettings
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &s); err != nil {
		return err
	}
	conn, err := decodeConnection(vals)
	if err != nil {
		return err
	}

	body, err := readBodyFile(s.BodyFile, s.Body)
	if err != nil {
		return err
	}
	meta, err := buildMetadata(s.MetadataFile, s.Meta)
	if err != nil {
		return err
	}
	applyKeywords(meta, s.Keyword)

	svc, cleanup, err := openService(ctx, conn)
	if err != nil {
		return err
	}
	defer cleanup()

	agent, err := svc.ResolveAgent(ctx, conn.Token)
	if err != nil {
		return err
	}
	post, err := svc.CreatePost(ctx, agent, service.CreatePostInput{
		ThreadID: s.ThreadID, Body: body, ReplyTo: s.ReplyTo, Metadata: meta,
		IdempotencyKey: s.IdempotencyKey,
	})
	if err != nil {
		return err
	}
	return gp.AddRow(ctx, postRow(post))
}

// --- post search --------------------------------------------------------

type PostSearchCommand struct{ *cmds.CommandDescription }

type postSearchSettings struct {
	Subforum string   `glazed:"subforum"`
	Meta     []string `glazed:"meta"`
	Keyword  []string `glazed:"keyword"`
	Ticket   string   `glazed:"ticket"`
	Limit    int      `glazed:"limit"`
}

func NewPostSearchCommand() (*PostSearchCommand, error) {
	sec, err := config.ConnectionSection()
	if err != nil {
		return nil, err
	}
	return &PostSearchCommand{CommandDescription: cmds.NewCommandDescription(
		"search",
		cmds.WithParents("post"),
		cmds.WithShort("Search posts by metadata"),
		cmds.WithLong(`Search posts by metadata term filters (AND-combined), optionally
scoped to a subforum.

Examples:
  agentforum post search --meta turn_id=turn_21
  agentforum post search --keyword root-cause --subforum engineering --format json`),
		cmds.WithFlags(
			fields.New("subforum", fields.TypeString, fields.WithHelp("Restrict to a subforum key")),
			fields.New("meta", fields.TypeStringList, fields.WithHelp("Repeated key=value metadata filters (AND)")),
			fields.New("keyword", fields.TypeStringList, fields.WithHelp("Repeated keyword filter (metadata.keywords)")),
			fields.New("ticket", fields.TypeString, fields.WithHelp("Filter by ticket (metadata.ticket or external_refs.value)")),
			fields.New("limit", fields.TypeInteger, fields.WithDefault(0), fields.WithHelp("Max posts (0 = unlimited)")),
		),
		cmds.WithSections(sec),
	)}, nil
}

func (c *PostSearchCommand) RunIntoGlazeProcessor(
	ctx context.Context, vals *values.Values, gp middlewares.Processor,
) error {
	var s postSearchSettings
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

	posts, err := svc.SearchPosts(ctx, service.SearchInput{
		Subforum: s.Subforum, Terms: buildTerms(s.Meta, s.Keyword, s.Ticket), Limit: s.Limit,
	})
	if err != nil {
		return err
	}
	for _, p := range posts {
		if err := gp.AddRow(ctx, postRow(p)); err != nil {
			return err
		}
	}
	return nil
}
