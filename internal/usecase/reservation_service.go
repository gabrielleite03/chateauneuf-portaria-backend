package usecase

import (
	"context"
	"strings"

	"chateauneuf-portaria-backend/internal/domain"
)

type ReservationRepository interface {
	List(ctx context.Context) ([]domain.CommonAreaReservation, error)
	Create(ctx context.Context, reservation domain.CommonAreaReservation) (*domain.CommonAreaReservation, error)
	UpdateStatus(ctx context.Context, id string, status domain.ReservationStatus) (*domain.CommonAreaReservation, error)
	Delete(ctx context.Context, id string) error
}

type ReservationService struct {
	repository ReservationRepository
}

type CreateReservationInput struct {
	Area            string `json:"area"`
	ResidentName    string `json:"residentName"`
	Unit            string `json:"unit"`
	ReservationDate string `json:"reservationDate"`
	StartTime       string `json:"startTime"`
	EndTime         string `json:"endTime"`
	Guests          string `json:"guests"`
	Notes           string `json:"notes"`
}

type UpdateReservationStatusInput struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type DeleteReservationInput struct {
	ID string `json:"id"`
}

func NewReservationService(repository ReservationRepository) *ReservationService {
	return &ReservationService{repository: repository}
}

func (s *ReservationService) List(ctx context.Context) ([]domain.CommonAreaReservation, error) {
	return s.repository.List(ctx)
}

func (s *ReservationService) Create(ctx context.Context, input CreateReservationInput) (*domain.CommonAreaReservation, error) {
	area := strings.TrimSpace(input.Area)
	if area != "Churrasqueira" && area != "Salão de festas" && area != "Salao de festas" {
		return nil, domain.ErrInvalidInput
	}
	if strings.TrimSpace(input.ResidentName) == "" ||
		strings.TrimSpace(input.Unit) == "" ||
		strings.TrimSpace(input.ReservationDate) == "" ||
		strings.TrimSpace(input.StartTime) == "" ||
		strings.TrimSpace(input.EndTime) == "" {
		return nil, domain.ErrInvalidInput
	}
	if area == "Salao de festas" {
		area = "Salão de festas"
	}

	return s.repository.Create(ctx, domain.CommonAreaReservation{
		Area:            area,
		ResidentName:    strings.TrimSpace(input.ResidentName),
		Unit:            strings.TrimSpace(input.Unit),
		ReservationDate: strings.TrimSpace(input.ReservationDate),
		StartTime:       strings.TrimSpace(input.StartTime),
		EndTime:         strings.TrimSpace(input.EndTime),
		Guests:          strings.TrimSpace(input.Guests),
		Notes:           strings.TrimSpace(input.Notes),
		Status:          domain.ReservationStatusBooked,
		SyncStatus:      domain.SyncStatusPending,
	})
}

func (s *ReservationService) UpdateStatus(ctx context.Context, input UpdateReservationStatusInput) (*domain.CommonAreaReservation, error) {
	status := domain.ReservationStatus(strings.TrimSpace(input.Status))
	if strings.TrimSpace(input.ID) == "" || !status.IsValid() {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.UpdateStatus(ctx, strings.TrimSpace(input.ID), status)
}

func (s *ReservationService) Delete(ctx context.Context, input DeleteReservationInput) error {
	if strings.TrimSpace(input.ID) == "" {
		return domain.ErrInvalidInput
	}
	return s.repository.Delete(ctx, strings.TrimSpace(input.ID))
}
