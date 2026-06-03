package domain

import "time"

type KeyStatus string

const (
	KeyStatusPickedUp KeyStatus = "retirada"
	KeyStatusReturned KeyStatus = "devolvida"
)

type KeyRecord struct {
	ID           string     `json:"id"`
	Date         string     `json:"date"`
	Local        string     `json:"local"`
	ResidentName string     `json:"residentName"`
	Unit         string     `json:"unit"`
	PickupTime   string     `json:"pickupTime"`
	ReturnTime   string     `json:"returnTime,omitempty"`
	Gatekeeper   string     `json:"gatekeeper"`
	Status       KeyStatus  `json:"status"`
	SyncStatus   SyncStatus `json:"syncStatus"`
	CreatedAt    time.Time  `json:"-"`
	UpdatedAt    time.Time  `json:"-"`
}

func (s KeyStatus) IsValid() bool {
	switch s {
	case KeyStatusPickedUp, KeyStatusReturned:
		return true
	default:
		return false
	}
}
