ALTER TABLE incidents ADD COLUMN fingerprint VARCHAR(64) DEFAULT '' AFTER channel_id, ADD INDEX idx_incidents_fingerprint (fingerprint);
