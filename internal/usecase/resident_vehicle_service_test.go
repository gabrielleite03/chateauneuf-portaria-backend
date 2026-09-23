package usecase

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"chateauneuf-portaria-backend/internal/database"
	"chateauneuf-portaria-backend/internal/domain"
)

func TestResidentVehicleCRUD(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "vehicles.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(db, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	for _, unit := range []string{"11", "12"} {
		if _, err := db.Exec(`INSERT INTO residents (unit, created_at, updated_at) VALUES (?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, unit); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	service := NewResidentVehicleService(db)
	input := ResidentVehicleInput{Plate: " abc-1d23 ", Brand: " Fiat ", Model: " Argo ", Color: " Prata "}
	if _, err := service.Save(ctx, "99", "", input); !errors.Is(err, ErrVehicleUnitMissing) {
		t.Fatalf("missing unit: %v", err)
	}
	invalid := []ResidentVehicleInput{
		{Brand: "Fiat", Model: "Argo", Color: "Prata"},
		{Plate: "ABC1234", Model: "Argo", Color: "Prata"},
		{Plate: "ABC1234", Brand: "Fiat", Color: "Prata"},
		{Plate: "ABC1234", Brand: "Fiat", Model: "Argo"},
		{Plate: "!!!", Brand: "Fiat", Model: "Argo", Color: "Prata"},
	}
	for _, value := range invalid {
		if _, err := service.Save(ctx, "11", "", value); !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("invalid input accepted: %+v %v", value, err)
		}
	}
	first, err := service.Save(ctx, "11", "", input)
	if err != nil {
		t.Fatal(err)
	}
	if first.Plate != "ABC1D23" || first.Brand != "Fiat" || first.Model != "Argo" || first.Color != "Prata" {
		t.Fatalf("normalization: %+v", first)
	}
	input.Plate = "ABC1D23"
	if _, err := service.Save(ctx, "11", "", input); !errors.Is(err, ErrVehiclePlateExists) {
		t.Fatalf("duplicate plate: %v", err)
	}
	if _, err := service.Save(ctx, "12", "", input); err != nil {
		t.Fatalf("independent unit: %v", err)
	}
	if _, err := service.Save(ctx, "12", first.ID, input); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-unit update: %v", err)
	}
	if err := service.Delete(ctx, "12", first.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-unit delete: %v", err)
	}
	input.Plate = "XYZ9876"
	second, err := service.Save(ctx, "11", "", input)
	if err != nil {
		t.Fatal(err)
	}
	input.Plate = first.Plate
	if _, err := service.Save(ctx, "11", second.ID, input); !errors.Is(err, ErrVehiclePlateExists) {
		t.Fatalf("duplicate on edit: %v", err)
	}
	input = ResidentVehicleInput{Plate: "DEF-5678", Brand: "Honda", Model: "Civic", Color: "Preto"}
	if _, err := service.Save(ctx, "11", first.ID, input); err != nil {
		t.Fatal(err)
	}
	rows, err := NewResidentVehicleService(db).List(ctx, "11")
	if err != nil || len(rows) != 2 || rows[0].Plate != "DEF5678" || rows[0].Brand != "Honda" || rows[0].Model != "Civic" || rows[0].Color != "Preto" {
		t.Fatalf("saved edit: %+v %v", rows, err)
	}
	if err := service.Delete(ctx, "11", first.ID); err != nil {
		t.Fatal(err)
	}
	rows, err = service.List(ctx, "11")
	if err != nil || len(rows) != 1 || rows[0].ID != second.ID {
		t.Fatalf("delete: %+v %v", rows, err)
	}
	rows, err = service.List(ctx, "12")
	if err != nil || len(rows) != 1 || rows[0].Plate != "ABC1D23" {
		t.Fatalf("other unit changed: %+v %v", rows, err)
	}
	if err := service.Delete(ctx, "11", second.ID); err != nil {
		t.Fatal(err)
	}
	rows, err = service.List(ctx, "11")
	if err != nil || rows == nil || len(rows) != 0 {
		t.Fatalf("empty list: %+v %v", rows, err)
	}
}
