CREATE TABLE IF NOT EXISTS access_logs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	external_id TEXT NOT NULL DEFAULT '',
	visitor_name TEXT NOT NULL,
	document TEXT NOT NULL,
	company TEXT NOT NULL DEFAULT '',
	phone TEXT NOT NULL DEFAULT '',
	unit TEXT NOT NULL,
	resident_name TEXT NOT NULL DEFAULT '',
	service_type TEXT NOT NULL DEFAULT '',
	vehicle_plate TEXT NOT NULL DEFAULT '',
	authorized_by TEXT NOT NULL DEFAULT '',
	doorman TEXT NOT NULL DEFAULT '',
	photo TEXT NOT NULL DEFAULT '',
	entry_at DATETIME NOT NULL,
	exit_at DATETIME,
	visit_status TEXT NOT NULL CHECK (visit_status IN ('EM_ANDAMENTO', 'FINALIZADO', 'CANCELADO', 'BLOQUEADO')),
	sync_status TEXT NOT NULL CHECK (sync_status IN ('PENDENTE_SYNC', 'SINCRONIZADO', 'ERRO_SYNC')),
	sync_error TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	synced_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_access_logs_document ON access_logs(document);
CREATE INDEX IF NOT EXISTS idx_access_logs_unit ON access_logs(unit);
CREATE INDEX IF NOT EXISTS idx_access_logs_entry_at ON access_logs(entry_at);
CREATE INDEX IF NOT EXISTS idx_access_logs_sync_status ON access_logs(sync_status);
CREATE INDEX IF NOT EXISTS idx_access_logs_visit_status ON access_logs(visit_status);
