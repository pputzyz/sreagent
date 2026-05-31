ALTER TABLE users ADD COLUMN oidc_subject VARCHAR(256) NULL;

CREATE INDEX idx_users_oidc_subject ON users (oidc_subject);
