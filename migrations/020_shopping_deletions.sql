CREATE TABLE IF NOT EXISTS shopping_delivery_deletions (
    delivery_id INTEGER PRIMARY KEY REFERENCES shopping_deliveries(id),
    deleted_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
