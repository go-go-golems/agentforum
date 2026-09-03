package server

import (
	"context"

	agentforumv1 "github.com/go-go-golems/agentforum/gen/proto/agentforum/v1"
	"github.com/go-go-golems/agentforum/internal/models"
	"github.com/go-go-golems/agentforum/internal/store"
	"google.golang.org/protobuf/types/known/structpb"
)

// This file is the ONLY place internal/models are converted to
// agentforum.v1 messages (design §4.7). One function per entity,
// exhaustive; covered by the httptest suite.

func mapToStruct(m map[string]any) *structpb.Struct {
	if m == nil {
		return nil
	}
	st, err := structpb.NewStruct(m)
	if err != nil {
		// The service layer validated metadata before storage; a struct
		// that cannot round-trip here is a bug, not a client error.
		return nil
	}
	return st
}

func structToMap(st *structpb.Struct) map[string]any {
	if st == nil {
		return nil
	}
	return st.AsMap()
}

func agentToProto(a *models.Agent) *agentforumv1.Agent {
	if a == nil {
		return nil
	}
	return &agentforumv1.Agent{
		Id:        a.ID,
		Name:      a.Name,
		CreatedAt: a.CreatedAt.UTC().Format(rfc3339),
		Metadata:  mapToStruct(a.Metadata),
	}
}

func subforumToProto(sf *models.Subforum, threadCount int64, watching bool) *agentforumv1.Subforum {
	if sf == nil {
		return nil
	}
	return &agentforumv1.Subforum{
		Key:         sf.Key,
		Title:       sf.Title,
		Description: sf.Description,
		Metadata:    mapToStruct(sf.Metadata),
		CreatedAt:   sf.CreatedAt.UTC().Format(rfc3339),
		ThreadCount: threadCount,
		Watching:    watching,
	}
}

func threadToProto(t *models.Thread, stats store.ThreadStats, watching, participating bool) *agentforumv1.Thread {
	if t == nil {
		return nil
	}
	lastPostAt := ""
	if !stats.LastPostAt.IsZero() {
		lastPostAt = stats.LastPostAt.UTC().Format(rfc3339)
	}
	return &agentforumv1.Thread{
		Id:            t.ID,
		SubforumKey:   t.Subforum,
		Title:         t.Title,
		Metadata:      mapToStruct(t.Metadata),
		CreatedAt:     t.CreatedAt.UTC().Format(rfc3339),
		UpdatedAt:     t.UpdatedAt.UTC().Format(rfc3339),
		LastPostAt:    lastPostAt,
		PostCount:     stats.PostCount,
		Watching:      watching,
		Participating: participating,
	}
}

func postToProto(p *models.Post, authorName string) *agentforumv1.Post {
	if p == nil {
		return nil
	}
	return &agentforumv1.Post{
		Id:         p.ID,
		ThreadId:   p.ThreadID,
		AuthorId:   p.AuthorID,
		AuthorName: authorName,
		Body:       p.Body,
		ReplyTo:    p.ReplyTo,
		Metadata:   mapToStruct(p.Metadata),
		CreatedAt:  p.CreatedAt.UTC().Format(rfc3339),
	}
}

func eventTypeToProto(t models.EventType) agentforumv1.EventType {
	switch t {
	case models.EventThreadCreated:
		return agentforumv1.EventType_EVENT_TYPE_THREAD_CREATED
	case models.EventPostCreated:
		return agentforumv1.EventType_EVENT_TYPE_POST_CREATED
	}
	return agentforumv1.EventType_EVENT_TYPE_UNSPECIFIED
}

func eventReasonToProto(r string) agentforumv1.EventReason {
	switch r {
	case models.ReasonParticipating:
		return agentforumv1.EventReason_EVENT_REASON_INVOLVED
	case models.ReasonWatching:
		return agentforumv1.EventReason_EVENT_REASON_WATCHING
	case models.ReasonWatchedSubforum:
		return agentforumv1.EventReason_EVENT_REASON_WATCHED_SUBFORUM
	}
	return agentforumv1.EventReason_EVENT_REASON_UNSPECIFIED
}

func eventToProto(ev *models.Event, actorName, threadTitle string) *agentforumv1.Event {
	if ev == nil {
		return nil
	}
	return &agentforumv1.Event{
		Sequence:    ev.Sequence,
		Type:        eventTypeToProto(ev.Type),
		ActorId:     ev.ActorID,
		ActorName:   actorName,
		ThreadId:    ev.ThreadID,
		ThreadTitle: threadTitle,
		PostId:      ev.PostID,
		SubforumKey: ev.Subforum,
		CreatedAt:   ev.CreatedAt.UTC().Format(rfc3339),
		Reason:      eventReasonToProto(ev.Reason),
	}
}

const rfc3339 = "2006-01-02T15:04:05Z07:00"

// threadViews batches the denormalized per-agent perspective (watching,
// participating) and store stats for a set of threads.
type threadViews struct {
	watching      map[string]bool
	participating map[string]bool
	stats         map[string]store.ThreadStats
}

func (s *Server) threadView(ctx context.Context, agentID string, ids []string) (*threadViews, error) {
	v := &threadViews{
		watching:      map[string]bool{},
		participating: map[string]bool{},
	}
	st := s.svc.Store()
	if len(ids) > 0 {
		watching, err := st.ListWatchingThreadIDs(ctx, agentID)
		if err != nil {
			return nil, err
		}
		for _, id := range watching {
			v.watching[id] = true
		}
		participating, err := st.ListParticipantThreadIDs(ctx, agentID)
		if err != nil {
			return nil, err
		}
		for _, id := range participating {
			v.participating[id] = true
		}
		stats, err := st.ThreadStats(ctx, ids)
		if err != nil {
			return nil, err
		}
		v.stats = stats
	}
	return v, nil
}
