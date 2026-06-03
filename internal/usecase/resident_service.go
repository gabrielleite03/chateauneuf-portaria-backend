package usecase

import (
	"context"
	"strings"

	"chateauneuf-portaria-backend/internal/domain"
)

type ResidentRepository interface {
	List(ctx context.Context) ([]domain.Resident, error)
	Upsert(ctx context.Context, resident domain.Resident) (*domain.Resident, error)
}

type ResidentService struct {
	repository ResidentRepository
}

type UpsertResidentInput struct {
	Unit          string `json:"unit"`
	Owner         string `json:"owner"`
	Phones        string `json:"phones"`
	Tenant        string `json:"tenant"`
	FamilyMembers string `json:"familyMembers"`
	Photo         string `json:"photo"`
}

func NewResidentService(repository ResidentRepository) *ResidentService {
	return &ResidentService{repository: repository}
}

func (s *ResidentService) List(ctx context.Context) ([]domain.Resident, error) {
	return s.repository.List(ctx)
}

func (s *ResidentService) Upsert(ctx context.Context, input UpsertResidentInput) (*domain.Resident, error) {
	if strings.TrimSpace(input.Unit) == "" {
		return nil, domain.ErrInvalidInput
	}

	return s.repository.Upsert(ctx, domain.Resident{
		Unit:          strings.TrimSpace(input.Unit),
		Owner:         strings.TrimSpace(input.Owner),
		Phones:        strings.TrimSpace(input.Phones),
		Tenant:        strings.TrimSpace(input.Tenant),
		FamilyMembers: strings.TrimSpace(input.FamilyMembers),
		Photo:         strings.TrimSpace(input.Photo),
		SyncStatus:    domain.SyncStatusSynced,
	})
}
