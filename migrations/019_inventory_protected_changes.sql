CREATE TABLE IF NOT EXISTS inventory_product_changes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  product_id TEXT NOT NULL REFERENCES inventory_products(id),
  action TEXT NOT NULL CHECK(action IN ('edit', 'delete')),
  before_json TEXT NOT NULL,
  after_json TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
ALTER TABLE inventory_products ADD COLUMN deleted_at TEXT NOT NULL DEFAULT '';
