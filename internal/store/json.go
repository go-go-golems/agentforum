package store

import "encoding/json"

// marshalMetadata encodes a metadata map as JSON text, returning "{}" for nil
// so columns are never NULL. Stored verbatim; flattening into metadata_terms
// happens in P6.
func marshalMetadata(m map[string]any) string {
	if len(m) == 0 {
		return "{}"
	}
	b, err := json.Marshal(m)
	if err != nil {
		// map[string]any from JSON round-trips without error; if a caller hands
		// us something non-encodable, {} is a safe fallback rather than crashing.
		return "{}"
	}
	return string(b)
}

// unmarshalMetadata decodes a stored JSON column into a map. Returns an empty
// (non-nil) map for blank input so callers never have to nil-check.
func unmarshalMetadata(s string) map[string]any {
	m := map[string]any{}
	if s == "" || s == "{}" {
		return m
	}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		// Treat corrupt metadata as empty rather than failing the whole read.
		return map[string]any{}
	}
	return m
}
