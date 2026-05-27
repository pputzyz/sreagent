ALTER TABLE notify_medias ADD COLUMN team_id BIGINT UNSIGNED NULL;
ALTER TABLE notify_medias ADD INDEX idx_team_id (team_id);
