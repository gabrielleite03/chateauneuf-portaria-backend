package domain

import "time"

type ScheduledServiceStatus string

const (
	ScheduledServiceStatusScheduled ScheduledServiceStatus = "agendado"
	ScheduledServiceStatusDone      ScheduledServiceStatus = "realizado"
	ScheduledServiceStatusCanceled  ScheduledServiceStatus = "cancelado"
)

type ScheduledService struct {
	ID           string                 `json:"id"`
	Date         string                 `json:"date"`
	Name         string                 `json:"name"`
	Document     string                 `json:"document"`
	Company      string                 `json:"company"`
	Unit         string                 `json:"unit"`
	AuthorizedBy string                 `json:"authorizedBy"`
	ArrivalTime  string                 `json:"arrivalTime,omitempty"`
	Notes        string                 `json:"notes,omitempty"`
	Status       ScheduledServiceStatus `json:"status"`
	Photo        string                 `json:"photo,omitempty"`
	SyncStatus   SyncStatus             `json:"syncStatus"`
	CreatedAt    time.Time              `json:"-"`
	UpdatedAt    time.Time              `json:"-"`
}

func (s ScheduledServiceStatus) IsValid() bool {
	switch s {
	case ScheduledServiceStatusScheduled, ScheduledServiceStatusDone, ScheduledServiceStatusCanceled:
		return true
	default:
		return false
	}
}
