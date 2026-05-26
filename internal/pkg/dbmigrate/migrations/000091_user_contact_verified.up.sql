ALTER TABLE user_contacts ADD COLUMN verified TINYINT(1) NOT NULL DEFAULT 0 AFTER is_default;
