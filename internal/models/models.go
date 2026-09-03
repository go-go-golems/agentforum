// Package models defines the plain data structures passed between the store,
// service, and CLI layers. Types here carry JSON tags for serialization and
// intentionally have no methods and no dependencies on store or service code,
// so the layers stay decoupled (see design doc §3.1).
package models

import "time"

// Agent is an identity. Token is populated only at registration; everywhere
// else the caller works from TokenHash-free copies.
type Agent struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	DisplayName string         `json:"display_name"`
	Bio         string         `json:"bio"`
	Status      string         `json:"status"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	TokenHash   string         `json:"-"` // never serialized
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// Subforum is a named bucket of threads. Key is the user-chosen identifier.
type Subforum struct {
	Key         string         `json:"key"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatorID   string         `json:"creator_id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// Thread lives in exactly one subforum. Its opening post is created
// atomically with the thread and returned alongside it where relevant.
type Thread struct {
	ID        string         `json:"id"`
	Subforum  string         `json:"subforum"`
	Title     string         `json:"title"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatorID string         `json:"creator_id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// Post belongs to one thread. ReplyTo is empty for a top-level post.
type Post struct {
	ID        string         `json:"id"`
	ThreadID  string         `json:"thread_id"`
	AuthorID  string         `json:"author_id"`
	Body      string         `json:"body"`
	ReplyTo   string         `json:"reply_to,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// EventType is the discriminator on the append-only events table.
type EventType string

const (
	EventThreadCreated EventType = "thread.created"
	EventPostCreated   EventType = "post.created"
)

// Event is one row in the unified inbox log. Reason is computed at query time
// for the requesting agent and is therefore not stored.
type Event struct {
	Sequence  int64     `json:"sequence"`
	Type      EventType `json:"type"`
	ActorID   string    `json:"actor_id"`
	ThreadID  string    `json:"thread_id"`
	PostID    string    `json:"post_id,omitempty"`
	Subforum  string    `json:"subforum,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	// Derived fields, populated only when an event is delivered to a specific
	// agent (see service.PollEvents).
	Reason      string `json:"reason,omitempty"`
	ThreadTitle string `json:"-"`
}

// Participant links an agent to a thread it has posted in.
type Participant struct {
	AgentID    string    `json:"agent_id"`
	ThreadID   string    `json:"thread_id"`
	LastPostAt time.Time `json:"last_post_at"`
}

// MetadataTerm is one flattened (key,value) projection of an entity's JSON
// metadata, used for filtering (see design doc §7).
type MetadataTerm struct {
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	Key        string `json:"key"`
	Value      string `json:"value"`
}

// EventReason values delivered to clients.
const (
	ReasonParticipating   = "participating"
	ReasonWatching        = "watching"
	ReasonWatchedSubforum = "watched_subforum"
)
