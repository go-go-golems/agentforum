package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-go-golems/agentforum/internal/models"
)

// flattenMetadata walks a metadata map and appends one MetadataTerm per scalar
// leaf, using dotted keys for nested objects and repeating the key for array
// elements (see design doc §7.2). nil leaves are skipped.
func flattenMetadata(v any, prefix string, out *[]models.MetadataTerm) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			flattenMetadata(child, key, out)
		}
	case []any:
		// Array of scalars → same key, many rows; array of objects → dotted
		// keys per field (each object flattened under the array's key).
		for _, child := range t {
			flattenMetadata(child, prefix, out)
		}
	case nil:
		// skip null leaves
	case string:
		*out = append(*out, models.MetadataTerm{Key: prefix, Value: t})
	case bool:
		*out = append(*out, models.MetadataTerm{Key: prefix, Value: fmt.Sprintf("%t", t)})
	case float64:
		*out = append(*out, models.MetadataTerm{Key: prefix, Value: formatNumber(t)})
	case int:
		*out = append(*out, models.MetadataTerm{Key: prefix, Value: fmt.Sprintf("%d", t)})
	case int64:
		*out = append(*out, models.MetadataTerm{Key: prefix, Value: fmt.Sprintf("%d", t)})
	default:
		*out = append(*out, models.MetadataTerm{Key: prefix, Value: fmt.Sprintf("%v", t)})
	}
}

// formatNumber renders a JSON number without trailing zeros (json.Unmarshal
// gives float64 for all numbers).
func formatNumber(f float64) string {
	s := fmt.Sprintf("%g", f)
	return s
}

// indexMetadataTermsTx replaces the flattened terms for an entity within a
// transaction. Delete-then-insert keeps the index in sync with the canonical
// JSON column on every write.
func indexMetadataTermsTx(ctx context.Context, tx dbtx, entityType, entityID string, m map[string]any) error {
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM metadata_terms WHERE entity_type = ? AND entity_id = ?",
		entityType, entityID); err != nil {
		return fmt.Errorf("store: clear metadata terms: %w", err)
	}
	if len(m) == 0 {
		return nil
	}
	var terms []models.MetadataTerm
	flattenMetadata(m, "", &terms)
	for _, term := range terms {
		if term.Key == "" || strings.TrimSpace(term.Value) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO metadata_terms
(entity_type, entity_id, key, value) VALUES (?,?,?,?)`,
			entityType, entityID, term.Key, term.Value); err != nil {
			return fmt.Errorf("store: insert metadata term: %w", err)
		}
	}
	return nil
}

// IndexMetadataTerms is the non-transactional variant for a future reindex
// command (P6). Most writes use indexMetadataTermsTx inside their tx.
func (s *Store) IndexMetadataTerms(ctx context.Context, entityType, entityID string, m map[string]any) error {
	return indexMetadataTermsTx(ctx, s.db, entityType, entityID, m)
}
