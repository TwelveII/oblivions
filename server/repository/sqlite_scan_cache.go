package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/akinalp/mqvi/database"
)

type sqliteScanHashCacheRepo struct {
	db database.TxQuerier
}

func NewSQLiteScanHashCacheRepo(db database.TxQuerier) ScanHashCacheRepository {
	return &sqliteScanHashCacheRepo{db: db}
}

func (r *sqliteScanHashCacheRepo) Get(ctx context.Context, sha256 string) (*ScanHashCacheEntry, error) {
	var entry ScanHashCacheEntry
	err := r.db.QueryRowContext(ctx,
		`SELECT sha256, status, signature, scanned_at FROM scan_hash_cache WHERE sha256 = ?`,
		sha256,
	).Scan(&entry.SHA256, &entry.Status, &entry.Signature, &entry.ScannedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan cache get: %w", err)
	}
	return &entry, nil
}

func (r *sqliteScanHashCacheRepo) Upsert(ctx context.Context, sha256, status string, signature *string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO scan_hash_cache (sha256, status, signature, scanned_at)
		 VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(sha256) DO UPDATE SET
		   status = excluded.status,
		   signature = excluded.signature,
		   scanned_at = excluded.scanned_at`,
		sha256, status, signature,
	)
	if err != nil {
		return fmt.Errorf("scan cache upsert: %w", err)
	}
	return nil
}

func (r *sqliteScanHashCacheRepo) DeleteBefore(ctx context.Context, status, cutoff string) (int, error) {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM scan_hash_cache WHERE status = ? AND scanned_at < ?`,
		status, cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("scan cache delete before: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("scan cache delete rows affected: %w", err)
	}
	return int(n), nil
}

var _ ScanHashCacheRepository = (*sqliteScanHashCacheRepo)(nil)
