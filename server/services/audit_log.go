package services

import (
	"context"
	"database/sql"
	"encoding/json"
)

// recordAuditLog stores administrator/security lifecycle events in the same searchable
// platform log stream as application diagnostics. Log records are intentionally short
// and must not include message bodies, credentials, tokens, or other secrets.
func recordAuditLog(ctx context.Context, db *sql.DB, actorID, targetID, serverID, message string, metadata map[string]string) {
	if db == nil {
		return
	}
	if metadata == nil { metadata = make(map[string]string) }
	if actorID != "" { metadata["actor_id"] = actorID }
	if targetID != "" { metadata["target_id"] = targetID }
	if serverID != "" { metadata["server_id"] = serverID }
	raw := "{}"
	if len(metadata) > 0 {
		if b, err := json.Marshal(metadata); err == nil {
			raw = string(b)
		}
	}
	var userID, sid any
	if actorID != "" { userID = actorID } else if targetID != "" { userID = targetID }
	if serverID != "" { sid = serverID }
	_, _ = db.ExecContext(ctx,
		`INSERT INTO app_logs (level, category, user_id, server_id, message, metadata)
		 VALUES ('info', 'audit', ?, ?, ?, ?)`,
		userID, sid, message, raw,
	)
}
