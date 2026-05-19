DROP INDEX idx_incidents_fingerprint ON incidents;
ALTER TABLE incidents DROP COLUMN fingerprint;
