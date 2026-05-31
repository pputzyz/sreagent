DROP INDEX idx_users_oidc_subject ON users;

ALTER TABLE users DROP COLUMN oidc_subject;
