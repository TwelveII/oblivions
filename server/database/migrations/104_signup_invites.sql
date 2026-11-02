ALTER TABLE users ADD COLUMN invite_limit INTEGER NOT NULL DEFAULT 1;
ALTER TABLE users ADD COLUMN invite_codes_created INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN invited_by_user_id TEXT REFERENCES users(id);

CREATE TABLE signup_invite_codes (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    code_hash TEXT NOT NULL UNIQUE,
    inviter_user_id TEXT NOT NULL REFERENCES users(id),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME,
    used_at DATETIME,
    used_by_user_id TEXT REFERENCES users(id),
    revoked_at DATETIME
);

CREATE INDEX idx_signup_invite_codes_inviter
    ON signup_invite_codes(inviter_user_id);

CREATE INDEX idx_signup_invite_codes_status
    ON signup_invite_codes(used_at, revoked_at, expires_at);

CREATE INDEX idx_users_invited_by
    ON users(invited_by_user_id);
