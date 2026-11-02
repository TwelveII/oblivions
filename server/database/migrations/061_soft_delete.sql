-- 061_soft_delete.sql
-- Phase 16 P2: Soft-delete with 30-day TTL for users and servers.
--
-- Users:
--   deleted_at        — null = active, set = soft-deleted (recoverable) or tombstone
--   deleted_by_admin  — 1 if admin initiated, 0 if user self-deleted
--   is_hard_deleted   — 1 = anonymized tombstone (not recoverable), 0 = soft-deleted (recoverable)
-- Servers:
--   deleted_at        — null = active, set = soft-deleted (30-day TTL before hard delete)
--   deleted_by        — userID who initiated the delete (owner or admin)
--   deleted_by_admin  — 1 if admin initiated (owner cannot restore), 0 if owner-initiated
--
-- Tombstone hard-delete (users):
--   Username renamed to deleted_<userID>, email cleared, password_hash cleared,
--   personal data wiped. Row stays so messages.user_id keeps referential integrity.
--
-- Migration uses ALTER TABLE ADD COLUMN — no table recreation needed.

ALTER TABLE servers ADD COLUMN deleted_at DATETIME;
ALTER TABLE servers ADD COLUMN deleted_by TEXT REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE servers ADD COLUMN deleted_by_admin INTEGER NOT NULL DEFAULT 0;

ALTER TABLE users ADD COLUMN deleted_at DATETIME;
ALTER TABLE users ADD COLUMN deleted_by_admin INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN is_hard_deleted INTEGER NOT NULL DEFAULT 0;

-- Partial indexes for "find soft-deleted X" queries (worker scan, owner restore list).
CREATE INDEX IF NOT EXISTS idx_servers_deleted_at ON servers(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at) WHERE deleted_at IS NOT NULL;
