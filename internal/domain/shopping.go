package domain

import "time"

type ShoppingStatus string

const (
	ShoppingStatusWaiting   ShoppingStatus = "aguardando_retirada"
	ShoppingStatusWithdrawn ShoppingStatus = "retirada"
)

type ShoppingDelivery struct {
	ID          string         `json:"id"`
	Unit        string         `json:"unit"`
	CourierName string         `json:"courierName"`
	Document    string         `json:"document"`
	Store       string         `json:"store"`
	Product     string         `json:"product"`
	Notes       string         `json:"notes,omitempty"`
	Photo       string         `json:"photo,omitempty"`
	ReceivedAt  time.Time      `json:"receivedAt"`
	WithdrawnAt *time.Time     `json:"withdrawnAt,omitempty"`
	Status      ShoppingStatus `json:"status"`
	SyncStatus  SyncStatus     `json:"syncStatus"`
	CreatedAt   time.Time      `json:"-"`
	UpdatedAt   time.Time      `json:"-"`
}

func (s ShoppingStatus) IsValid() bool {
	switch s {
	case ShoppingStatusWaiting, ShoppingStatusWithdrawn:
		return true
	default:
		return false
	}
}
