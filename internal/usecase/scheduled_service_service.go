package usecase

import (
	"context"
	"strings"

	"chateauneuf-portaria-backend/internal/domain"
)

type ScheduledServiceRepository interface {
	List(ctx context.Context) ([]domain.ScheduledService, error)
	Create(ctx context.Context, service domain.ScheduledService) (*domain.ScheduledService, error)
	UpdateStatus(ctx context.Context, id string, status domain.ScheduledServiceStatus, photo string, arrivalTime string) (*domain.ScheduledService, error)
	Delete(ctx context.Context, id string) error
}

type ScheduledServiceService struct {
	repository ScheduledServiceRepository
}

type CreateScheduledServiceInput struct {
	Date         string `json:"date"`
	Name         string `json:"name"`
	Document     string `json:"document"`
	Company      string `json:"company"`
	Unit         string `json:"unit"`
	AuthorizedBy string `json:"authorizedBy"`
	Notes        string `json:"notes"`
}

type UpdateScheduledServiceStatusInput struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Photo       string `json:"photo"`
	ArrivalTime string `json:"arrivalTime"`
}

type DeleteScheduledServiceInput struct {
	ID string `json:"id"`
}

func NewScheduledServiceService(repository ScheduledServiceRepository) *ScheduledServiceService {
	return &ScheduledServiceService{repository: repository}
}

func (s *ScheduledServiceService) List(ctx context.Context) ([]domain.ScheduledService, error) {
	return s.repository.List(ctx)
}

func (s *ScheduledServiceService) Create(ctx context.Context, input CreateScheduledServiceInput) (*domain.ScheduledService, error) {
	if strings.TrimSpace(input.Date) == "" ||
		strings.TrimSpace(input.Name) == "" ||
		strings.TrimSpace(input.Document) == "" ||
		strings.TrimSpace(input.Company) == "" ||
		strings.TrimSpace(input.Unit) == "" ||
		strings.TrimSpace(input.AuthorizedBy) == "" {
		return nil, domain.ErrInvalidInput
	}

	return s.repository.Create(ctx, domain.ScheduledService{
		Date:         strings.TrimSpace(input.Date),
		Name:         strings.TrimSpace(input.Name),
		Document:     strings.TrimSpace(input.Document),
		Company:      strings.TrimSpace(input.Company),
		Unit:         strings.TrimSpace(input.Unit),
		AuthorizedBy: strings.TrimSpace(input.AuthorizedBy),
		Notes:        strings.TrimSpace(input.Notes),
		Status:       domain.ScheduledServiceStatusScheduled,
		SyncStatus:   domain.SyncStatusPending,
	})
}

func (s *ScheduledServiceService) UpdateStatus(ctx context.Context, input UpdateScheduledServiceStatusInput) (*domain.ScheduledService, error) {
	status := domain.ScheduledServiceStatus(strings.TrimSpace(input.Status))
	if strings.TrimSpace(input.ID) == "" || !status.IsValid() || status == domain.ScheduledServiceStatusScheduled {
		return nil, domain.ErrInvalidInput
	}

	return s.repository.UpdateStatus(ctx, strings.TrimSpace(input.ID), status, strings.TrimSpace(input.Photo), strings.TrimSpace(input.ArrivalTime))
}

func (s *ScheduledServiceService) Delete(ctx context.Context, input DeleteScheduledServiceInput) error {
	if strings.TrimSpace(input.ID) == "" {
		return domain.ErrInvalidInput
	}
	return s.repository.Delete(ctx, strings.TrimSpace(input.ID))
}
