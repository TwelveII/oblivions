package models

import (
	"fmt"
	"time"
)

// VoiceMessage is ephemeral chat tied to an active voice channel session.
type VoiceMessage struct {
	ID                string                   `json:"id"`
	ChannelID         string                   `json:"channel_id"`
	UserID            string                   `json:"user_id"`
	Content           *string                  `json:"content"`
	EncryptionVersion int                      `json:"encryption_version"`
	Ciphertext        *string                  `json:"ciphertext,omitempty"`
	SenderDeviceID    *string                  `json:"sender_device_id,omitempty"`
	E2EEMetadata      *string                  `json:"e2ee_metadata,omitempty"`
	EditedAt          *time.Time               `json:"edited_at"`
	CreatedAt         time.Time                `json:"created_at"`
	Author            *User                    `json:"author,omitempty"`
	Attachments       []VoiceMessageAttachment `json:"attachments,omitempty"`
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
	Content           string  `json:"content"`
	EncryptionVersion int     `json:"encryption_version"`
	Ciphertext        *string `json:"ciphertext,omitempty"`
	SenderDeviceID    *string `json:"sender_device_id,omitempty"`
	E2EEMetadata      *string `json:"e2ee_metadata,omitempty"`
	HasFiles          bool    `json:"-"`
}

func (r *CreateVoiceMessageRequest) Validate() error {
	if r.EncryptionVersion != 1 || r.Ciphertext == nil || *r.Ciphertext == "" || r.SenderDeviceID == nil || *r.SenderDeviceID == "" {
		return fmt.Errorf("voice chat messages require end-to-end encryption")
	}
	return nil
}

type UpdateVoiceMessageRequest struct {
	Content           string  `json:"content"`
	EncryptionVersion int     `json:"encryption_version"`
	Ciphertext        *string `json:"ciphertext,omitempty"`
	SenderDeviceID    *string `json:"sender_device_id,omitempty"`
	E2EEMetadata      *string `json:"e2ee_metadata,omitempty"`
}

func (r *UpdateVoiceMessageRequest) Validate() error {
	if r.EncryptionVersion != 1 || r.Ciphertext == nil || *r.Ciphertext == "" || r.SenderDeviceID == nil || *r.SenderDeviceID == "" {
		return fmt.Errorf("voice chat edits require end-to-end encryption")
	}
	return nil
}
