-- Reserve usernames for 24 hours after emergency account deletion.
CREATE TABLE IF NOT EXISTS username_reservations (
    username TEXT PRIMARY KEY COLLATE NOCASE,
    reserved_until DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_username_reservations_expiry
    ON username_reservations(reserved_until);
