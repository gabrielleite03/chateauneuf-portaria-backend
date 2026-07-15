CREATE TABLE IF NOT EXISTS shopping_deliveries (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	unit TEXT NOT NULL,
	recipient TEXT NOT NULL DEFAULT '',
	courier_name TEXT NOT NULL,
	document TEXT NOT NULL,
	store TEXT NOT NULL,
	product TEXT NOT NULL,
	notes TEXT NOT NULL DEFAULT '',
	photo TEXT NOT NULL DEFAULT '',
	received_at DATETIME NOT NULL,
	withdrawn_at DATETIME,
	status TEXT NOT NULL CHECK (status IN ('aguardando_retirada', 'retirada')),
	sync_status TEXT NOT NULL DEFAULT 'SINCRONIZADO' CHECK (sync_status IN ('PENDENTE_SYNC', 'SINCRONIZADO', 'ERRO_SYNC')),
	sync_error TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	synced_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_shopping_deliveries_unit ON shopping_deliveries(unit);
CREATE INDEX IF NOT EXISTS idx_shopping_deliveries_status ON shopping_deliveries(status);
CREATE INDEX IF NOT EXISTS idx_shopping_deliveries_received_at ON shopping_deliveries(received_at);
CREATE INDEX IF NOT EXISTS idx_shopping_deliveries_sync_status ON shopping_deliveries(sync_status);
