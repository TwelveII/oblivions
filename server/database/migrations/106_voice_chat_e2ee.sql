-- 106: encryption fields for ephemeral voice-channel chat.
-- Existing rows remain legacy plaintext; all new writes are rejected unless encrypted.
ALTER TABLE voice_messages ADD COLUMN encryption_version INTEGER NOT NULL DEFAULT 0;
ALTER TABLE voice_messages ADD COLUMN ciphertext TEXT;
ALTER TABLE voice_messages ADD COLUMN sender_device_id TEXT;
ALTER TABLE voice_messages ADD COLUMN e2ee_metadata TEXT;

CREATE TRIGGER IF NOT EXISTS voice_messages_require_e2ee_insert
BEFORE INSERT ON voice_messages
WHEN NEW.encryption_version <> 1 OR NEW.ciphertext IS NULL OR NEW.sender_device_id IS NULL
BEGIN
  SELECT RAISE(ABORT, 'voice chat requires end-to-end encryption');
END;

CREATE TRIGGER IF NOT EXISTS voice_messages_require_e2ee_update
BEFORE UPDATE OF content, encryption_version, ciphertext, sender_device_id ON voice_messages
WHEN NEW.encryption_version <> 1 OR NEW.ciphertext IS NULL OR NEW.sender_device_id IS NULL
BEGIN
  SELECT RAISE(ABORT, 'voice chat requires end-to-end encryption');
END;
