package usecase

import "time"

type SyncStats struct {
	PendingCount int        `json:"pending_count"`
	LastSyncedAt *time.Time `json:"last_synced_at"`
	LastError    string     `json:"last_error"`
}

type SyncStatus struct {
	Online       bool       `json:"online"`
	PendingCount int        `json:"pending_count"`
	LastSyncedAt *time.Time `json:"last_synced_at"`
	LastError    string     `json:"last_error"`
}
