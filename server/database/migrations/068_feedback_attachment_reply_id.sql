-- Backfill reply_id on older DBs where feedback_attachments was created without it
-- (early 049 lacked the column; migration bootstrap then marked it applied unrun).
-- The runner skips "duplicate column name", so this is a no-op where it already exists.
ALTER TABLE feedback_attachments ADD COLUMN reply_id TEXT REFERENCES feedback_replies(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_feedback_attachments_reply ON feedback_attachments(reply_id);
