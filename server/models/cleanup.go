package models

import "time"

// CleanupFailedFile represents a file delete that previously failed on disk and
// is queued for retry by the daily cleanup worker. Backoff is applied via NextRetryAt
// (2^retry_count minutes); the worker gives up after MaxCleanupRetries attempts.
type CleanupFailedFile struct {
	ID            string    `json:"id"`
	FileURL       string    `json:"fileUrl"`
	FailureReason string    `json:"failureReason"`
	RetryCount    int       `json:"retryCount"`
	NextRetryAt   time.Time `json:"nextRetryAt"`
	CreatedAt     time.Time `json:"createdAt"`
}

// MaxCleanupRetries caps retry attempts before a failed file is considered
// permanently lost and requires manual operator intervention.
const MaxCleanupRetries = 7
