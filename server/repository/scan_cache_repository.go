package repository

import "context"

type ScanHashCacheEntry struct {
	SHA256    string
	Status    string
	Signature *string
	ScannedAt string
}

type ScanHashCacheRepository interface {
	Get(ctx context.Context, sha256 string) (*ScanHashCacheEntry, error)
	Upsert(ctx context.Context, sha256, status string, signature *string) error
	DeleteBefore(ctx context.Context, status, cutoff string) (int, error)
}
