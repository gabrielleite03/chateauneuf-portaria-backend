CREATE TRIGGER IF NOT EXISTS trg_access_logs_prevent_duplicate_active_visit_insert
BEFORE INSERT ON access_logs
WHEN NEW.exit_at IS NULL
  AND NEW.visit_status = 'EM_ANDAMENTO'
  AND EXISTS (
    SELECT 1
    FROM access_logs
    WHERE exit_at IS NULL
      AND visit_status = 'EM_ANDAMENTO'
      AND lower(replace(replace(replace(replace(replace(trim(document), '.', ''), '-', ''), '/', ''), ' ', ''), char(9), '')) =
          lower(replace(replace(replace(replace(replace(trim(NEW.document), '.', ''), '-', ''), '/', ''), ' ', ''), char(9), ''))
  )
BEGIN
  SELECT RAISE(ABORT, 'active visit already exists');
END;

CREATE TRIGGER IF NOT EXISTS trg_access_logs_prevent_duplicate_active_visit_update
BEFORE UPDATE OF document, exit_at, visit_status ON access_logs
WHEN NEW.exit_at IS NULL
  AND NEW.visit_status = 'EM_ANDAMENTO'
  AND EXISTS (
    SELECT 1
    FROM access_logs
    WHERE id <> NEW.id
      AND exit_at IS NULL
      AND visit_status = 'EM_ANDAMENTO'
      AND lower(replace(replace(replace(replace(replace(trim(document), '.', ''), '-', ''), '/', ''), ' ', ''), char(9), '')) =
          lower(replace(replace(replace(replace(replace(trim(NEW.document), '.', ''), '-', ''), '/', ''), ' ', ''), char(9), ''))
  )
BEGIN
  SELECT RAISE(ABORT, 'active visit already exists');
END;
