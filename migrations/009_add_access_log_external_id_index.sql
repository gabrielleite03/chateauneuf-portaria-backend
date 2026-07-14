CREATE UNIQUE INDEX IF NOT EXISTS idx_access_logs_external_id
ON access_logs(external_id)
WHERE external_id != '';
