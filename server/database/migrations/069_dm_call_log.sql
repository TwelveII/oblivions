-- DM call log: server-written system messages recording P2P call events
-- (missed / declined / completed). message_type distinguishes them from normal
-- text/E2EE messages; call_meta holds the JSON payload (caller_id, call_type,
-- outcome, duration_sec). Call logs are plaintext (encryption_version=0) — they
-- carry call metadata, not private content.

ALTER TABLE dm_messages ADD COLUMN message_type TEXT NOT NULL DEFAULT 'text';
ALTER TABLE dm_messages ADD COLUMN call_meta TEXT;
