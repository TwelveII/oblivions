package repository

import (
	"context"

	"github.com/akinalp/mqvi/models"
)

type PasswordResetRepository interface {
	Create(ctx context.Context, token *models.PasswordResetToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*models.PasswordResetToken, error)
	DeleteByID(ctx context.Context, id string) error
	DeleteByUserID(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context) error
	GetLatestByUserID(ctx context.Context, userID string) (*models.PasswordResetToken, error)
}
