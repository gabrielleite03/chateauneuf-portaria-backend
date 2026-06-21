package domain

import "time"

type ReservationStatus string

const (
	ReservationStatusBooked    ReservationStatus = "reservada"
	ReservationStatusCompleted ReservationStatus = "concluida"
	ReservationStatusCanceled  ReservationStatus = "cancelada"
)

type CommonAreaReservation struct {
	ID              string            `json:"id"`
	Area            string            `json:"area"`
	ResidentName    string            `json:"residentName"`
	Unit            string            `json:"unit"`
	ReservationDate string            `json:"reservationDate"`
	StartTime       string            `json:"startTime"`
	EndTime         string            `json:"endTime"`
	Guests          string            `json:"guests,omitempty"`
	Notes           string            `json:"notes,omitempty"`
	Status          ReservationStatus `json:"status"`
	SyncStatus      SyncStatus        `json:"syncStatus"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
}

func (s ReservationStatus) IsValid() bool {
	switch s {
	case ReservationStatusBooked, ReservationStatusCompleted, ReservationStatusCanceled:
		return true
	default:
		return false
	}
}
