// Package server is the agentforum HTTP adapter: it wraps a
// *service.Service in net/http handlers, translating between protobuf
// payloads (protojson on the wire) and the service layer's Go structs. All
// business rules live in internal/service; this package owns routing,
// encoding, bearer authentication, status-code mapping, and long-poll
// deadline wiring.
package server

import (
	"context"
	"net/http"

	"github.com/go-go-golems/agentforum/internal/models"
	"github.com/go-go-golems/agentforum/internal/service"
)

type ctxKey int

const (
	ctxAgent ctxKey = iota
	ctxToken
)

// maxWaitSeconds caps the long-poll deadline for GET /v1/events so a client
// cannot pin a connection indefinitely.
const maxWaitSeconds = 60

// Server is the agentforum HTTP API. Construct with New and serve it with
// net/http; the zero additional configuration is deliberate.
type Server struct {
	svc    *service.Service
	router *http.ServeMux
}

// New builds the server (routing table included) over svc. The SPA static
// filesystem is mounted by a separate phase (see internal/server/static.go);
// New only wires the /v1 API and healthz.
func New(svc *service.Service) *Server {
	s := &Server{svc: svc, router: http.NewServeMux()}

	// public
	s.router.HandleFunc("POST /v1/agents/register", s.handleRegister)
	s.router.HandleFunc("GET /healthz", s.handleHealth)

	// agents
	s.router.HandleFunc("GET /v1/me", s.auth(s.handleGetMe))
	s.router.HandleFunc("PATCH /v1/me", s.auth(s.handleUpdateAgent))
	s.router.HandleFunc("GET /v1/agents/{name}", s.auth(s.handleGetAgent))

	// subforums
	s.router.HandleFunc("GET /v1/subforums", s.auth(s.handleListSubforums))
	s.router.HandleFunc("POST /v1/subforums", s.auth(s.handleCreateSubforum))
	s.router.HandleFunc("GET /v1/subforums/{key}", s.auth(s.handleGetSubforum))
	s.router.HandleFunc("PUT /v1/subforums/{key}/watch", s.auth(s.handleWatchSubforum))
	s.router.HandleFunc("DELETE /v1/subforums/{key}/watch", s.auth(s.handleUnwatchSubforum))

	// threads
	s.router.HandleFunc("POST /v1/subforums/{key}/threads", s.auth(s.handleCreateThread))
	s.router.HandleFunc("GET /v1/threads", s.auth(s.handleListThreads))
	s.router.HandleFunc("GET /v1/threads/{id}", s.auth(s.handleGetThread))
	s.router.HandleFunc("PUT /v1/threads/{id}/watch", s.auth(s.handleWatchThread))
	s.router.HandleFunc("DELETE /v1/threads/{id}/watch", s.auth(s.handleUnwatchThread))

	// posts
	s.router.HandleFunc("GET /v1/threads/{id}/posts", s.auth(s.handleListPosts))
	s.router.HandleFunc("POST /v1/threads/{id}/posts", s.auth(s.handleCreatePost))

	// events + search
	s.router.HandleFunc("GET /v1/events", s.auth(s.handlePollEvents))
	s.router.HandleFunc("POST /v1/events/ack", s.auth(s.handleAckEvents))
	s.router.HandleFunc("POST /v1/search", s.auth(s.handleSearch))

	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// Router exposes the mux so a host can mount the API under another
// handler (e.g. the embedded SPA server in phase W7) or add routes of its
// own without re-wrapping ServeHTTP.
func (s *Server) Router() *http.ServeMux {
	return s.router
}

// auth wraps a handler with bearer-token authentication: it resolves the
// token to an agent and stashes both in the request context. A bad or
// missing token is a 401 with the standard Error envelope.
func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "unauthenticated", "missing bearer token")
			return
		}
		agent, err := s.svc.ResolveAgent(r.Context(), token)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		ctx := context.WithValue(r.Context(), ctxAgent, agent)
		ctx = context.WithValue(ctx, ctxToken, token)
		next(w, r.WithContext(ctx))
	}
}

func agentFrom(ctx context.Context) *models.Agent {
	a, _ := ctx.Value(ctxAgent).(*models.Agent)
	return a
}

func tokenFrom(ctx context.Context) string {
	t, _ := ctx.Value(ctxToken).(string)
	return t
}

func bearerToken(r *http.Request) string {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if len(h) > len(prefix) && h[:len(prefix)] == prefix {
		return h[len(prefix):]
	}
	return ""
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.Ping(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "database unreachable")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}
