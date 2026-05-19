-- Rollback: Revert DataSource AuthConfig encryption
-- Note: encrypted values in the DB will remain encrypted.
-- To fully rollback, re-save each datasource through the API with the encryption key removed.
SELECT 1;
