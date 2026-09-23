package usecase

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"chateauneuf-portaria-backend/internal/domain"
)

var ErrVehiclePlateExists = errors.New("esta placa ja esta cadastrada neste apartamento")
var ErrVehicleUnitMissing = errors.New("salve o cadastro do apartamento antes de cadastrar veiculos")
var vehiclePlatePattern = regexp.MustCompile(`^[A-Z0-9]{1,20}$`)

type ResidentVehicle struct {
	ID    string `json:"id"`
	Unit  string `json:"unit"`
	Plate string `json:"plate"`
	Brand string `json:"brand"`
	Model string `json:"model"`
	Color string `json:"color"`
}

type ResidentVehicleInput struct {
	Plate string `json:"plate"`
	Brand string `json:"brand"`
	Model string `json:"model"`
	Color string `json:"color"`
}

type ResidentVehicleService struct{ db *sql.DB }

func NewResidentVehicleService(db *sql.DB) *ResidentVehicleService {
	return &ResidentVehicleService{db: db}
}

func (s *ResidentVehicleService) List(ctx context.Context, unit string) ([]ResidentVehicle, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, unit, plate, brand, model, color FROM resident_vehicles WHERE unit = ? ORDER BY plate, id`, unit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	vehicles := make([]ResidentVehicle, 0)
	for rows.Next() {
		var v ResidentVehicle
		if err := rows.Scan(&v.ID, &v.Unit, &v.Plate, &v.Brand, &v.Model, &v.Color); err != nil {
			return nil, err
		}
		vehicles = append(vehicles, v)
	}
	return vehicles, rows.Err()
}

func (s *ResidentVehicleService) Save(ctx context.Context, unit, id string, input ResidentVehicleInput) (*ResidentVehicle, error) {
	v := ResidentVehicle{ID: id, Unit: strings.TrimSpace(unit), Plate: strings.ToUpper(strings.TrimSpace(input.Plate)), Brand: strings.TrimSpace(input.Brand), Model: strings.TrimSpace(input.Model), Color: strings.TrimSpace(input.Color)}
	v.Plate = strings.NewReplacer("-", "", " ", "").Replace(v.Plate)
	if v.Unit == "" || !vehiclePlatePattern.MatchString(v.Plate) || v.Brand == "" || v.Model == "" || v.Color == "" || len([]rune(v.Brand)) > 100 || len([]rune(v.Model)) > 100 || len([]rune(v.Color)) > 60 {
		return nil, domain.ErrInvalidInput
	}
	var exists bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM residents WHERE unit = ?)`, v.Unit).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrVehicleUnitMissing
	}
	var result sql.Result
	var err error
	if id == "" {
		result, err = s.db.ExecContext(ctx, `INSERT INTO resident_vehicles (unit, plate, brand, model, color) VALUES (?, ?, ?, ?, ?)`, v.Unit, v.Plate, v.Brand, v.Model, v.Color)
	} else {
		result, err = s.db.ExecContext(ctx, `UPDATE resident_vehicles SET plate = ?, brand = ?, model = ?, color = ? WHERE unit = ? AND id = ?`, v.Plate, v.Brand, v.Model, v.Color, v.Unit, id)
	}
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: resident_vehicles.unit, resident_vehicles.plate") {
			return nil, ErrVehiclePlateExists
		}
		return nil, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, domain.ErrNotFound
	}
	if id == "" {
		numericID, err := result.LastInsertId()
		if err != nil {
			return nil, err
		}
		v.ID = strconv.FormatInt(numericID, 10)
	}
	return &v, nil
}

func (s *ResidentVehicleService) Delete(ctx context.Context, unit, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM resident_vehicles WHERE unit = ? AND id = ?`, unit, id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return domain.ErrNotFound
	}
	return nil
}
