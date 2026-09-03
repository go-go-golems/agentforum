package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// readBodyFile returns the contents of a file, or the inline body if no file is
// given. Used by `--body-file` / `--body` across thread and post commands.
func readBodyFile(bodyFile, body string) (string, error) {
	if bodyFile == "" {
		return body, nil
	}
	b, err := os.ReadFile(bodyFile)
	if err != nil {
		return "", fmt.Errorf("read body file: %w", err)
	}
	return string(b), nil
}

// loadMetadataFile reads a JSON object from a file into a map.
func loadMetadataFile(path string) (map[string]any, error) {
	m := map[string]any{}
	if path == "" {
		return m, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read metadata file: %w", err)
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return m, nil
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("parse metadata file %s: %w", path, err)
	}
	return m, nil
}

// applyMetaPairs applies repeated `--meta k=v` pairs onto a metadata map as
// string scalar values. Later pairs override earlier ones and any file values.
func applyMetaPairs(m map[string]any, pairs []string) error {
	for _, p := range pairs {
		k, v, found := strings.Cut(p, "=")
		if !found {
			return fmt.Errorf("invalid --meta %q: expected key=value", p)
		}
		if k == "" {
			return fmt.Errorf("invalid --meta %q: empty key", p)
		}
		m[k] = v
	}
	return nil
}

// buildMetadata composes a metadata map from an optional JSON file and repeated
// `--meta k=v` pairs (pairs override file values). This is the shared metadata
// constructor for register/subforum/thread/post create commands.
func buildMetadata(metadataFile string, metas []string) (map[string]any, error) {
	m, err := loadMetadataFile(metadataFile)
	if err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]any{}
	}
	if err := applyMetaPairs(m, metas); err != nil {
		return nil, err
	}
	return m, nil
}

// metadataJSON returns a compact JSON string for a metadata map, for clean
// structured output across table/json/jsonl formats (avoids printing a Go map).
func metadataJSON(m map[string]any) string {
	if len(m) == 0 {
		return ""
	}
	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(b)
}
