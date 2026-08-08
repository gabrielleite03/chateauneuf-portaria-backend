package usecase

import (
	"context"
	"strings"
	"time"

	"chateauneuf-portaria-backend/internal/domain"
)

type ReservationRepository interface {
	List(ctx context.Context) ([]domain.CommonAreaReservation, error)
	Create(ctx context.Context, reservation domain.CommonAreaReservation) (*domain.CommonAreaReservation, error)
	GetByID(ctx context.Context, id string) (*domain.CommonAreaReservation, error)
	HasActiveConflict(ctx context.Context, reservationDate, area, unit string) (bool, error)
	UpdateStatus(ctx context.Context, id string, status domain.ReservationStatus) (*domain.CommonAreaReservation, error)
	Delete(ctx context.Context, id string) error
}

const (
	maxReservationAdvanceDays = 30
	cancellationLimitDays     = 7
)

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
	reservationDateText := strings.TrimSpace(input.ReservationDate)
	reservationDate, err := time.ParseInLocation("2006-01-02", reservationDateText, time.Local)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}
	today := startOfDay(time.Now())
	daysAhead := daysBetween(today, reservationDate)
	if daysAhead < 0 || daysAhead > maxReservationAdvanceDays || input.StartTime >= input.EndTime {
		return nil, domain.ErrInvalidInput
	}
	occupied, err := s.repository.HasActiveConflict(ctx, reservationDateText, area, strings.TrimSpace(input.Unit))
	if err != nil {
		return nil, err
	}
	if occupied {
		return nil, domain.ErrReservationDateUnavailable
	}

	return s.repository.Create(ctx, domain.CommonAreaReservation{
		Area:            area,
		ResidentName:    strings.TrimSpace(input.ResidentName),
		Unit:            strings.TrimSpace(input.Unit),
		ReservationDate: reservationDateText,
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
	if status == domain.ReservationStatusCanceled {
		reservation, err := s.repository.GetByID(ctx, strings.TrimSpace(input.ID))
		if err != nil {
			return nil, err
		}
		reservationDate, err := time.ParseInLocation("2006-01-02", reservation.ReservationDate, time.Local)
		if err != nil {
			return nil, domain.ErrInvalidInput
		}
		if daysBetween(startOfDay(time.Now()), reservationDate) < cancellationLimitDays {
			return nil, domain.ErrCancellationDeadline
		}
	}
	return s.repository.UpdateStatus(ctx, strings.TrimSpace(input.ID), status)
}

func (s *ReservationService) Delete(ctx context.Context, input DeleteReservationInput) error {
	if strings.TrimSpace(input.ID) == "" {
		return domain.ErrInvalidInput
	}
	reservation, err := s.repository.GetByID(ctx, strings.TrimSpace(input.ID))
	if err != nil {
		return err
	}
	if reservation.Status == domain.ReservationStatusBooked {
		return domain.ErrActiveReservationDeletion
	}
	return s.repository.Delete(ctx, strings.TrimSpace(input.ID))
}

func startOfDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func daysBetween(from, to time.Time) int {
	fromUTC := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	toUTC := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	return int(toUTC.Sub(fromUTC).Hours() / 24)
}
