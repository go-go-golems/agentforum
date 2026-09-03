// Package id generates prefixed, sortable identifiers and opaque auth tokens
// for agentforum entities.
//
// IDs are ULID-based (time-prefixed, Crockford base32, 26 chars) with a short
// human-readable prefix so log lines are scannable:
//
//	ag_<ulid>  agents
//	th_<ulid>  threads
//	po_<ulid>  posts
//
// Subforums do not get a ULID; their identifier is the user-chosen key (e.g.
// "engineering").
//
// Tokens are 32 random bytes from crypto/rand, base64url-encoded, and prefixed
// with "af_". Only the SHA-256 hash of a token is ever stored (see HashToken);
// the plaintext is returned to the caller exactly once at registration time.
package id

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"

	"github.com/oklog/ulid/v2"
)

const (
	agentPrefix  = "ag_"
	threadPrefix = "th_"
	postPrefix   = "po_"
	tokenPrefix  = "af_"
)

// NewAgentID returns a fresh agent identifier (ag_<ulid>).
func NewAgentID() string { return agentPrefix + ulid.Make().String() }

// NewThreadID returns a fresh thread identifier (th_<ulid>).
func NewThreadID() string { return threadPrefix + ulid.Make().String() }

// NewPostID returns a fresh post identifier (po_<ulid>).
func NewPostID() string { return postPrefix + ulid.Make().String() }

// NewToken returns a fresh opaque bearer token (af_<32 random bytes, base64url>).
// Use HashToken to derive the value to persist.
func NewToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand should never fail on Linux. If it does, the process is in
		// a state where continuing would be worse than stopping.
		panic("agentforum/id: crypto/rand unavailable: " + err.Error())
	}
	return tokenPrefix + base64.RawURLEncoding.EncodeToString(b)
}

// HashToken returns the lowercase hex SHA-256 of a token. This is the only
// representation that should be stored durably.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
