ALTER TABLE incidents ADD COLUMN fingerprint VARCHAR(64) DEFAULT '' AFTER channel_id;
CREATE INDEX idx_incidents_fingerprint ON incidents (fingerprint);
