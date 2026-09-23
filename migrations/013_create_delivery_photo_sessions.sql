CREATE TABLE IF NOT EXISTS delivery_photo_sessions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code_hash TEXT NOT NULL UNIQUE,
  photo_data TEXT NOT NULL DEFAULT '',
  expires_at DATETIME NOT NULL,
  uploaded_at DATETIME,
  consumed_at DATETIME,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_delivery_photo_sessions_expires
ON delivery_photo_sessions(expires_at);
