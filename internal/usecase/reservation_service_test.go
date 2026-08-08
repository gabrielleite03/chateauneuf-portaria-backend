package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"chateauneuf-portaria-backend/internal/domain"
)

type reservationRepositoryStub struct {
	reservation *domain.CommonAreaReservation
	occupied    bool
	created     *domain.CommonAreaReservation
	updated     domain.ReservationStatus
	deleted     bool
}

func (r *reservationRepositoryStub) List(context.Context) ([]domain.CommonAreaReservation, error) {
	return nil, nil
}
func (r *reservationRepositoryStub) Create(_ context.Context, reservation domain.CommonAreaReservation) (*domain.CommonAreaReservation, error) {
	r.created = &reservation
	return &reservation, nil
}
func (r *reservationRepositoryStub) GetByID(context.Context, string) (*domain.CommonAreaReservation, error) {
	if r.reservation == nil {
		return nil, domain.ErrNotFound
	}
	return r.reservation, nil
}
func (r *reservationRepositoryStub) HasActiveConflict(context.Context, string, string, string) (bool, error) {
	return r.occupied, nil
}
func (r *reservationRepositoryStub) UpdateStatus(_ context.Context, _ string, status domain.ReservationStatus) (*domain.CommonAreaReservation, error) {
	r.updated = status
	copy := *r.reservation
	copy.Status = status
	return &copy, nil
}
func (r *reservationRepositoryStub) Delete(context.Context, string) error {
	r.deleted = true
	return nil
}

func validReservationInput(date string) CreateReservationInput {
	return CreateReservationInput{
		Area: "Churrasqueira", ResidentName: "Morador", Unit: "101",
		ReservationDate: date, StartTime: "09:00", EndTime: "18:00",
	}
}

func dateFromToday(days int) string {
	return time.Now().AddDate(0, 0, days).Format("2006-01-02")
}

func TestCreateReservationAcceptsDatesThroughThirtyDays(t *testing.T) {
	for _, days := range []int{0, 30} {
		repository := &reservationRepositoryStub{}
		service := NewReservationService(repository)
		if _, err := service.Create(context.Background(), validReservationInput(dateFromToday(days))); err != nil {
			t.Fatalf("day %d should be accepted: %v", days, err)
		}
	}
}

func TestCreateReservationRejectsDatesOutsideAllowedWindow(t *testing.T) {
	for _, days := range []int{-1, 31} {
		service := NewReservationService(&reservationRepositoryStub{})
		_, err := service.Create(context.Background(), validReservationInput(dateFromToday(days)))
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("day %d should be rejected, got %v", days, err)
		}
	}
}

func TestCreateReservationRejectsOccupiedDate(t *testing.T) {
	service := NewReservationService(&reservationRepositoryStub{occupied: true})
	_, err := service.Create(context.Background(), validReservationInput(dateFromToday(10)))
	if !errors.Is(err, domain.ErrReservationDateUnavailable) {
		t.Fatalf("expected occupied date error, got %v", err)
	}
}

func TestCancellationRequiresSevenDaysNotice(t *testing.T) {
	for _, test := range []struct {
		days    int
		allowed bool
	}{{6, false}, {7, true}} {
		repository := &reservationRepositoryStub{reservation: &domain.CommonAreaReservation{
			ID: "1", ReservationDate: dateFromToday(test.days), Status: domain.ReservationStatusBooked,
		}}
		service := NewReservationService(repository)
		_, err := service.UpdateStatus(context.Background(), UpdateReservationStatusInput{ID: "1", Status: "cancelada"})
		if test.allowed && err != nil {
			t.Fatalf("day %d should allow cancellation: %v", test.days, err)
		}
		if !test.allowed && !errors.Is(err, domain.ErrCancellationDeadline) {
			t.Fatalf("day %d should reject cancellation, got %v", test.days, err)
		}
	}
}

func TestDeleteRejectsActiveReservation(t *testing.T) {
	repository := &reservationRepositoryStub{reservation: &domain.CommonAreaReservation{ID: "1", Status: domain.ReservationStatusBooked}}
	service := NewReservationService(repository)
	err := service.Delete(context.Background(), DeleteReservationInput{ID: "1"})
	if !errors.Is(err, domain.ErrActiveReservationDeletion) || repository.deleted {
		t.Fatalf("active reservation deletion should be rejected, got %v", err)
	}
}
