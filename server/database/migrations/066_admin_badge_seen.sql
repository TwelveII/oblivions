-- Platform admin badge "last seen" timestamps. Drive the dot indicator
-- on Feedback and Reports menu items in settings.
ALTER TABLE users ADD COLUMN feedback_last_seen_at TIMESTAMP;
ALTER TABLE users ADD COLUMN reports_last_seen_at TIMESTAMP;
