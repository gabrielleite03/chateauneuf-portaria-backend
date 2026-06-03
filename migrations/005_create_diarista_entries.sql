CREATE TABLE IF NOT EXISTS diarista_entries (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	date TEXT NOT NULL,
	name TEXT NOT NULL,
	rg TEXT NOT NULL,
	unit TEXT NOT NULL,
	authorized_by TEXT NOT NULL,
	entry_time TEXT NOT NULL,
	exit_time TEXT NOT NULL DEFAULT '',
	gatekeeper TEXT NOT NULL,
	photo TEXT NOT NULL DEFAULT '',
	sync_status TEXT NOT NULL CHECK (sync_status IN ('PENDENTE_SYNC', 'SINCRONIZADO', 'ERRO_SYNC')),
	sync_error TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	synced_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_diarista_entries_date ON diarista_entries(date);
CREATE INDEX IF NOT EXISTS idx_diarista_entries_rg ON diarista_entries(rg);
CREATE INDEX IF NOT EXISTS idx_diarista_entries_unit ON diarista_entries(unit);
