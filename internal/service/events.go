package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-go-golems/agentforum/internal/models"
)

// pollPageSize caps how many events are fetched per pass. A full page of
// ineligible events is advanced past without sleeping, so a chatty author
// never wedges the poll.
const pollPageSize = 500

// pollInterval is the sleep between passes when caught up (no new events).
const pollInterval = 200 * time.Millisecond

// PollEventsOptions configures a unified inbox poll.
type PollEventsOptions struct {
	Cursor int64
	Wait   time.Duration // 0 = single pass, no blocking
	Scope  string        // "involved,watching,watched-subforums"; empty = all
}

// reasonSet is the set of accepted reasons derived from the scope string.
type reasonSet struct {
	participating   bool
	watching        bool
	watchedSubforum bool
}

func parseScope(s string) reasonSet {
	s = strings.TrimSpace(s)
	if s == "" {
		return reasonSet{participating: true, watching: true, watchedSubforum: true}
	}
	rs := reasonSet{}
	for _, part := range strings.Split(s, ",") {
		switch strings.TrimSpace(part) {
		case "involved":
			rs.participating = true
		case "watching", "watched-threads", "watching-threads":
			rs.watching = true
		case "watched-subforum", "watched-subforums":
			rs.watchedSubforum = true
		}
	}
	return rs
}

// Accepts reports whether a reason is in the set.
func (rs reasonSet) Accepts(reason string) bool {
	switch reason {
	case models.ReasonParticipating:
		return rs.participating
	case models.ReasonWatching:
		return rs.watching
	case models.ReasonWatchedSubforum:
		return rs.watchedSubforum
	}
	return false
}

// PollEvents returns eligible events with sequence > cursor, waiting up to
// opts.Wait if none are available yet. nextCursor advances past every event
// examined (forward-only inbox), so the agent does not receive pre-watch
// history; eligible events are always delivered before being advanced past.
func (s *Service) PollEvents(ctx context.Context, agent *models.Agent, opts PollEventsOptions) ([]*models.Event, int64, error) {
	rs := parseScope(opts.Scope)
	cursor := opts.Cursor
	deadline := time.Now()
	if opts.Wait > 0 {
		deadline = deadline.Add(opts.Wait)
	}

	for {
		page, err := s.store.ListEventsAfter(ctx, cursor, pollPageSize)
		if err != nil {
			return nil, cursor, err
		}

		// Batch the membership sets for the agent once per pass.
		partThreads, watchThreads, watchSubs, err := s.membershipSets(ctx, agent.ID)
		if err != nil {
			return nil, cursor, err
		}

		var eligible []*models.Event
		for _, ev := range page {
			if ev.ActorID == agent.ID {
				continue // never notify an agent of its own action
			}
			reason := eventReason(ev, partThreads, watchThreads, watchSubs)
			if reason == "" || !rs.Accepts(reason) {
				continue
			}
			ev.Reason = reason
			eligible = append(eligible, ev)
		}

		if len(page) > 0 {
			// Advance past everything examined (forward-only).
			cursor = page[len(page)-1].Sequence
		}

		if len(eligible) > 0 {
			return eligible, cursor, nil
		}

		// No eligible events this pass.
		if len(page) < pollPageSize {
			// Caught up: wait if we have time budget, else return.
			if opts.Wait <= 0 || time.Now().After(deadline) {
				return nil, cursor, nil
			}
			sleep := pollInterval
			if remaining := time.Until(deadline); remaining < sleep {
				sleep = remaining
			}
			if err := sleepCtx(ctx, sleep); err != nil {
				return nil, cursor, err
			}
		}
		// A full page of ineligible events: loop immediately to advance.
	}
}

// membershipSets returns the agent's participated threads, watched threads, and
// watched subforums as sets for O(1) reason lookups.
func (s *Service) membershipSets(ctx context.Context, agentID string) (map[string]bool, map[string]bool, map[string]bool, error) {
	partIDs, err := s.store.ListParticipantThreadIDs(ctx, agentID)
	if err != nil {
		return nil, nil, nil, err
	}
	watchIDs, err := s.store.ListWatchingThreadIDs(ctx, agentID)
	if err != nil {
		return nil, nil, nil, err
	}
	watchSubs, err := s.store.WatchedSubforumKeys(ctx, agentID)
	if err != nil {
		return nil, nil, nil, err
	}
	toSet := func(xs []string) map[string]bool {
		m := make(map[string]bool, len(xs))
		for _, x := range xs {
			m[x] = true
		}
		return m
	}
	return toSet(partIDs), toSet(watchIDs), toSet(watchSubs), nil
}

// eventReason computes why an event is relevant to an agent, given the
// agent's membership sets. Returns "" if not relevant. Precedence:
// participating > watching > watched_subforum.
func eventReason(ev *models.Event, partThreads, watchThreads, watchSubs map[string]bool) string {
	switch {
	case partThreads[ev.ThreadID]:
		return models.ReasonParticipating
	case watchThreads[ev.ThreadID]:
		return models.ReasonWatching
	case ev.Subforum != "" && watchSubs[ev.Subforum]:
		return models.ReasonWatchedSubforum
	}
	return ""
}

// AckEvents durably records that an agent processed through a sequence, so
// multiple processes sharing one identity can coordinate.
func (s *Service) AckEvents(ctx context.Context, agent *models.Agent, through int64) error {
	if through < 0 {
		return fmt.Errorf("%w: through_sequence must be >= 0", ErrInvalidInput)
	}
	return s.store.AckEvents(ctx, agent.ID, through)
}

// GetAck returns the agent's last acked sequence (0 if none).
func (s *Service) GetAck(ctx context.Context, agent *models.Agent) (int64, error) {
	return s.store.GetAck(ctx, agent.ID)
}

// sleepCtx sleeps for d or until ctx is cancelled.
func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
