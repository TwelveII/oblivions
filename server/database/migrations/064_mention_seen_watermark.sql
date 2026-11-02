-- 064: Mention-seen watermark per (user, channel). Tuple (at, message_id) — id
-- breaks ties since SQLite DATETIME is second-precision.

ALTER TABLE channel_reads ADD COLUMN last_mention_seen_at DATETIME;
ALTER TABLE channel_reads ADD COLUMN last_mention_seen_message_id TEXT;
