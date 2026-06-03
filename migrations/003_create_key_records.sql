CREATE TABLE IF NOT EXISTS key_records (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	date TEXT NOT NULL,
	local TEXT NOT NULL,
	resident_name TEXT NOT NULL,
	unit TEXT NOT NULL,
	pickup_time TEXT NOT NULL,
	return_time TEXT NOT NULL DEFAULT '',
	gatekeeper TEXT NOT NULL,
	status TEXT NOT NULL CHECK (status IN ('retirada', 'devolvida')),
	sync_status TEXT NOT NULL DEFAULT 'SINCRONIZADO' CHECK (sync_status IN ('PENDENTE_SYNC', 'SINCRONIZADO', 'ERRO_SYNC')),
	sync_error TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	synced_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_key_records_date ON key_records(date);
CREATE INDEX IF NOT EXISTS idx_key_records_unit ON key_records(unit);
CREATE INDEX IF NOT EXISTS idx_key_records_status ON key_records(status);
CREATE INDEX IF NOT EXISTS idx_key_records_sync_status ON key_records(sync_status);
