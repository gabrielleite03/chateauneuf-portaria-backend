CREATE TABLE IF NOT EXISTS scheduled_services (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	date TEXT NOT NULL,
	name TEXT NOT NULL,
	document TEXT NOT NULL,
	company TEXT NOT NULL,
	unit TEXT NOT NULL,
	authorized_by TEXT NOT NULL,
	arrival_time TEXT NOT NULL DEFAULT '',
	notes TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL CHECK (status IN ('agendado', 'realizado', 'cancelado')),
	photo TEXT NOT NULL DEFAULT '',
	sync_status TEXT NOT NULL CHECK (sync_status IN ('PENDENTE_SYNC', 'SINCRONIZADO', 'ERRO_SYNC')),
	sync_error TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	synced_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_scheduled_services_date ON scheduled_services(date);
CREATE INDEX IF NOT EXISTS idx_scheduled_services_document ON scheduled_services(document);
CREATE INDEX IF NOT EXISTS idx_scheduled_services_status ON scheduled_services(status);
