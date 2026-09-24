CREATE TABLE IF NOT EXISTS inventory_requests (
  id TEXT PRIMARY KEY,
  fingerprint TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS inventory_products (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  name_key TEXT NOT NULL UNIQUE,
  unit TEXT NOT NULL,
  minimum_milli INTEGER NOT NULL DEFAULT 0 CHECK(minimum_milli >= 0),
  stock_milli INTEGER NOT NULL DEFAULT 0 CHECK(stock_milli >= 0)
);
CREATE TABLE IF NOT EXISTS inventory_purchases (
  id TEXT PRIMARY KEY,
  date TEXT NOT NULL,
  supplier TEXT NOT NULL,
  document TEXT NOT NULL DEFAULT '',
  notes TEXT NOT NULL DEFAULT '',
  total_cents INTEGER NOT NULL CHECK(total_cents >= 0),
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS inventory_movements (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  product_id TEXT NOT NULL REFERENCES inventory_products(id),
  purchase_id TEXT REFERENCES inventory_purchases(id),
  kind TEXT NOT NULL CHECK(kind IN ('initial', 'purchase', 'withdrawal')),
  date TEXT NOT NULL,
  quantity_milli INTEGER NOT NULL CHECK(quantity_milli != 0),
  unit_cost_cents INTEGER NOT NULL DEFAULT 0 CHECK(unit_cost_cents >= 0),
  total_cents INTEGER NOT NULL DEFAULT 0 CHECK(total_cents >= 0),
  responsible TEXT NOT NULL DEFAULT '',
  notes TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_inventory_movement_product ON inventory_movements(product_id, id);
CREATE INDEX IF NOT EXISTS idx_inventory_movement_purchase ON inventory_movements(purchase_id);
