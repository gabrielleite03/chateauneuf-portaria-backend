CREATE TABLE IF NOT EXISTS delivery_withdrawal_signatures (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    delivery_id INTEGER NOT NULL,
    code_hash TEXT NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    consumed_at DATETIME,
    signature_png BLOB,
    recipient_email TEXT NOT NULL DEFAULT '',
    email_status TEXT NOT NULL DEFAULT 'pending',
    email_error TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL,
    FOREIGN KEY (delivery_id) REFERENCES shopping_deliveries(id)
);

CREATE INDEX IF NOT EXISTS idx_delivery_withdrawal_signatures_delivery
ON delivery_withdrawal_signatures(delivery_id, expires_at);
