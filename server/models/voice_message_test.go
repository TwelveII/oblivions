package models

import "testing"

func TestCreateVoiceMessageRequestRequiresE2EE(t *testing.T) {
	plain := &CreateVoiceMessageRequest{Content: "hello"}
	if err := plain.Validate(); err == nil {
		t.Fatal("expected plaintext voice chat message to be rejected")
	}

	ciphertext, device := "encrypted-payload", "device-1"
	encrypted := &CreateVoiceMessageRequest{
		EncryptionVersion: 1,
		Ciphertext:        &ciphertext,
		SenderDeviceID:    &device,
	}
	if err := encrypted.Validate(); err != nil {
		t.Fatalf("encrypted voice chat message should validate: %v", err)
	}
}

func TestUpdateVoiceMessageRequestRequiresE2EE(t *testing.T) {
	plain := &UpdateVoiceMessageRequest{Content: "edited"}
	if err := plain.Validate(); err == nil {
		t.Fatal("expected plaintext voice chat edit to be rejected")
	}

	ciphertext, device := "encrypted-edit", "device-1"
	encrypted := &UpdateVoiceMessageRequest{
		EncryptionVersion: 1,
		Ciphertext:        &ciphertext,
		SenderDeviceID:    &device,
	}
	if err := encrypted.Validate(); err != nil {
		t.Fatalf("encrypted voice chat edit should validate: %v", err)
	}
}
