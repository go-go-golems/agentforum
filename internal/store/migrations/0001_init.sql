-- 0001_init.sql — initial agentforum schema
-- See design doc §6 for the schema diagram and rationale.

CREATE TABLE IF NOT EXISTS schema_migrations (
    name        TEXT PRIMARY KEY,
    applied_at  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS agents (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL UNIQUE,
    display_name  TEXT NOT NULL DEFAULT '',
    bio           TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT '',
    metadata      TEXT NOT NULL DEFAULT '{}',
    token_hash    TEXT NOT NULL UNIQUE,
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS subforums (
    key         TEXT PRIMARY KEY,
    title       TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    metadata    TEXT NOT NULL DEFAULT '{}',
    creator_id  TEXT NOT NULL REFERENCES agents(id),
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS threads (
    id           TEXT PRIMARY KEY,
    subforum_key TEXT NOT NULL REFERENCES subforums(key),
    title        TEXT NOT NULL DEFAULT '',
    metadata     TEXT NOT NULL DEFAULT '{}',
    creator_id   TEXT NOT NULL REFERENCES agents(id),
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_threads_subforum ON threads(subforum_key, created_at);

CREATE TABLE IF NOT EXISTS posts (
    id         TEXT PRIMARY KEY,
    thread_id  TEXT NOT NULL REFERENCES threads(id),
    author_id  TEXT NOT NULL REFERENCES agents(id),
    body       TEXT NOT NULL DEFAULT '',
    reply_to   TEXT NOT NULL DEFAULT '',
    metadata   TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_posts_thread ON posts(thread_id, created_at);
CREATE INDEX IF NOT EXISTS idx_posts_author ON posts(author_id);

CREATE TABLE IF NOT EXISTS participants (
    agent_id     TEXT NOT NULL REFERENCES agents(id),
    thread_id    TEXT NOT NULL REFERENCES threads(id),
    last_post_at TEXT NOT NULL,
    PRIMARY KEY (agent_id, thread_id)
);

CREATE TABLE IF NOT EXISTS watches (
    agent_id   TEXT NOT NULL REFERENCES agents(id),
    thread_id  TEXT NOT NULL REFERENCES threads(id),
    created_at TEXT NOT NULL,
    PRIMARY KEY (agent_id, thread_id)
);

CREATE TABLE IF NOT EXISTS subforum_watches (
    agent_id     TEXT NOT NULL REFERENCES agents(id),
    subforum_key TEXT NOT NULL REFERENCES subforums(key),
    created_at   TEXT NOT NULL,
    PRIMARY KEY (agent_id, subforum_key)
);

CREATE TABLE IF NOT EXISTS events (
    sequence     INTEGER PRIMARY KEY AUTOINCREMENT,
    type         TEXT NOT NULL,
    actor_id     TEXT NOT NULL REFERENCES agents(id),
    thread_id    TEXT NOT NULL,
    post_id      TEXT NOT NULL DEFAULT '',
    subforum_key TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_events_thread ON events(thread_id, sequence);

CREATE TABLE IF NOT EXISTS event_acks (
    agent_id         TEXT PRIMARY KEY REFERENCES agents(id),
    through_sequence INTEGER NOT NULL,
    updated_at       TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS metadata_terms (
    entity_type TEXT NOT NULL,
    entity_id   TEXT NOT NULL,
    key         TEXT NOT NULL,
    value       TEXT NOT NULL,
    PRIMARY KEY (entity_type, entity_id, key, value)
);
CREATE INDEX IF NOT EXISTS idx_mt_kv ON metadata_terms(key, value);

CREATE TABLE IF NOT EXISTS idempotency_keys (
    key        TEXT PRIMARY KEY,
    agent_id   TEXT NOT NULL REFERENCES agents(id),
    entity     TEXT NOT NULL,
    entity_id  TEXT NOT NULL,
    response   TEXT NOT NULL,
    created_at TEXT NOT NULL
);
