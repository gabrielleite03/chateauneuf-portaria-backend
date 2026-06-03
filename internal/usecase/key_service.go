package usecase

import (
	"context"
	"strings"

	"chateauneuf-portaria-backend/internal/domain"
)

type KeyRepository interface {
	List(ctx context.Context) ([]domain.KeyRecord, error)
	Create(ctx context.Context, key domain.KeyRecord) (*domain.KeyRecord, error)
	Return(ctx context.Context, id string, returnTime string) (*domain.KeyRecord, error)
	Delete(ctx context.Context, id string) error
}

type KeyService struct {
	repository KeyRepository
}

type CreateKeyInput struct {
	Date         string `json:"date"`
	Local        string `json:"local"`
	ResidentName string `json:"residentName"`
	Unit         string `json:"unit"`
	PickupTime   string `json:"pickupTime"`
	Gatekeeper   string `json:"gatekeeper"`
}

type ReturnKeyInput struct {
	ID         string `json:"id"`
	ReturnTime string `json:"returnTime"`
}

type DeleteKeyInput struct {
	ID string `json:"id"`
}

func NewKeyService(repository KeyRepository) *KeyService {
	return &KeyService{repository: repository}
}

func (s *KeyService) List(ctx context.Context) ([]domain.KeyRecord, error) {
	return s.repository.List(ctx)
}

func (s *KeyService) Create(ctx context.Context, input CreateKeyInput) (*domain.KeyRecord, error) {
	if strings.TrimSpace(input.Date) == "" ||
		strings.TrimSpace(input.Local) == "" ||
		strings.TrimSpace(input.ResidentName) == "" ||
		strings.TrimSpace(input.Unit) == "" ||
		strings.TrimSpace(input.PickupTime) == "" ||
		strings.TrimSpace(input.Gatekeeper) == "" {
		return nil, domain.ErrInvalidInput
	}

	return s.repository.Create(ctx, domain.KeyRecord{
		Date:         strings.TrimSpace(input.Date),
		Local:        strings.TrimSpace(input.Local),
		ResidentName: strings.TrimSpace(input.ResidentName),
		Unit:         strings.TrimSpace(input.Unit),
		PickupTime:   strings.TrimSpace(input.PickupTime),
		Gatekeeper:   strings.TrimSpace(input.Gatekeeper),
		Status:       domain.KeyStatusPickedUp,
		SyncStatus:   domain.SyncStatusPending,
	})
}

func (s *KeyService) Return(ctx context.Context, input ReturnKeyInput) (*domain.KeyRecord, error) {
	if strings.TrimSpace(input.ID) == "" || strings.TrimSpace(input.ReturnTime) == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.Return(ctx, strings.TrimSpace(input.ID), strings.TrimSpace(input.ReturnTime))
}

func (s *KeyService) Delete(ctx context.Context, input DeleteKeyInput) error {
	if strings.TrimSpace(input.ID) == "" {
		return domain.ErrInvalidInput
	}
	return s.repository.Delete(ctx, strings.TrimSpace(input.ID))
}
