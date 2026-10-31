package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/akinalp/mqvi/database"
	"github.com/akinalp/mqvi/models"
	"github.com/akinalp/mqvi/pkg"
)

type sqliteVoiceMessageRepo struct {
	db database.TxQuerier
}

func NewSQLiteVoiceMessageRepo(db database.TxQuerier) VoiceMessageRepository {
	return &sqliteVoiceMessageRepo{db: db}
}

func (r *sqliteVoiceMessageRepo) Create(ctx context.Context, m *models.VoiceMessage) error {
	query := `
		INSERT INTO voice_messages (id, channel_id, user_id, content)
		VALUES (lower(hex(randomblob(8))), ?, ?, ?)
		RETURNING id, created_at`
	err := r.db.QueryRowContext(ctx, query, m.ChannelID, m.UserID, m.Content).
		Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return fmt.Errorf("create voice message: %w", err)
	}
	return nil
}

func (r *sqliteVoiceMessageRepo) GetByID(ctx context.Context, id string) (*models.VoiceMessage, error) {
	query := `
		SELECT vm.id, vm.channel_id, vm.user_id, vm.content, vm.edited_at, vm.created_at,
		       u.id, u.username, u.display_name, u.avatar_url, u.status, u.deleted_at, u.is_hard_deleted
		FROM voice_messages vm
		LEFT JOIN users u ON vm.user_id = u.id
		WHERE vm.id = ?`

	msg := &models.VoiceMessage{}
	var author models.User
	var authorID sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&msg.ID, &msg.ChannelID, &msg.UserID, &msg.Content, &msg.EditedAt, &msg.CreatedAt,
		&authorID, &author.Username, &author.DisplayName, &author.AvatarURL, &author.Status, &author.DeletedAt, &author.IsHardDeleted,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, pkg.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get voice message by id: %w", err)
	}
	if authorID.Valid {
		author.ID = authorID.String
		author.PasswordHash = ""
		msg.Author = &author
	}
	return msg, nil
}

func (r *sqliteVoiceMessageRepo) GetByChannelID(ctx context.Context, channelID string, limit int) ([]models.VoiceMessage, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := `
		SELECT vm.id, vm.channel_id, vm.user_id, vm.content, vm.edited_at, vm.created_at,
		       u.id, u.username, u.display_name, u.avatar_url, u.status, u.deleted_at, u.is_hard_deleted
		FROM voice_messages vm
		LEFT JOIN users u ON vm.user_id = u.id
		WHERE vm.channel_id = ?
		ORDER BY vm.created_at ASC
		LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, channelID, limit)
	if err != nil {
		return nil, fmt.Errorf("list voice messages: %w", err)
	}
	defer rows.Close()

	var out []models.VoiceMessage
	for rows.Next() {
		var msg models.VoiceMessage
		var author models.User
		var authorID sql.NullString
		if err := rows.Scan(
			&msg.ID, &msg.ChannelID, &msg.UserID, &msg.Content, &msg.EditedAt, &msg.CreatedAt,
			&authorID, &author.Username, &author.DisplayName, &author.AvatarURL, &author.Status, &author.DeletedAt, &author.IsHardDeleted,
		); err != nil {
			return nil, fmt.Errorf("scan voice message row: %w", err)
		}
		if authorID.Valid {
			author.ID = authorID.String
			author.PasswordHash = ""
			msg.Author = &author
		}
		out = append(out, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate voice messages: %w", err)
	}
	return out, nil
}

func (r *sqliteVoiceMessageRepo) UpdateContent(ctx context.Context, id, content string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE voice_messages SET content = ?, edited_at = CURRENT_TIMESTAMP WHERE id = ?`,
		content, id,
	)
	if err != nil {
		return fmt.Errorf("update voice message: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update voice message rows: %w", err)
	}
	if n == 0 {
		return pkg.ErrNotFound
	}
	return nil
}

func (r *sqliteVoiceMessageRepo) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM voice_messages WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete voice message: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete voice message rows: %w", err)
	}
	if n == 0 {
		return pkg.ErrNotFound
	}
	return nil
}

// DeleteByChannel wipes every message in a channel and returns the deleted IDs
// so the caller can purge their on-disk attachment files (FK CASCADE handles the row side).
func (r *sqliteVoiceMessageRepo) DeleteByChannel(ctx context.Context, channelID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id FROM voice_messages WHERE channel_id = ?`, channelID)
	if err != nil {
		return nil, fmt.Errorf("list voice messages for delete: %w", err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan id for delete: %w", err)
		}
		ids = append(ids, id)
	}
	rows.Close()

	if _, err := r.db.ExecContext(ctx, `DELETE FROM voice_messages WHERE channel_id = ?`, channelID); err != nil {
		return nil, fmt.Errorf("delete voice messages by channel: %w", err)
	}
	return ids, nil
}

func (r *sqliteVoiceMessageRepo) CreateAttachment(ctx context.Context, a *models.VoiceMessageAttachment) error {
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO voice_message_attachments (id, voice_message_id, file_name, file_url, file_size, mime_type)
		 VALUES (lower(hex(randomblob(8))), ?, ?, ?, ?, ?)
		 RETURNING id`,
		a.VoiceMessageID, a.Filename, a.FileURL, a.FileSize, a.MimeType,
	).Scan(&a.ID)
	if err != nil {
		return fmt.Errorf("create voice message attachment: %w", err)
	}
	return nil
}

func (r *sqliteVoiceMessageRepo) GetAttachmentsByMessageIDs(ctx context.Context, messageIDs []string) (map[string][]models.VoiceMessageAttachment, error) {
	if len(messageIDs) == 0 {
		return make(map[string][]models.VoiceMessageAttachment), nil
	}
	placeholders := strings.Repeat("?,", len(messageIDs))
	placeholders = placeholders[:len(placeholders)-1]

	query := fmt.Sprintf(`
		SELECT id, voice_message_id, file_name, file_url, file_size, mime_type
		FROM voice_message_attachments
		WHERE voice_message_id IN (%s)`, placeholders)

	args := make([]any, len(messageIDs))
	for i, id := range messageIDs {
		args[i] = id
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list voice message attachments: %w", err)
	}
	defer rows.Close()

	out := make(map[string][]models.VoiceMessageAttachment)
	for rows.Next() {
		var a models.VoiceMessageAttachment
		if err := rows.Scan(&a.ID, &a.VoiceMessageID, &a.Filename, &a.FileURL, &a.FileSize, &a.MimeType); err != nil {
			return nil, fmt.Errorf("scan voice message attachment: %w", err)
		}
		out[a.VoiceMessageID] = append(out[a.VoiceMessageID], a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate voice message attachments: %w", err)
	}
	return out, nil
}
