package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"chateauneuf-portaria-backend/internal/domain"
)

type AccessLogRepository interface {
	Create(ctx context.Context, accessLog *domain.AccessLog) error
	UpsertImported(ctx context.Context, accessLog *domain.AccessLog) error
	List(ctx context.Context, filters domain.AccessLogFilters) ([]domain.AccessLog, error)
	ListOpen(ctx context.Context) ([]domain.AccessLog, error)
	FindByID(ctx context.Context, id int64) (*domain.AccessLog, error)
	Checkout(ctx context.Context, id int64, exitAt time.Time) (*domain.AccessLog, error)
	ListPendingSync(ctx context.Context, limit int) ([]domain.AccessLog, error)
	MarkSynced(ctx context.Context, id int64, syncedAt time.Time) error
	MarkSyncError(ctx context.Context, id int64, syncError string) error
	SyncStats(ctx context.Context) (SyncStats, error)
}

type SyncService interface {
	RunOnce(ctx context.Context) error
	ImportAccessLogs(ctx context.Context) (int, error)
	ImportResidents(ctx context.Context) (int, error)
	Status(ctx context.Context) (SyncStatus, error)
}

type AccessLogService struct {
	repository AccessLogRepository
	sync       SyncService
}

type CreateAccessLogInput struct {
	VisitorName  string `json:"visitor_name"`
	Document     string `json:"document"`
	Company      string `json:"company"`
	Phone        string `json:"phone"`
	Unit         string `json:"unit"`
	ResidentName string `json:"resident_name"`
	ServiceType  string `json:"service_type"`
	VehiclePlate string `json:"vehicle_plate"`
	AuthorizedBy string `json:"authorized_by"`
	Doorman      string `json:"doorman"`
	Photo        string `json:"photo"`
	EntryAt      string `json:"entry_at"`
}

func NewAccessLogService(repository AccessLogRepository, sync SyncService) *AccessLogService {
	return &AccessLogService{repository: repository, sync: sync}
}

func (s *AccessLogService) Create(ctx context.Context, input CreateAccessLogInput) (*domain.AccessLog, error) {
	if strings.TrimSpace(input.VisitorName) == "" || strings.TrimSpace(input.Document) == "" || strings.TrimSpace(input.Unit) == "" {
		return nil, domain.ErrInvalidInput
	}

	entryAt := time.Now()
	if input.EntryAt != "" {
		parsed, err := time.Parse(time.RFC3339, input.EntryAt)
		if err != nil {
			return nil, domain.ErrInvalidInput
		}
		entryAt = parsed
	}

	accessLog := &domain.AccessLog{
		ExternalID:   newAccessLogExternalID(),
		VisitorName:  strings.TrimSpace(input.VisitorName),
		Document:     strings.TrimSpace(input.Document),
		Company:      strings.TrimSpace(input.Company),
		Phone:        strings.TrimSpace(input.Phone),
		Unit:         strings.TrimSpace(input.Unit),
		ResidentName: strings.TrimSpace(input.ResidentName),
		ServiceType:  strings.TrimSpace(input.ServiceType),
		VehiclePlate: strings.TrimSpace(input.VehiclePlate),
		AuthorizedBy: strings.TrimSpace(input.AuthorizedBy),
		Doorman:      strings.TrimSpace(input.Doorman),
		Photo:        strings.TrimSpace(input.Photo),
		EntryAt:      entryAt,
		VisitStatus:  domain.VisitStatusInProgress,
		SyncStatus:   domain.SyncStatusPending,
	}

	if err := s.repository.Create(ctx, accessLog); err != nil {
		return nil, err
	}

	go func(id int64) {
		runCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = s.sync.RunOnce(runCtx)
	}(accessLog.ID)

	return accessLog, nil
}

func newAccessLogExternalID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Sprintf("visit-%d", time.Now().UnixNano())
	}
	return "visit-" + hex.EncodeToString(raw[:])
}

func (s *AccessLogService) List(ctx context.Context, filters domain.AccessLogFilters) ([]domain.AccessLog, error) {
	if filters.Status != "" && !filters.Status.IsValid() {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.List(ctx, filters)
}

func (s *AccessLogService) ListOpen(ctx context.Context) ([]domain.AccessLog, error) {
	return s.repository.ListOpen(ctx)
}

func (s *AccessLogService) Checkout(ctx context.Context, id int64) (*domain.AccessLog, error) {
	if id <= 0 {
		return nil, domain.ErrInvalidInput
	}

	accessLog, err := s.repository.Checkout(ctx, id, time.Now())
	if errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	runCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := s.sync.RunOnce(runCtx); err != nil {
		return accessLog, nil
	}
	if syncedAccessLog, err := s.repository.FindByID(ctx, id); err == nil {
		accessLog = syncedAccessLog
	}

	return accessLog, nil
}
