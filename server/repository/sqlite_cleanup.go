package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/akinalp/mqvi/database"
	"github.com/akinalp/mqvi/models"
	"github.com/akinalp/mqvi/pkg"
	"github.com/google/uuid"
)

type sqliteCleanupRepo struct {
	db database.TxQuerier
}

func NewSQLiteCleanupRepo(db database.TxQuerier) CleanupRepository {
	return &sqliteCleanupRepo{db: db}
}

func (r *sqliteCleanupRepo) GetLastRunAt(ctx context.Context) (*time.Time, error) {
	var raw sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT last_run_at FROM cleanup_state WHERE id = 1`,
	).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get last_run_at: %w", err)
	}
	if !raw.Valid {
		return nil, nil
	}
	return &raw.Time, nil
}

func (r *sqliteCleanupRepo) SetLastRunAt(ctx context.Context, t time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE cleanup_state SET last_run_at = ? WHERE id = 1`,
		t.UTC(),
	)
	if err != nil {
		return fmt.Errorf("set last_run_at: %w", err)
	}
	return nil
}

func (r *sqliteCleanupRepo) EnqueueFailedFile(ctx context.Context, fileURL, reason string, nextRetryAt time.Time) error {
	// ON CONFLICT(file_url): refresh the failure reason but keep retry_count and
	// next_retry_at frozen so a re-enqueue cannot reset the backoff schedule.
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO cleanup_failed_files (id, file_url, failure_reason, retry_count, next_retry_at)
		VALUES (?, ?, ?, 0, ?)
		ON CONFLICT(file_url) DO UPDATE SET failure_reason = excluded.failure_reason`,
		uuid.NewString(), fileURL, reason, nextRetryAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("enqueue failed file: %w", err)
	}
	return nil
}

func (r *sqliteCleanupRepo) ListDueRetries(ctx context.Context, now time.Time, maxRetries int) ([]models.CleanupFailedFile, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, file_url, failure_reason, retry_count, next_retry_at, created_at
		FROM cleanup_failed_files
		WHERE next_retry_at <= ? AND retry_count < ?
		ORDER BY next_retry_at ASC`,
		now.UTC(), maxRetries,
	)
	if err != nil {
		return nil, fmt.Errorf("list due retries: %w", err)
	}
	defer rows.Close()

	var out []models.CleanupFailedFile
	for rows.Next() {
		var f models.CleanupFailedFile
		if err := rows.Scan(&f.ID, &f.FileURL, &f.FailureReason, &f.RetryCount, &f.NextRetryAt, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan failed file: %w", err)
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate failed files: %w", err)
	}
	return out, nil
}

func (r *sqliteCleanupRepo) BumpRetry(ctx context.Context, id string, nextRetryAt time.Time, reason string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE cleanup_failed_files
		SET retry_count = retry_count + 1,
		    next_retry_at = ?,
		    failure_reason = ?
		WHERE id = ?`,
		nextRetryAt.UTC(), reason, id,
	)
	if err != nil {
		return fmt.Errorf("bump retry: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("bump retry rows affected: %w", err)
	}
	if affected == 0 {
		return pkg.ErrNotFound
	}
	return nil
}

func (r *sqliteCleanupRepo) DeleteFailedFile(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM cleanup_failed_files WHERE id = ?`, id,
	)
	if err != nil {
		return fmt.Errorf("delete failed file: %w", err)
	}
	return nil
}
