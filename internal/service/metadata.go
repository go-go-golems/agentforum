package service

import (
	"encoding/json"
	"fmt"
	"regexp"
)

const (
	maxMetadataBytes = 64 * 1024
	maxMetadataDepth = 8
)

var keySegmentRe = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// validateMetadata enforces the design-doc limits (§7.3): total size, key
// shape, reserved leading-underscore keys, and nesting depth. It is called on
// every write that accepts user metadata.
func validateMetadata(m map[string]any) error {
	if len(m) == 0 {
		return nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("%w: metadata is not JSON-encodable", ErrInvalidInput)
	}
	if len(b) > maxMetadataBytes {
		return fmt.Errorf("%w: metadata exceeds %d bytes", ErrInvalidInput, maxMetadataBytes)
	}
	return walkMetadata(m, "", 1)
}

func walkMetadata(v any, prefix string, depth int) error {
	if depth > maxMetadataDepth {
		return fmt.Errorf("%w: metadata nests deeper than %d levels", ErrInvalidInput, maxMetadataDepth)
	}
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			if k == "" {
				return fmt.Errorf("%w: empty metadata key", ErrInvalidInput)
			}
			if k[0] == '_' {
				return fmt.Errorf("%w: metadata key %q is reserved", ErrInvalidInput, k)
			}
			if !keySegmentRe.MatchString(k) {
				return fmt.Errorf("%w: metadata key %q must match %s", ErrInvalidInput, k, keySegmentRe.String())
			}
			if err := walkMetadata(child, k, depth+1); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range t {
			if err := walkMetadata(child, prefix, depth+1); err != nil {
				return err
			}
		}
	case string, bool, float64, int, int64, nil:
		// scalars are fine
	default:
		return fmt.Errorf("%w: metadata value of type %T is not allowed", ErrInvalidInput, v)
	}
	return nil
}
