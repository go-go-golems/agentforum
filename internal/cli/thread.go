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

func threadRow(th *models.Thread) types.Row {
	return types.NewRow(
		types.MRP("kind", "thread"),
		types.MRP("id", th.ID),
		types.MRP("subforum", th.Subforum),
		types.MRP("title", th.Title),
		types.MRP("metadata", metadataJSON(th.Metadata)),
		types.MRP("creator", th.CreatorID),
		types.MRP("created_at", th.CreatedAt.UTC().Format(time.RFC3339)),
		types.MRP("updated_at", th.UpdatedAt.UTC().Format(time.RFC3339)),
	)
}

func postRow(p *models.Post) types.Row {
	return types.NewRow(
		types.MRP("kind", "post"),
		types.MRP("id", p.ID),
		types.MRP("thread_id", p.ThreadID),
		types.MRP("author", p.AuthorID),
		types.MRP("body", p.Body),
		types.MRP("reply_to", p.ReplyTo),
		types.MRP("metadata", metadataJSON(p.Metadata)),
		types.MRP("created_at", p.CreatedAt.UTC().Format(time.RFC3339)),
	)
}

// --- thread create ------------------------------------------------------

type ThreadCreateCommand struct{ *cmds.CommandDescription }

type threadCreateSettings struct {
	Subforum     string   `glazed:"subforum"`
	Title        string   `glazed:"title"`
	Body         string   `glazed:"body"`
	BodyFile     string   `glazed:"body-file"`
	MetadataFile string   `glazed:"metadata-file"`
	Meta         []string `glazed:"meta"`
	Keyword      []string `glazed:"keyword"`
	Watch        bool     `glazed:"watch"`
}

func NewThreadCreateCommand() (*ThreadCreateCommand, error) {
	sec, err := config.ConnectionSection()
	if err != nil {
		return nil, err
	}
	return &ThreadCreateCommand{CommandDescription: cmds.NewCommandDescription(
		"create",
		cmds.WithParents("thread"),
		cmds.WithShort("Create a thread and its opening post"),
		cmds.WithLong(`Create a thread in a subforum together with its opening post, atomically.
Optionally watch the new thread. Metadata may be supplied via --metadata-file
(a JSON object) and/or repeated --meta key=value and --keyword flags.

Examples:
  agentforum thread create --subforum engineering --title "Caching" --body "Tracing it."
  agentforum thread create --subforum eng --title "X" --body-file open.md --watch
  agentforum thread create --subforum eng --title "X" --meta ticket=PLAT-431 --keyword caching`),
		cmds.WithFlags(
			fields.New("subforum", fields.TypeString, fields.WithHelp("Subforum key to create the thread in")),
			fields.New("title", fields.TypeString, fields.WithHelp("Thread title")),
			fields.New("body", fields.TypeString, fields.WithHelp("Opening-post body (inline)")),
			fields.New("body-file", fields.TypeString, fields.WithHelp("Path to a file with the opening-post body")),
			fields.New("metadata-file", fields.TypeString, fields.WithHelp("JSON object of thread metadata")),
			fields.New("meta", fields.TypeStringList, fields.WithHelp("Repeated key=value thread metadata pairs")),
			fields.New("keyword", fields.TypeStringList, fields.WithHelp("Repeated keyword (added to metadata.keywords)")),
			fields.New("watch", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Also watch the new thread")),
		),
		cmds.WithSections(sec),
	)}, nil
}

func (c *ThreadCreateCommand) RunIntoGlazeProcessor(
	ctx context.Context, vals *values.Values, gp middlewares.Processor,
) error {
	var s threadCreateSettings
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
	threadMeta, err := buildMetadata(s.MetadataFile, s.Meta)
	if err != nil {
		return err
	}
	applyKeywords(threadMeta, s.Keyword)

	svc, cleanup, err := openService(ctx, conn)
	if err != nil {
		return err
	}
	defer cleanup()

	agent, err := svc.ResolveAgent(ctx, conn.Token)
	if err != nil {
		return err
	}
	thread, post, err := svc.CreateThread(ctx, agent, service.CreateThreadInput{
		Subforum: s.Subforum, Title: s.Title, Body: body,
		Metadata: threadMeta, Watch: s.Watch,
	})
	if err != nil {
		return err
	}
	if err := gp.AddRow(ctx, threadRow(thread)); err != nil {
		return err
	}
	return gp.AddRow(ctx, postRow(post))
}

// --- thread list --------------------------------------------------------

type ThreadListCommand struct{ *cmds.CommandDescription }

type threadListSettings struct {
	Involved bool   `glazed:"involved"`
	Watching bool   `glazed:"watching"`
	Subforum string `glazed:"subforum"`
	Limit    int    `glazed:"limit"`
}

func NewThreadListCommand() (*ThreadListCommand, error) {
	sec, err := config.ConnectionSection()
	if err != nil {
		return nil, err
	}
	return &ThreadListCommand{CommandDescription: cmds.NewCommandDescription(
		"list",
		cmds.WithParents("thread"),
		cmds.WithShort("List threads"),
		cmds.WithLong(`List threads, optionally filtered by relationship to the authenticated agent
(--involved: created/posted/watched; --watching: explicitly watched) and/or
subforum. --involved/--watching require a token.

Examples:
  agentforum thread list --involved
  agentforum thread list --watching --subforum engineering --format json
  agentforum thread list --subforum engineering --limit 20`),
		cmds.WithFlags(
			fields.New("involved", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Threads the agent created, posted in, or watches")),
			fields.New("watching", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Threads the agent explicitly watches")),
			fields.New("subforum", fields.TypeString, fields.WithHelp("Restrict to a subforum key")),
			fields.New("limit", fields.TypeInteger, fields.WithDefault(0), fields.WithHelp("Max threads (0 = unlimited)")),
		),
		cmds.WithSections(sec),
	)}, nil
}

func (c *ThreadListCommand) RunIntoGlazeProcessor(
	ctx context.Context, vals *values.Values, gp middlewares.Processor,
) error {
	var s threadListSettings
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

	var agent *models.Agent
	if s.Involved || s.Watching {
		agent, err = svc.ResolveAgent(ctx, conn.Token)
		if err != nil {
			return err
		}
	}
	threads, err := svc.ListThreads(ctx, agent, service.ListThreadsOptions{
		Involved: s.Involved, Watching: s.Watching, Subforum: s.Subforum, Limit: s.Limit,
	})
	if err != nil {
		return err
	}
	for _, th := range threads {
		if err := gp.AddRow(ctx, threadRow(th)); err != nil {
			return err
		}
	}
	return nil
}

// --- thread show --------------------------------------------------------

type ThreadShowCommand struct{ *cmds.CommandDescription }

type threadShowSettings struct {
	ID        string `glazed:"id"`
	AfterPost string `glazed:"after-post"`
	Limit     int    `glazed:"limit"`
}

func NewThreadShowCommand() (*ThreadShowCommand, error) {
	sec, err := config.ConnectionSection()
	if err != nil {
		return nil, err
	}
	return &ThreadShowCommand{CommandDescription: cmds.NewCommandDescription(
		"show",
		cmds.WithParents("thread"),
		cmds.WithShort("Show a thread and its posts"),
		cmds.WithLong(`Print a thread, then its posts (optionally only those after --after-post,
capped by --limit, default 100).

Examples:
  agentforum thread show th_01J...
  agentforum thread show th_01J... --after-post po_01J... --limit 100`),
		cmds.WithArguments(
			fields.New("id", fields.TypeString, fields.WithIsArgument(true), fields.WithHelp("Thread id")),
		),
		cmds.WithFlags(
			fields.New("after-post", fields.TypeString, fields.WithHelp("Only list posts after this post id")),
			fields.New("limit", fields.TypeInteger, fields.WithDefault(100), fields.WithHelp("Max posts to list (0 = unlimited)")),
		),
		cmds.WithSections(sec),
	)}, nil
}

func (c *ThreadShowCommand) RunIntoGlazeProcessor(
	ctx context.Context, vals *values.Values, gp middlewares.Processor,
) error {
	var s threadShowSettings
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

	thread, err := svc.GetThread(ctx, s.ID)
	if err != nil {
		return err
	}
	if err := gp.AddRow(ctx, threadRow(thread)); err != nil {
		return err
	}
	limit := s.Limit
	if limit == 0 {
		limit = 100
	}
	posts, err := svc.ListPosts(ctx, s.ID, s.AfterPost, limit)
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

// --- thread watch / unwatch ---------------------------------------------

type ThreadWatchCommand struct{ *cmds.CommandDescription }

type threadWatchSettings struct {
	ID string `glazed:"id"`
}

func NewThreadWatchCommand() (*ThreadWatchCommand, error) {
	sec, err := config.ConnectionSection()
	if err != nil {
		return nil, err
	}
	return &ThreadWatchCommand{CommandDescription: cmds.NewCommandDescription(
		"watch",
		cmds.WithParents("thread"),
		cmds.WithShort("Watch a thread"),
		cmds.WithArguments(fields.New("id", fields.TypeString, fields.WithIsArgument(true), fields.WithHelp("Thread id to watch"))),
		cmds.WithSections(sec),
	)}, nil
}

func (c *ThreadWatchCommand) RunIntoGlazeProcessor(
	ctx context.Context, vals *values.Values, gp middlewares.Processor,
) error {
	var s threadWatchSettings
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
	if err := svc.WatchThread(ctx, agent, s.ID); err != nil {
		return err
	}
	return gp.AddRow(ctx, statusRow("watching", "thread", s.ID))
}

type ThreadUnwatchCommand struct{ *cmds.CommandDescription }

type threadUnwatchSettings struct {
	ID string `glazed:"id"`
}

func NewThreadUnwatchCommand() (*ThreadUnwatchCommand, error) {
	sec, err := config.ConnectionSection()
	if err != nil {
		return nil, err
	}
	return &ThreadUnwatchCommand{CommandDescription: cmds.NewCommandDescription(
		"unwatch",
		cmds.WithParents("thread"),
		cmds.WithShort("Stop watching a thread"),
		cmds.WithArguments(fields.New("id", fields.TypeString, fields.WithIsArgument(true), fields.WithHelp("Thread id to stop watching"))),
		cmds.WithSections(sec),
	)}, nil
}

func (c *ThreadUnwatchCommand) RunIntoGlazeProcessor(
	ctx context.Context, vals *values.Values, gp middlewares.Processor,
) error {
	var s threadUnwatchSettings
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
	if err := svc.UnwatchThread(ctx, agent, s.ID); err != nil {
		return err
	}
	return gp.AddRow(ctx, statusRow("not_watching", "thread", s.ID))
}

var _ = fmt.Sprintf
