package models

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// VoiceMessage — ephemeral chat tied to an active voice channel session.
// All rows for a channel are deleted when the last participant leaves.
type VoiceMessage struct {
	ID          string                   `json:"id"`
	ChannelID   string                   `json:"channel_id"`
	UserID      string                   `json:"user_id"`
	Content           *string                  `json:"content"`
	EncryptionVersion int                      `json:"encryption_version"`
	Ciphertext        *string                  `json:"ciphertext,omitempty"`
	SenderDeviceID    *string                  `json:"sender_device_id,omitempty"`
	E2EEMetadata      *string                  `json:"e2ee_metadata,omitempty"`
	EditedAt          *time.Time               `json:"edited_at"`
	E2EEFileKeys      []EncryptedFileMeta      `json:"e2ee_file_keys,omitempty"
	CreatedAt   time.Time                `json:"created_at"`
	Author      *User                    `json:"author,omitempty"`
	Attachments []VoiceMessageAttachment `json:"attachments,omitempty"`
}

type VoiceMessageAttachment struct {
	ID             string  `json:"id"`
	VoiceMessageID string  `json:"voice_message_id"`
	Filename       string  `json:"filename"`
	FileURL        string  `json:"file_url"`
	FileSize       int64   `json:"file_size"`
	MimeType       *string `json:"mime_type,omitempty"`
}

type CreateVoiceMessageRequest struct {
	Content  string `json:"content"`
	HasFiles bool   `json:"-"`
}

func (r *CreateVoiceMessageRequest) Validate() error {
	r.Content = strings.TrimSpace(r.Content)
	contentLen := utf8.RuneCountInString(r.Content)
	if r.EncryptionVersion != 1 || r.Ciphertext == nil || *r.Ciphertext == "" || r.SenderDeviceID == nil || *r.SenderDeviceID == "" {
		return fmt.Errorf("voice chat messages require end-to-end encryption")
	}
	if contentLen == 0 && !r.HasFiles {
		return fmt.Errorf("content or attachment required")
	}
	if contentLen > MaxMessageLength {
		return fmt.Errorf("content exceeds %d characters", MaxMessageLength)
	}
	return nil
}

type UpdateVoiceMessageRequest struct {
	Content           string  `json:"content"`
	EncryptionVersion int     `json:"encryption_version"`
	Ciphertext        *string `json:"ciphertext,omitempty"`
	SenderDeviceID    *string `json:"sender_device_id,omitempty"`
	E2EEMetadata      *string `json:"e2ee_metadata,omitempty"
}

func (r *UpdateVoiceMessageRequest) Validate() error {
	r.Content = strings.TrimSpace(r.Content)
	contentLen := utf8.RuneCountInString(r.Content)
	if r.EncryptionVersion != 1 || r.Ciphertext == nil || *r.Ciphertext == "" || r.SenderDeviceID == nil || *r.SenderDeviceID == "" {
		return fmt.Errorf("voice chat edits require end-to-end encryption")
	}
	if contentLen == 0 {
		return fmt.Errorf("content required")
	}
	if contentLen > MaxMessageLength {
		return fmt.Errorf("content exceeds %d characters", MaxMessageLength)
	}
	return nil
}
