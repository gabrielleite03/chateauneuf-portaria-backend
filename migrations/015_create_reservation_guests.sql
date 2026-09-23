CREATE TABLE reservation_guests (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  reservation_id INTEGER NOT NULL REFERENCES common_area_reservations(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  document TEXT NOT NULL,
  confirmed BOOLEAN NOT NULL DEFAULT 0
);
CREATE INDEX idx_reservation_guests_reservation ON reservation_guests(reservation_id);
