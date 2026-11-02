-- 071_push_token_type.sql
-- Distinguishes FCM registration tokens from iOS PushKit VoIP tokens. VoIP tokens
-- are delivered via direct APNs (CallKit), not FCM, so call dispatch must tell them
-- apart. Existing rows default to 'fcm'. Values: 'fcm' | 'apns_voip'.

ALTER TABLE push_tokens ADD COLUMN token_type TEXT NOT NULL DEFAULT 'fcm';
