CREATE TABLE recording_rule_executions (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  rule_id BIGINT NOT NULL,
  status VARCHAR(20) NOT NULL,
  error_message TEXT,
  duration_ms INT NOT NULL DEFAULT 0,
  executed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_rule_executed (rule_id, executed_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
