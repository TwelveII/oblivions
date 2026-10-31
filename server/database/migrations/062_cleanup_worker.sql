-- 062_cleanup_worker.sql
-- Phase 16 P3: Embedded daily cleanup worker.
--
-- Tables:
--   cleanup_state         — single-row tracker for last successful run timestamp.
--                            Lets the in-process scheduler skip a tick if the daemon
--                            was restarted within the last 24h (avoids double-runs).
--   cleanup_failed_files  — retry queue for file deletes that failed on disk.
--                            Daily worker drains this with exponential backoff
--                            (2^retry_count minutes), gives up after 7 attempts.

CREATE TABLE IF NOT EXISTS cleanup_state (
    id          INTEGER PRIMARY KEY CHECK (id = 1),
    last_run_at DATETIME
);
INSERT OR IGNORE INTO cleanup_state (id, last_run_at) VALUES (1, NULL);

CREATE TABLE IF NOT EXISTS cleanup_failed_files (
    id             TEXT PRIMARY KEY,
    file_url       TEXT NOT NULL UNIQUE,
    failure_reason TEXT NOT NULL DEFAULT '',
    retry_count    INTEGER NOT NULL DEFAULT 0,
    next_retry_at  DATETIME NOT NULL,
    created_at     DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_cleanup_failed_next_retry ON cleanup_failed_files(next_retry_at);
