CREATE TABLE IF NOT EXISTS common_area_reservations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  area TEXT NOT NULL,
  resident_name TEXT NOT NULL,
  unit TEXT NOT NULL,
  reservation_date TEXT NOT NULL,
  start_time TEXT NOT NULL,
  end_time TEXT NOT NULL,
  guests TEXT NOT NULL DEFAULT '',
  notes TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'reservada',
  sync_status TEXT NOT NULL DEFAULT 'PENDENTE_SYNC',
  sync_error TEXT NOT NULL DEFAULT '',
  synced_at DATETIME,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_common_area_reservations_date ON common_area_reservations(reservation_date);
CREATE INDEX IF NOT EXISTS idx_common_area_reservations_area ON common_area_reservations(area);
CREATE INDEX IF NOT EXISTS idx_common_area_reservations_status ON common_area_reservations(status);
