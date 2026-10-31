-- 106: encryption fields for ephemeral voice-channel chat.
-- Existing rows remain legacy plaintext; all new writes are rejected unless encrypted.
ALTER TABLE voice_messages ADD COLUMN encryption_version INTEGER NOT NULL DEFAULT 0;
ALTER TABLE voice_messages ADD COLUMN ciphertext TEXT;
ALTER TABLE voice_messages ADD COLUMN sender_device_id TEXT;
ALTER TABLE voice_messages ADD COLUMN e2ee_metadata TEXT;
