package domain

import "time"

type Resident struct {
	Unit                 string     `json:"unit"`
	Owner                string     `json:"owner"`
	Phones               string     `json:"phones"`
	Email                string     `json:"email"`
	Tenant               string     `json:"tenant"`
	TenantEmail          string     `json:"tenantEmail"`
	TenantPhone          string     `json:"tenantPhone"`
	TenantPhoto          string     `json:"tenantPhoto"`
	FamilyMembers        string     `json:"familyMembers"`
	AuthorizedRecipients string     `json:"authorizedRecipients"`
	Photo                string     `json:"photo"`
	SyncStatus           SyncStatus `json:"-"`
	LastUpdated          time.Time  `json:"lastUpdated"`
}
