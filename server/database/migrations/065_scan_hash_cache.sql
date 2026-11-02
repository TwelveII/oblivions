CREATE TABLE IF NOT EXISTS scan_hash_cache (
    sha256 TEXT PRIMARY KEY,
    status TEXT NOT NULL CHECK (status IN ('clean', 'infected')),
    signature TEXT,
    scanned_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_scan_hash_cache_status_scanned
    ON scan_hash_cache(status, scanned_at);
