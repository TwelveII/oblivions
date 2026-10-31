-- 070_push_tokens.sql
-- Push notification registration tokens (FCM). Deliberately decoupled from the
-- E2EE user_devices table: E2EE is optional and most users have no user_devices
-- row, so push tokens live in their own table keyed on the FCM token itself
-- (one token per physical install). ON CONFLICT(token) reassigns a token to the
-- current user, covering the case where a device switches accounts.

CREATE TABLE IF NOT EXISTS push_tokens (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(8)))),
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    platform TEXT NOT NULL CHECK(platform IN ('android', 'ios', 'web')),
    device_label TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_seen_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_push_tokens_user ON push_tokens(user_id);
