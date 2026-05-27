-- If any rules have NULL data_source_id, set them to 0 before making column NOT NULL
UPDATE alert_rules SET data_source_id = 0 WHERE data_source_id IS NULL;
ALTER TABLE alert_rules MODIFY COLUMN data_source_id BIGINT NOT NULL;
