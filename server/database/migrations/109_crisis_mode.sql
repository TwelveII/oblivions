-- 109_crisis_mode.sql
-- Explicitly separate conversation retention cleanup from routine maintenance.
CREATE TABLE IF NOT EXISTS crisis_mode_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    enabled INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0, 1)),
    site_enabled INTEGER NOT NULL DEFAULT 1 CHECK (site_enabled IN (0, 1)),
    last_run_at DATETIME
);
INSERT OR IGNORE INTO crisis_mode_state (id, enabled, site_enabled, last_run_at) VALUES (1, 0, 1, NULL);
