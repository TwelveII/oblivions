-- token_version is bumped to invalidate every JWT (access + file) ever issued
-- to a user. Used by password change and admin force-logout. Each JWT carries
-- a tv claim; file/api validation rejects on mismatch.

ALTER TABLE users ADD COLUMN token_version INTEGER NOT NULL DEFAULT 0;
