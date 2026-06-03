package domain

import "time"

type VisitStatus string

const (
	VisitStatusInProgress VisitStatus = "EM_ANDAMENTO"
	VisitStatusFinished   VisitStatus = "FINALIZADO"
	VisitStatusCanceled   VisitStatus = "CANCELADO"
	VisitStatusBlocked    VisitStatus = "BLOQUEADO"
)

type SyncStatus string

const (
	SyncStatusPending SyncStatus = "PENDENTE_SYNC"
	SyncStatusSynced  SyncStatus = "SINCRONIZADO"
	SyncStatusError   SyncStatus = "ERRO_SYNC"
)

type AccessLog struct {
	ID           int64       `json:"id"`
	ExternalID   string      `json:"external_id"`
	VisitorName  string      `json:"visitor_name"`
	Document     string      `json:"document"`
	Company      string      `json:"company"`
	Phone        string      `json:"phone"`
	Unit         string      `json:"unit"`
	ResidentName string      `json:"resident_name"`
	ServiceType  string      `json:"service_type"`
	VehiclePlate string      `json:"vehicle_plate"`
	AuthorizedBy string      `json:"authorized_by"`
	Doorman      string      `json:"doorman"`
	Photo        string      `json:"photo"`
	EntryAt      time.Time   `json:"entry_at"`
	ExitAt       *time.Time  `json:"exit_at"`
	VisitStatus  VisitStatus `json:"visit_status"`
	SyncStatus   SyncStatus  `json:"sync_status"`
	SyncError    string      `json:"sync_error"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
	SyncedAt     *time.Time  `json:"synced_at"`
}

type AccessLogFilters struct {
	Date        string
	Unit        string
	Status      VisitStatus
	VisitorName string
	Document    string
}

func (s VisitStatus) IsValid() bool {
	switch s {
	case VisitStatusInProgress, VisitStatusFinished, VisitStatusCanceled, VisitStatusBlocked:
		return true
	default:
		return false
	}
}

func (s SyncStatus) IsValid() bool {
	switch s {
	case SyncStatusPending, SyncStatusSynced, SyncStatusError:
		return true
	default:
		return false
	}
}
