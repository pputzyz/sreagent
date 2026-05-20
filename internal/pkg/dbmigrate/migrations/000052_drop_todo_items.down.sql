CREATE TABLE IF NOT EXISTS todo_items (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT UNSIGNED NOT NULL,
  title VARCHAR(255) NOT NULL,
  description TEXT,
  priority VARCHAR(20) DEFAULT 'medium',
  due_at DATETIME,
  status VARCHAR(20) NOT NULL DEFAULT 'pending',
  completed_at DATETIME,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  KEY idx_todo_user_status (user_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
