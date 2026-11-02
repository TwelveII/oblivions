-- Ephemeral chat tied to active voice channel sessions.
-- Wiped (DELETE WHERE channel_id = ?) when the last participant leaves.

CREATE TABLE IF NOT EXISTS voice_messages (
    id TEXT PRIMARY KEY,
    channel_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    content TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    edited_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_voice_messages_channel_id ON voice_messages(channel_id);
CREATE INDEX IF NOT EXISTS idx_voice_messages_created_at ON voice_messages(created_at);

CREATE TABLE IF NOT EXISTS voice_message_attachments (
    id TEXT PRIMARY KEY,
    voice_message_id TEXT NOT NULL REFERENCES voice_messages(id) ON DELETE CASCADE,
    file_url TEXT NOT NULL,
    file_name TEXT NOT NULL,
    file_size INTEGER NOT NULL,
    mime_type TEXT
);

CREATE INDEX IF NOT EXISTS idx_voice_message_attachments_voice_message_id ON voice_message_attachments(voice_message_id);
