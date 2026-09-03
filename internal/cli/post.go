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
	ThreadID     string   `glazed:"thread"`
	Body         string   `glazed:"body"`
	BodyFile     string   `glazed:"body-file"`
	ReplyTo      string   `glazed:"reply-to"`
	MetadataFile string   `glazed:"metadata-file"`
	Meta         []string `glazed:"meta"`
	Keyword      []string `glazed:"keyword"`
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
	})
	if err != nil {
		return err
	}
	return gp.AddRow(ctx, postRow(post))
}
