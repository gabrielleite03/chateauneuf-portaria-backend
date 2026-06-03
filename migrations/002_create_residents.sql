CREATE TABLE IF NOT EXISTS residents (
	unit TEXT PRIMARY KEY,
	owner TEXT NOT NULL DEFAULT '',
	phones TEXT NOT NULL DEFAULT '',
	tenant TEXT NOT NULL DEFAULT '',
	family_members TEXT NOT NULL DEFAULT '',
	photo TEXT NOT NULL DEFAULT '',
	sync_status TEXT NOT NULL DEFAULT 'SINCRONIZADO' CHECK (sync_status IN ('PENDENTE_SYNC', 'SINCRONIZADO', 'ERRO_SYNC')),
	sync_error TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	synced_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_residents_sync_status ON residents(sync_status);
CREATE INDEX IF NOT EXISTS idx_residents_updated_at ON residents(updated_at);
