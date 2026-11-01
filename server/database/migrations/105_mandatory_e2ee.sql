-- 105: E2EE is mandatory for all user-authored messages.
-- Legacy plaintext history remains readable as legacy data; it is not falsely relabeled as encrypted.
UPDATE servers SET e2ee_enabled = 1;
UPDATE dm_channels SET e2ee_enabled = 1;

CREATE TRIGGER IF NOT EXISTS servers_e2ee_force_insert
AFTER INSERT ON servers
WHEN NEW.e2ee_enabled <> 1
BEGIN
  UPDATE servers SET e2ee_enabled = 1 WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS servers_e2ee_prevent_disable
BEFORE UPDATE OF e2ee_enabled ON servers
WHEN NEW.e2ee_enabled <> 1
BEGIN
  SELECT RAISE(ABORT, 'end-to-end encryption is mandatory');
END;

CREATE TRIGGER IF NOT EXISTS dm_channels_e2ee_force_insert
AFTER INSERT ON dm_channels
WHEN NEW.e2ee_enabled <> 1
BEGIN
  UPDATE dm_channels SET e2ee_enabled = 1 WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS dm_channels_e2ee_prevent_disable
BEFORE UPDATE OF e2ee_enabled ON dm_channels
WHEN NEW.e2ee_enabled <> 1
BEGIN
  SELECT RAISE(ABORT, 'end-to-end encryption is mandatory');
END;
