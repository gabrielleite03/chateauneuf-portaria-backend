CREATE TABLE IF NOT EXISTS reservation_signatures (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 reservation_id INTEGER NOT NULL,
 code_hash TEXT NOT NULL UNIQUE,
 expires_at DATETIME NOT NULL,
 consumed_at DATETIME,
 signature_png BLOB,
 pdf_data BLOB,
 recipient_email TEXT NOT NULL DEFAULT '',
 email_status TEXT NOT NULL DEFAULT 'pending',
 email_error TEXT NOT NULL DEFAULT '',
 created_at DATETIME NOT NULL,
 FOREIGN KEY(reservation_id) REFERENCES common_area_reservations(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_reservation_signatures_reservation ON reservation_signatures(reservation_id);
