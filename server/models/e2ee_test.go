package models

import (
	"strings"
	"testing"
)

func TestCreateKeyBackupRequestValidateRejectsOversizedFields(t *testing.T) {
	valid := func() CreateKeyBackupRequest {
		return CreateKeyBackupRequest{EncryptedData: "ciphertext", Nonce: "nonce", Salt: "salt"}
	}
	tests := []struct {
		name string
		edit func(*CreateKeyBackupRequest)
	}{
		{"encrypted data", func(r *CreateKeyBackupRequest) { r.EncryptedData = strings.Repeat("x", MaxKeyBackupEncryptedDataBytes+1) }},
		{"nonce", func(r *CreateKeyBackupRequest) { r.Nonce = strings.Repeat("x", MaxKeyBackupFieldBytes+1) }},
		{"salt", func(r *CreateKeyBackupRequest) { r.Salt = strings.Repeat("x", MaxKeyBackupFieldBytes+1) }},
		{"algorithm", func(r *CreateKeyBackupRequest) { r.Algorithm = strings.Repeat("x", MaxKeyBackupFieldBytes+1) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := valid()
			tt.edit(&req)
			if err := req.Validate(); err == nil {
				t.Fatal("expected oversized field to be rejected")
			}
		})
	}
}

func TestCreateGroupSessionRequestValidateRejectsOversizedFields(t *testing.T) {
	tests := []struct {
		name string
		req  CreateGroupSessionRequest
	}{
		{"session ID", CreateGroupSessionRequest{SessionID: strings.Repeat("x", MaxGroupSessionIDBytes+1), SessionData: "data"}},
		{"session data", CreateGroupSessionRequest{SessionID: "id", SessionData: strings.Repeat("x", MaxGroupSessionDataBytes+1)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.req.Validate(); err == nil {
				t.Fatal("expected oversized field to be rejected")
			}
		})
	}
}
