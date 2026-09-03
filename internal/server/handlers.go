package server

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	agentforumv1 "github.com/go-go-golems/agentforum/gen/proto/agentforum/v1"
	"github.com/go-go-golems/agentforum/internal/service"
	"github.com/go-go-golems/agentforum/internal/store"
)

// ── agents ────────────────────────────────────────────────────────────

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req agentforumv1.RegisterAgentRequest
	if err := decodeProtoJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", err.Error())
		return
	}
	agent, token, err := s.svc.Register(r.Context(), service.RegisterInput{
		Name:     req.GetName(),
		Metadata: structToMap(req.GetMetadata()),
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	_ = writeProtoJSON(w, http.StatusCreated, &agentforumv1.RegisterAgentResponse{
		SchemaVersion: 1,
		Agent:         agentToProto(agent),
		Token:         token,
	})
}

func (s *Server) handleGetMe(w http.ResponseWriter, r *http.Request) {
	_ = writeProtoJSON(w, http.StatusOK, &agentforumv1.GetMeResponse{
		SchemaVersion: 1,
		Agent:         agentToProto(agentFrom(r.Context())),
	})
}

func (s *Server) handleUpdateAgent(w http.ResponseWriter, r *http.Request) {
	var req agentforumv1.UpdateAgentRequest
	if err := decodeProtoJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", err.Error())
		return
	}
	agent, err := s.svc.UpdateMe(r.Context(), tokenFrom(r.Context()), service.UpdateMeInput{
		Metadata:    structToMap(req.GetMetadata()),
		HasMetadata: req.GetMetadata() != nil,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	_ = writeProtoJSON(w, http.StatusOK, &agentforumv1.UpdateAgentResponse{
		SchemaVersion: 1,
		Agent:         agentToProto(agent),
	})
}

func (s *Server) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	agent, err := s.svc.GetAgentByName(r.Context(), r.PathValue("name"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	_ = writeProtoJSON(w, http.StatusOK, &agentforumv1.GetAgentResponse{
		SchemaVersion: 1,
		Agent:         agentToProto(agent),
	})
}

// ── subforums ────────────────────────────────────────────────────────

func (s *Server) handleListSubforums(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sfs, err := s.svc.ListSubforums(ctx)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	counts, err := s.svc.Store().SubforumThreadCounts(ctx)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	watched, err := s.svc.Store().WatchedSubforumKeys(ctx, agentFrom(ctx).ID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	watching := map[string]bool{}
	for _, k := range watched {
		watching[k] = true
	}
	out := &agentforumv1.SubforumList{SchemaVersion: 1}
	for _, sf := range sfs {
		out.Subforums = append(out.Subforums,
			subforumToProto(sf, counts[sf.Key], watching[sf.Key]))
	}
	_ = writeProtoJSON(w, http.StatusOK, out)
}

func (s *Server) handleCreateSubforum(w http.ResponseWriter, r *http.Request) {
	var req agentforumv1.CreateSubforumRequest
	if err := decodeProtoJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", err.Error())
		return
	}
	sf, err := s.svc.CreateSubforum(r.Context(), agentFrom(r.Context()), service.CreateSubforumInput{
		Key:         req.GetKey(),
		Title:       req.GetTitle(),
		Description: req.GetDescription(),
		Metadata:    structToMap(req.GetMetadata()),
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	_ = writeProtoJSON(w, http.StatusCreated, &agentforumv1.CreateSubforumResponse{
		SchemaVersion: 1,
		Subforum:      subforumToProto(sf, 0, false),
	})
}

func (s *Server) handleGetSubforum(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sf, err := s.svc.GetSubforum(ctx, r.PathValue("key"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	counts, err := s.svc.Store().SubforumThreadCounts(ctx)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	watching, err := s.svc.Store().IsWatchingSubforum(ctx, agentFrom(ctx).ID, sf.Key)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	_ = writeProtoJSON(w, http.StatusOK, &agentforumv1.GetSubforumResponse{
		SchemaVersion: 1,
		Subforum:      subforumToProto(sf, counts[sf.Key], watching),
	})
}

func (s *Server) handleWatchSubforum(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.WatchSubforum(r.Context(), agentFrom(r.Context()), r.PathValue("key")); err != nil {
		writeServiceError(w, err)
		return
	}
	_ = writeProtoJSON(w, http.StatusOK, &agentforumv1.WatchSubforumResponse{
		SchemaVersion: 1, Watching: true,
	})
}

func (s *Server) handleUnwatchSubforum(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.UnwatchSubforum(r.Context(), agentFrom(r.Context()), r.PathValue("key")); err != nil {
		writeServiceError(w, err)
		return
	}
	_ = writeProtoJSON(w, http.StatusOK, &agentforumv1.WatchSubforumResponse{
		SchemaVersion: 1, Watching: false,
	})
}

// ── threads ──────────────────────────────────────────────────────────

func (s *Server) handleCreateThread(w http.ResponseWriter, r *http.Request) {
	var req agentforumv1.CreateThreadRequest
	if err := decodeProtoJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", err.Error())
		return
	}
	idem := req.GetIdempotencyKey()
	if idem == "" {
		idem = r.Header.Get("Idempotency-Key")
	}
	var body string
	var meta map[string]any
	if ip := req.GetInitialPost(); ip != nil {
		body = ip.GetBody()
		meta = structToMap(ip.GetMetadata())
	}
	agent := agentFrom(r.Context())
	thread, post, err := s.svc.CreateThread(r.Context(), agent, service.CreateThreadInput{
		Subforum:       r.PathValue("key"),
		Title:          req.GetTitle(),
		Body:           body,
		PostMetadata:   meta,
		Metadata:       structToMap(req.GetMetadata()),
		Watch:          req.GetWatch(),
		IdempotencyKey: idem,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	_ = writeProtoJSON(w, http.StatusCreated, &agentforumv1.CreateThreadResponse{
		SchemaVersion: 1,
		Thread: threadToProto(thread, store.ThreadStats{
			PostCount:  1,
			LastPostAt: post.CreatedAt,
		}, req.GetWatch(), true),
		InitialPost: postToProto(post, agent.Name),
	})
}

func (s *Server) handleListThreads(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	agent := agentFrom(ctx)
	threads, err := s.svc.ListThreads(ctx, agent, service.ListThreadsOptions{
		Involved: r.URL.Query().Get("participating") == "true",
		Watching: r.URL.Query().Get("watching") == "true",
		Subforum: r.URL.Query().Get("subforum"),
		Limit:    atoiDefault(r.URL.Query().Get("limit"), 0),
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	ids := make([]string, len(threads))
	for i, t := range threads {
		ids[i] = t.ID
	}
	views, err := s.threadView(ctx, agent.ID, ids)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := &agentforumv1.ThreadList{SchemaVersion: 1}
	for _, t := range threads {
		out.Threads = append(out.Threads,
			threadToProto(t, views.stats[t.ID], views.watching[t.ID], views.participating[t.ID]))
	}
	_ = writeProtoJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetThread(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	agent := agentFrom(ctx)
	thread, err := s.svc.GetThread(ctx, r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	views, err := s.threadView(ctx, agent.ID, []string{thread.ID})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	_ = writeProtoJSON(w, http.StatusOK, &agentforumv1.GetThreadResponse{
		SchemaVersion: 1,
		Thread: threadToProto(thread, views.stats[thread.ID],
			views.watching[thread.ID], views.participating[thread.ID]),
	})
}

func (s *Server) handleWatchThread(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.WatchThread(r.Context(), agentFrom(r.Context()), r.PathValue("id")); err != nil {
		writeServiceError(w, err)
		return
	}
	_ = writeProtoJSON(w, http.StatusOK, &agentforumv1.WatchThreadResponse{
		SchemaVersion: 1, Watching: true,
	})
}

func (s *Server) handleUnwatchThread(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.UnwatchThread(r.Context(), agentFrom(r.Context()), r.PathValue("id")); err != nil {
		writeServiceError(w, err)
		return
	}
	_ = writeProtoJSON(w, http.StatusOK, &agentforumv1.WatchThreadResponse{
		SchemaVersion: 1, Watching: false,
	})
}

// ── posts ────────────────────────────────────────────────────────────

func (s *Server) handleListPosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	posts, err := s.svc.ListPosts(ctx, r.PathValue("id"), "",
		atoiDefault(r.URL.Query().Get("limit"), 0))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	authorIDs := make([]string, 0, len(posts))
	for _, p := range posts {
		authorIDs = append(authorIDs, p.AuthorID)
	}
	names, err := s.svc.Store().AgentNames(ctx, authorIDs)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := &agentforumv1.PostList{SchemaVersion: 1}
	for _, p := range posts {
		out.Posts = append(out.Posts, postToProto(p, names[p.AuthorID]))
	}
	_ = writeProtoJSON(w, http.StatusOK, out)
}

func (s *Server) handleCreatePost(w http.ResponseWriter, r *http.Request) {
	var req agentforumv1.CreatePostRequest
	if err := decodeProtoJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", err.Error())
		return
	}
	idem := req.GetIdempotencyKey()
	if idem == "" {
		idem = r.Header.Get("Idempotency-Key")
	}
	agent := agentFrom(r.Context())
	post, err := s.svc.CreatePost(r.Context(), agent, service.CreatePostInput{
		ThreadID:       r.PathValue("id"),
		Body:           req.GetBody(),
		ReplyTo:        req.GetReplyTo(),
		Metadata:       structToMap(req.GetMetadata()),
		IdempotencyKey: idem,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	_ = writeProtoJSON(w, http.StatusCreated, &agentforumv1.CreatePostResponse{
		SchemaVersion: 1,
		Post:          postToProto(post, agent.Name),
	})
}

// ── events ───────────────────────────────────────────────────────────

func (s *Server) handlePollEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	cursor, err := strconv.ParseInt(orDefault(q.Get("cursor"), "0"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", "cursor must be an integer")
		return
	}
	wait := atoiDefault(q.Get("wait"), 0)
	if wait > maxWaitSeconds {
		wait = maxWaitSeconds
	}
	pollCtx := ctx
	if wait > 0 {
		var cancel context.CancelFunc
		pollCtx, cancel = context.WithTimeout(ctx, time.Duration(wait)*time.Second)
		defer cancel()
	}

	events, nextCursor, err := s.svc.PollEvents(pollCtx, agentFrom(ctx), service.PollEventsOptions{
		Cursor: cursor,
		Wait:   time.Duration(wait) * time.Second,
		Scope:  q.Get("scope"), // comma-separated: involved,watching,watched-subforums
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	// Denormalize actor names and thread titles for inbox display in two
	// batched queries (never N+1).
	actorIDs := make([]string, 0, len(events))
	threadIDs := make([]string, 0, len(events))
	for _, ev := range events {
		actorIDs = append(actorIDs, ev.ActorID)
		threadIDs = append(threadIDs, ev.ThreadID)
	}
	names, err := s.svc.Store().AgentNames(ctx, actorIDs)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	titles, err := s.svc.Store().ThreadTitles(ctx, threadIDs)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	out := &agentforumv1.PollEventsResponse{
		SchemaVersion: 1,
		NextCursor:    nextCursor,
	}
	for _, ev := range events {
		out.Events = append(out.Events, eventToProto(ev, names[ev.ActorID], titles[ev.ThreadID]))
	}
	_ = writeProtoJSON(w, http.StatusOK, out)
}

func (s *Server) handleAckEvents(w http.ResponseWriter, r *http.Request) {
	var req agentforumv1.AckEventsRequest
	if err := decodeProtoJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", err.Error())
		return
	}
	if err := s.svc.AckEvents(r.Context(), agentFrom(r.Context()), req.GetThroughSequence()); err != nil {
		writeServiceError(w, err)
		return
	}
	_ = writeProtoJSON(w, http.StatusOK, &agentforumv1.AckEventsResponse{
		SchemaVersion: 1, ThroughSequence: req.GetThroughSequence(),
	})
}

// ── search ───────────────────────────────────────────────────────────

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	var req agentforumv1.SearchRequest
	if err := decodeProtoJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", err.Error())
		return
	}
	in := service.SearchInput{
		Text:  req.GetText(),
		Limit: int(req.GetLimit()),
	}
	if len(req.GetSubforums()) > 0 {
		in.Subforum = req.GetSubforums()[0] // single-subforum filter for now
	}
	for k, v := range req.GetMetadata() {
		in.Terms = append(in.Terms, service.TermFilter{Keys: []string{k}, Value: v})
	}
	if ca := req.GetCreatedAfter(); ca != "" {
		t, err := time.Parse(time.RFC3339, ca)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_argument", "createdAfter must be RFC 3339")
			return
		}
		in.CreatedAfter = t
	}
	results, err := s.svc.Search(r.Context(), in, req.GetEntityTypes())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	out := &agentforumv1.SearchResponse{SchemaVersion: 1}
	if results != nil {
		for _, t := range results.Threads {
			out.Hits = append(out.Hits, &agentforumv1.SearchHit{
				EntityType: "thread",
				Thread:     threadToProto(t, store.ThreadStats{}, false, false),
			})
		}
		for _, p := range results.Posts {
			out.Hits = append(out.Hits, &agentforumv1.SearchHit{
				EntityType: "post",
				Post:       postToProto(p, ""),
			})
		}
	}
	_ = writeProtoJSON(w, http.StatusOK, out)
}

// ── helpers ──────────────────────────────────────────────────────────

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func orDefault(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}
