package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"chateauneuf-portaria-backend/internal/usecase"
)

type ResidentVehicleHandler struct {
	service *usecase.ResidentVehicleService
}

func (h *ResidentVehicleHandler) List(w http.ResponseWriter, r *http.Request) {
	vehicles, err := h.service.List(r.Context(), r.PathValue("unit"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, vehicles)
}

func (h *ResidentVehicleHandler) Save(w http.ResponseWriter, r *http.Request) {
	var input usecase.ResidentVehicleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalido", "VALIDATION_ERROR")
		return
	}
	vehicle, err := h.service.Save(r.Context(), r.PathValue("unit"), r.PathValue("vehicleID"), input)
	if errors.Is(err, usecase.ErrVehiclePlateExists) {
		writeError(w, http.StatusConflict, err.Error(), "VEHICLE_PLATE_EXISTS")
		return
	}
	if errors.Is(err, usecase.ErrVehicleUnitMissing) {
		writeError(w, http.StatusBadRequest, err.Error(), "VEHICLE_UNIT_MISSING")
		return
	}
	if err != nil {
		writeDomainError(w, err)
		return
	}
	status := http.StatusOK
	if r.Method == http.MethodPost {
		status = http.StatusCreated
	}
	writeJSON(w, status, vehicle)
}

func (h *ResidentVehicleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), r.PathValue("unit"), r.PathValue("vehicleID")); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
