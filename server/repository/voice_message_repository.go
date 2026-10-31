package repository

import (
	"context"

	"github.com/akinalp/mqvi/models"
)

// VoiceMessageRepository — ephemeral chat for active voice channel sessions.
// DeleteByChannel returns the IDs that were wiped so the caller can clean up files on disk.
type VoiceMessageRepository interface {
	Create(ctx context.Context, message *models.VoiceMessage) error
	GetByID(ctx context.Context, id string) (*models.VoiceMessage, error)
	GetByChannelID(ctx context.Context, channelID string, limit int) ([]models.VoiceMessage, error)
	UpdateContent(ctx context.Context, id, content string) error
	Delete(ctx context.Context, id string) error
	DeleteByChannel(ctx context.Context, channelID string) ([]string, error)

	CreateAttachment(ctx context.Context, attachment *models.VoiceMessageAttachment) error
	GetAttachmentsByMessageIDs(ctx context.Context, messageIDs []string) (map[string][]models.VoiceMessageAttachment, error)
}
