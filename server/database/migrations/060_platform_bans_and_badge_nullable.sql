-- 060_platform_bans_and_badge_nullable.sql
-- 1) Standalone platform_bans table: persists after user hard-delete.
--    Enforces email + username re-registration block (case-insensitive).
--    IP column stored for future enforcement expansion.
-- 2) Recreate badges/user_badges with nullable created_by/assigned_by
--    so hard-deleting an admin who created/assigned badges doesn't FK-fail.
-- Note: migration runner wraps this in a transaction automatically.

-- ─── platform_bans ───

CREATE TABLE IF NOT EXISTS platform_bans (
    id         TEXT PRIMARY KEY,
    email      TEXT,
    username   TEXT NOT NULL,
    user_id    TEXT NOT NULL UNIQUE,
    ip         TEXT,
    reason     TEXT NOT NULL DEFAULT '',
    banned_by  TEXT NOT NULL,
    banned_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_platform_bans_email ON platform_bans(email COLLATE NOCASE);
CREATE INDEX IF NOT EXISTS idx_platform_bans_username ON platform_bans(username COLLATE NOCASE);
CREATE INDEX IF NOT EXISTS idx_platform_bans_user_id ON platform_bans(user_id);

-- ─── Recreate badges + user_badges with nullable creator/assigner ───
-- Create new tables first, copy data, drop old tables in reverse-dependency
-- order (user_badges before badges to avoid CASCADE), then rename.

CREATE TABLE IF NOT EXISTS badges_new (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    icon       TEXT NOT NULL DEFAULT '',
    icon_type  TEXT NOT NULL DEFAULT 'builtin' CHECK(icon_type IN ('builtin', 'custom')),
    color1     TEXT NOT NULL DEFAULT '#5865F2',
    color2     TEXT,
    created_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

INSERT OR IGNORE INTO badges_new SELECT * FROM badges;

CREATE TABLE IF NOT EXISTS user_badges_new (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    badge_id    TEXT NOT NULL REFERENCES badges_new(id) ON DELETE CASCADE,
    assigned_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    assigned_at DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, badge_id)
);

INSERT OR IGNORE INTO user_badges_new SELECT * FROM user_badges;

DROP TABLE IF EXISTS user_badges;
DROP TABLE IF EXISTS badges;

ALTER TABLE badges_new RENAME TO badges;
ALTER TABLE user_badges_new RENAME TO user_badges;

CREATE INDEX IF NOT EXISTS idx_user_badges_user ON user_badges(user_id);
CREATE INDEX IF NOT EXISTS idx_user_badges_badge ON user_badges(badge_id);
