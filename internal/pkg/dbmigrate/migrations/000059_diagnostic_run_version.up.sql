ALTER TABLE diagnostic_runs ADD COLUMN version INT NOT NULL DEFAULT 1 AFTER result_summary;
