package usecase

import (
	"context"
	"strings"

	"chateauneuf-portaria-backend/internal/domain"
)

type DiaristaRepository interface {
	List(ctx context.Context) ([]domain.DiaristaEntry, error)
	Create(ctx context.Context, entry domain.DiaristaEntry) (*domain.DiaristaEntry, error)
	Checkout(ctx context.Context, id string, exitTime string) (*domain.DiaristaEntry, error)
}

type DiaristaService struct {
	repository DiaristaRepository
}

type CreateDiaristaInput struct {
	Date         string `json:"date"`
	Name         string `json:"name"`
	RG           string `json:"rg"`
	Unit         string `json:"unit"`
	AuthorizedBy string `json:"authorizedBy"`
	EntryTime    string `json:"entryTime"`
	Gatekeeper   string `json:"gatekeeper"`
	Photo        string `json:"photo"`
}

type CheckoutDiaristaInput struct {
	ID       string `json:"id"`
	ExitTime string `json:"exitTime"`
}

func NewDiaristaService(repository DiaristaRepository) *DiaristaService {
	return &DiaristaService{repository: repository}
}

func (s *DiaristaService) List(ctx context.Context) ([]domain.DiaristaEntry, error) {
	return s.repository.List(ctx)
}

func (s *DiaristaService) Create(ctx context.Context, input CreateDiaristaInput) (*domain.DiaristaEntry, error) {
	if strings.TrimSpace(input.Date) == "" ||
		strings.TrimSpace(input.Name) == "" ||
		strings.TrimSpace(input.RG) == "" ||
		strings.TrimSpace(input.Unit) == "" ||
		strings.TrimSpace(input.AuthorizedBy) == "" ||
		strings.TrimSpace(input.EntryTime) == "" ||
		strings.TrimSpace(input.Gatekeeper) == "" {
		return nil, domain.ErrInvalidInput
	}

	return s.repository.Create(ctx, domain.DiaristaEntry{
		Date:         strings.TrimSpace(input.Date),
		Name:         strings.TrimSpace(input.Name),
		RG:           strings.TrimSpace(input.RG),
		Unit:         strings.TrimSpace(input.Unit),
		AuthorizedBy: strings.TrimSpace(input.AuthorizedBy),
		EntryTime:    strings.TrimSpace(input.EntryTime),
		Gatekeeper:   strings.TrimSpace(input.Gatekeeper),
		Photo:        strings.TrimSpace(input.Photo),
		SyncStatus:   domain.SyncStatusPending,
	})
}

func (s *DiaristaService) Checkout(ctx context.Context, input CheckoutDiaristaInput) (*domain.DiaristaEntry, error) {
	if strings.TrimSpace(input.ID) == "" || strings.TrimSpace(input.ExitTime) == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.Checkout(ctx, strings.TrimSpace(input.ID), strings.TrimSpace(input.ExitTime))
}
