package service

import "errors"

// Sentinel errors for agentforum business rules. A future HTTP server maps these
// to status codes (401/404/409); the CLI surfaces them as command errors.
var (
	// ErrUnauthenticated means no token was supplied or it did not match an agent.
	ErrUnauthenticated = errors.New("agentforum: unauthenticated (bad or missing token)")
	// ErrNotFound means the named entity does not exist.
	ErrNotFound = errors.New("agentforum: not found")
	// ErrConflict means a uniqueness constraint was violated (e.g. agent name).
	ErrConflict = errors.New("agentforum: conflict")
	// ErrInvalidInput means the request was malformed (bad metadata, empty field).
	ErrInvalidInput = errors.New("agentforum: invalid input")
)
