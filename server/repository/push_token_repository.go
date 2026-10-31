package repository

import (
	"context"

	"github.com/akinalp/mqvi/models"
)

// PushTokenRepository defines data access for push notification tokens.
type PushTokenRepository interface {
	// Upsert inserts or updates a token. ON CONFLICT(token) reassigns it to the current user.
	Upsert(ctx context.Context, token *models.PushToken) error
	ListByUser(ctx context.Context, userID string) ([]models.PushToken, error)
	// Delete removes one token owned by the user (logout / permission revoke).
	Delete(ctx context.Context, userID, token string) error
	// DeleteTokens removes a batch of tokens regardless of owner — used to prune
	// tokens FCM reports as unregistered/invalid after a failed send.
	DeleteTokens(ctx context.Context, tokens []string) error
}
