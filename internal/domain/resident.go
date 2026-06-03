package domain

import "time"

type Resident struct {
	Unit          string     `json:"unit"`
	Owner         string     `json:"owner"`
	Phones        string     `json:"phones"`
	Tenant        string     `json:"tenant"`
	FamilyMembers string     `json:"familyMembers"`
	Photo         string     `json:"photo"`
	SyncStatus    SyncStatus `json:"-"`
	LastUpdated   time.Time  `json:"lastUpdated"`
}
