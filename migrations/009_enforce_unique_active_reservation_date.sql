DROP INDEX IF EXISTS idx_common_area_reservations_unique_active_date;

CREATE UNIQUE INDEX IF NOT EXISTS idx_common_area_reservations_unique_active_area_date
ON common_area_reservations(reservation_date, area)
WHERE status = 'reservada';

CREATE UNIQUE INDEX IF NOT EXISTS idx_common_area_reservations_unique_active_unit_date
ON common_area_reservations(reservation_date, unit)
WHERE status = 'reservada';
