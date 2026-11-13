package postgres

import (
	"context"
	"encoding/json"
	"fmt"
)

func (s *Store) Write(
	ctx context.Context,
	actorID, action, entityType, entityID string,
	details map[string]any,
) error {
	if details == nil {
		details = map[string]any{}
	}
	encoded, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("encode audit details: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO audit_log (actor_id, action, entity_type, entity_id, details)
		VALUES (NULLIF($1, '')::uuid, $2, $3, $4, $5)
	`, actorID, action, entityType, entityID, encoded)
	if err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}
	return nil
}
