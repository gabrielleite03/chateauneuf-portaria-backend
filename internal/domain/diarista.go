package domain

import "time"

type DiaristaEntry struct {
	ID           string     `json:"id"`
	Date         string     `json:"date"`
	Name         string     `json:"name"`
	RG           string     `json:"rg"`
	Unit         string     `json:"unit"`
	AuthorizedBy string     `json:"authorizedBy"`
	EntryTime    string     `json:"entryTime"`
	ExitTime     string     `json:"exitTime,omitempty"`
	Gatekeeper   string     `json:"gatekeeper"`
	Photo        string     `json:"photo,omitempty"`
	SyncStatus   SyncStatus `json:"syncStatus"`
	CreatedAt    time.Time  `json:"-"`
	UpdatedAt    time.Time  `json:"-"`
}
