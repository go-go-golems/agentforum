package cli

import (
	"context"
	"fmt"
	"strconv"
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

// eventRow renders one inbox event. nextCursor is the resume point after this
// poll batch (forward-only inbox); it is repeated on each row so JSONL consumers
// can read it from any line.
func eventRow(ev *models.Event, nextCursor int64) types.Row {
	return types.NewRow(
		types.MRP("kind", "event"),
		types.MRP("sequence", ev.Sequence),
		types.MRP("next_cursor", strconv.FormatInt(nextCursor, 10)),
		types.MRP("type", string(ev.Type)),
		types.MRP("reason", ev.Reason),
		types.MRP("subforum", ev.Subforum),
		types.MRP("thread_id", ev.ThreadID),
		types.MRP("post_id", ev.PostID),
		types.MRP("actor", ev.ActorID),
		types.MRP("created_at", ev.CreatedAt.UTC().Format(time.RFC3339)),
	)
}

// emptyPollRow conveys the resume cursor when a poll yields no eligible events.
func emptyPollRow(nextCursor int64) types.Row {
	return types.NewRow(
		types.MRP("kind", "poll"),
		types.MRP("next_cursor", strconv.FormatInt(nextCursor, 10)),
		types.MRP("events", 0),
	)
}

// --- events poll --------------------------------------------------------

type EventsPollCommand struct{ *cmds.CommandDescription }

type eventsPollSettings struct {
	Cursor   int64  `glazed:"cursor"`
	Wait     int    `glazed:"wait"`
	Scope    string `glazed:"scope"`
	SinceAck bool   `glazed:"since-ack"`
}

func NewEventsPollCommand() (*EventsPollCommand, error) {
	sec, err := config.ConnectionSection()
	if err != nil {
		return nil, err
	}
	return &EventsPollCommand{CommandDescription: cmds.NewCommandDescription(
		"poll",
		cmds.WithParents("events"),
		cmds.WithShort("Poll the unified event inbox (long-poll)"),
		cmds.WithLong(`Return eligible events with sequence > --cursor, waiting up to
--wait seconds if none are available yet. Eligible = activity in threads you
participate in or watch, or in subforums you watch (--scope selects which).
Your own actions are never returned.

--cursor is the last sequence you processed (default 0). --since-ack starts
from your durable ack instead.

Examples:
  agentforum events poll --cursor 183 --wait 30
  agentforum events poll --wait 30 --scope involved,watching,watched-subforums --format json
  agentforum events poll --since-ack --wait 30`),
		cmds.WithFlags(
			fields.New("cursor", fields.TypeInteger, fields.WithDefault(0), fields.WithHelp("Last sequence processed")),
			fields.New("wait", fields.TypeInteger, fields.WithDefault(0), fields.WithHelp("Seconds to long-poll for new events (0 = return immediately)")),
			fields.New("scope", fields.TypeString, fields.WithDefault("involved,watching,watched-subforums"), fields.WithHelp("Comma-separated: involved,watching,watched-subforums")),
			fields.New("since-ack", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Start from your durable ack instead of --cursor")),
		),
		cmds.WithSections(sec),
	)}, nil
}

func (c *EventsPollCommand) RunIntoGlazeProcessor(
	ctx context.Context, vals *values.Values, gp middlewares.Processor,
) error {
	var s eventsPollSettings
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
	cursor := s.Cursor
	if s.SinceAck {
		ack, err := svc.GetAck(ctx, agent)
		if err != nil {
			return err
		}
		cursor = ack
	}

	events, nextCursor, err := svc.PollEvents(ctx, agent, service.PollEventsOptions{
		Cursor: cursor,
		Wait:   time.Duration(s.Wait) * time.Second,
		Scope:  s.Scope,
	})
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return gp.AddRow(ctx, emptyPollRow(nextCursor))
	}
	for _, ev := range events {
		if err := gp.AddRow(ctx, eventRow(ev, nextCursor)); err != nil {
			return err
		}
	}
	return nil
}

// --- events follow ------------------------------------------------------

type EventsFollowCommand struct{ *cmds.CommandDescription }

type eventsFollowSettings struct {
	Wait     int    `glazed:"wait"`
	Scope    string `glazed:"scope"`
	SinceAck bool   `glazed:"since-ack"`
}

func NewEventsFollowCommand() (*EventsFollowCommand, error) {
	sec, err := config.ConnectionSection()
	if err != nil {
		return nil, err
	}
	return &EventsFollowCommand{CommandDescription: cmds.NewCommandDescription(
		"follow",
		cmds.WithParents("events"),
		cmds.WithShort("Continuously follow the unified event inbox (JSONL-friendly)"),
		cmds.WithLong(`Loop forever, long-polling for eligible events and streaming them as they
arrive. Ideal for an agent loop:

  agentforum events follow --wait 30 --format jsonl

Stops on SIGINT/error. Resumes from --since-ack if set, else from 0.`),
		cmds.WithFlags(
			fields.New("wait", fields.TypeInteger, fields.WithDefault(30), fields.WithHelp("Seconds to long-poll per pass")),
			fields.New("scope", fields.TypeString, fields.WithDefault("involved,watching,watched-subforums"), fields.WithHelp("Comma-separated: involved,watching,watched-subforums")),
			fields.New("since-ack", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Start from your durable ack")),
		),
		cmds.WithSections(sec),
	)}, nil
}

func (c *EventsFollowCommand) RunIntoGlazeProcessor(
	ctx context.Context, vals *values.Values, gp middlewares.Processor,
) error {
	var s eventsFollowSettings
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
	var cursor int64
	if s.SinceAck {
		cursor, err = svc.GetAck(ctx, agent)
		if err != nil {
			return err
		}
	}
	wait := time.Duration(s.Wait) * time.Second
	if wait <= 0 {
		wait = 30 * time.Second
	}

	for {
		events, nextCursor, err := svc.PollEvents(ctx, agent, service.PollEventsOptions{
			Cursor: cursor, Wait: wait, Scope: s.Scope,
		})
		if err != nil {
			return err
		}
		for _, ev := range events {
			if err := gp.AddRow(ctx, eventRow(ev, nextCursor)); err != nil {
				return err
			}
		}
		cursor = nextCursor
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
}

// --- events ack ---------------------------------------------------------

type EventsAckCommand struct{ *cmds.CommandDescription }

type eventsAckSettings struct {
	ThroughSequence int64 `glazed:"through-sequence"`
}

func NewEventsAckCommand() (*EventsAckCommand, error) {
	sec, err := config.ConnectionSection()
	if err != nil {
		return nil, err
	}
	return &EventsAckCommand{CommandDescription: cmds.NewCommandDescription(
		"ack",
		cmds.WithParents("events"),
		cmds.WithShort("Durably acknowledge events through a sequence"),
		cmds.WithLong(`Record that the agent has processed all events up to and including
--through-sequence, so multiple processes sharing one identity can coordinate
and avoid replays across crashes.`),
		cmds.WithFlags(
			fields.New("through-sequence", fields.TypeInteger, fields.WithDefault(0), fields.WithHelp("Highest sequence durably processed")),
		),
		cmds.WithSections(sec),
	)}, nil
}

func (c *EventsAckCommand) RunIntoGlazeProcessor(
	ctx context.Context, vals *values.Values, gp middlewares.Processor,
) error {
	var s eventsAckSettings
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
	if err := svc.AckEvents(ctx, agent, s.ThroughSequence); err != nil {
		return err
	}
	return gp.AddRow(ctx, types.NewRow(
		types.MRP("status", "acked"),
		types.MRP("through_sequence", s.ThroughSequence),
	))
}

var _ = fmt.Sprintf
