package repository

import (
	"context"
	"time"

	"github.com/akinalp/mqvi/models"
)

// CleanupRepository persists state for the daily embedded cleanup worker:
//   - cleanup_state: single-row last-run timestamp (lets the in-process scheduler
//     skip a tick if the daemon was restarted within the last 24h).
//   - cleanup_failed_files: retry queue for file deletes that failed on disk.
type CleanupRepository interface {
	// GetLastRunAt returns the timestamp of the previous successful worker run, or nil
	// if the worker has never finished a run on this DB.
	GetLastRunAt(ctx context.Context) (*time.Time, error)
	// SetLastRunAt stamps the moment a run finished so future startups can decide
	// whether to fire immediately or wait for the next daily window.
	SetLastRunAt(ctx context.Context, t time.Time) error

	// EnqueueFailedFile records a file delete that failed on disk so the next worker
	// run can retry it. Idempotent on file_url: re-queuing an already-tracked file
	// updates the failure reason but leaves retry_count untouched (only the worker
	// itself bumps retry_count via BumpRetry).
	EnqueueFailedFile(ctx context.Context, fileURL, reason string, nextRetryAt time.Time) error
	// ListDueRetries returns failed files whose backoff has elapsed AND whose
	// retry_count is below maxRetries (gives up silently past the cap).
	ListDueRetries(ctx context.Context, now time.Time, maxRetries int) ([]models.CleanupFailedFile, error)
	// BumpRetry increments retry_count and pushes next_retry_at out by the new backoff.
	// Called when a retry attempt fails again.
	BumpRetry(ctx context.Context, id string, nextRetryAt time.Time, reason string) error
	// DeleteFailedFile removes a row after a retry attempt finally succeeds.
	DeleteFailedFile(ctx context.Context, id string) error
}
