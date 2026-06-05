package sync

import (
	"context"
	"log/slog"
	"time"

	"chateauneuf-portaria-backend/internal/domain"
	"chateauneuf-portaria-backend/internal/usecase"
)

type SpreadsheetClient interface {
	Ping(ctx context.Context) error
	ReadAccessLogs(ctx context.Context) ([]domain.AccessLog, error)
	AppendAccessLog(ctx context.Context, accessLog domain.AccessLog) error
	AppendDiaristaEntry(ctx context.Context, entry domain.DiaristaEntry) error
	AppendKeyRecord(ctx context.Context, key domain.KeyRecord) error
	AppendScheduledService(ctx context.Context, service domain.ScheduledService) error
	AppendShoppingDelivery(ctx context.Context, delivery domain.ShoppingDelivery) error
}

type Service struct {
	repository                 usecase.AccessLogRepository
	diaristaRepository         DiaristaRepository
	keyRepository              KeyRepository
	scheduledServiceRepository ScheduledServiceRepository
	shoppingRepository         ShoppingRepository
	client                     SpreadsheetClient
	logger                     *slog.Logger
}

type DiaristaRepository interface {
	ListPendingSync(ctx context.Context, limit int) ([]domain.DiaristaEntry, error)
	MarkSynced(ctx context.Context, id string, syncedAt time.Time) error
	MarkSyncError(ctx context.Context, id string, syncError string) error
	SyncStats(ctx context.Context) (int, error)
}

type KeyRepository interface {
	ListPendingSync(ctx context.Context, limit int) ([]domain.KeyRecord, error)
	MarkSynced(ctx context.Context, id string, syncedAt time.Time) error
	MarkSyncError(ctx context.Context, id string, syncError string) error
	SyncStats(ctx context.Context) (int, error)
}

type ScheduledServiceRepository interface {
	ListPendingSync(ctx context.Context, limit int) ([]domain.ScheduledService, error)
	MarkSynced(ctx context.Context, id string, syncedAt time.Time) error
	MarkSyncError(ctx context.Context, id string, syncError string) error
	SyncStats(ctx context.Context) (int, error)
}

type ShoppingRepository interface {
	ListPendingSync(ctx context.Context, limit int) ([]domain.ShoppingDelivery, error)
	MarkSynced(ctx context.Context, id string, syncedAt time.Time) error
	MarkSyncError(ctx context.Context, id string, syncError string) error
	SyncStats(ctx context.Context) (int, error)
}

func NewService(repository usecase.AccessLogRepository, diaristaRepository DiaristaRepository, keyRepository KeyRepository, scheduledServiceRepository ScheduledServiceRepository, shoppingRepository ShoppingRepository, client SpreadsheetClient, logger *slog.Logger) *Service {
	return &Service{
		repository:                 repository,
		diaristaRepository:         diaristaRepository,
		keyRepository:              keyRepository,
		scheduledServiceRepository: scheduledServiceRepository,
		shoppingRepository:         shoppingRepository,
		client:                     client,
		logger:                     logger,
	}
}

func (s *Service) Start(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.RunOnce(ctx); err != nil {
					s.logger.Warn("sync failed", "error", err)
				}
			}
		}
	}()
}

func (s *Service) RunOnce(ctx context.Context) error {
	if err := s.client.Ping(ctx); err != nil {
		return nil
	}

	accessLogs, err := s.repository.ListPendingSync(ctx, 50)
	if err != nil {
		return err
	}

	for _, accessLog := range accessLogs {
		if err := s.client.AppendAccessLog(ctx, accessLog); err != nil {
			_ = s.repository.MarkSyncError(ctx, accessLog.ID, err.Error())
			continue
		}
		if err := s.repository.MarkSynced(ctx, accessLog.ID, time.Now()); err != nil {
			return err
		}
	}

	if s.diaristaRepository != nil {
		diaristas, err := s.diaristaRepository.ListPendingSync(ctx, 50)
		if err != nil {
			return err
		}

		for _, diarista := range diaristas {
			if err := s.client.AppendDiaristaEntry(ctx, diarista); err != nil {
				_ = s.diaristaRepository.MarkSyncError(ctx, diarista.ID, err.Error())
				continue
			}
			if err := s.diaristaRepository.MarkSynced(ctx, diarista.ID, time.Now()); err != nil {
				return err
			}
		}
	}

	if s.keyRepository != nil {
		keys, err := s.keyRepository.ListPendingSync(ctx, 50)
		if err != nil {
			return err
		}

		for _, key := range keys {
			if err := s.client.AppendKeyRecord(ctx, key); err != nil {
				_ = s.keyRepository.MarkSyncError(ctx, key.ID, err.Error())
				continue
			}
			if err := s.keyRepository.MarkSynced(ctx, key.ID, time.Now()); err != nil {
				return err
			}
		}
	}

	if s.scheduledServiceRepository != nil {
		services, err := s.scheduledServiceRepository.ListPendingSync(ctx, 50)
		if err != nil {
			return err
		}

		for _, service := range services {
			if err := s.client.AppendScheduledService(ctx, service); err != nil {
				_ = s.scheduledServiceRepository.MarkSyncError(ctx, service.ID, err.Error())
				continue
			}
			if err := s.scheduledServiceRepository.MarkSynced(ctx, service.ID, time.Now()); err != nil {
				return err
			}
		}
	}

	if s.shoppingRepository != nil {
		deliveries, err := s.shoppingRepository.ListPendingSync(ctx, 50)
		if err != nil {
			return err
		}

		for _, delivery := range deliveries {
			if err := s.client.AppendShoppingDelivery(ctx, delivery); err != nil {
				_ = s.shoppingRepository.MarkSyncError(ctx, delivery.ID, err.Error())
				continue
			}
			if err := s.shoppingRepository.MarkSynced(ctx, delivery.ID, time.Now()); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Service) ImportAccessLogs(ctx context.Context) (int, error) {
	if err := s.client.Ping(ctx); err != nil {
		return 0, err
	}

	accessLogs, err := s.client.ReadAccessLogs(ctx)
	if err != nil {
		return 0, err
	}

	importedCount := 0
	for index := range accessLogs {
		if accessLogs[index].ID <= 0 {
			continue
		}
		if err := s.repository.UpsertImported(ctx, &accessLogs[index]); err != nil {
			return importedCount, err
		}
		importedCount++
	}

	return importedCount, nil
}

func (s *Service) Status(ctx context.Context) (usecase.SyncStatus, error) {
	stats, err := s.repository.SyncStats(ctx)
	if err != nil {
		return usecase.SyncStatus{}, err
	}
	if s.diaristaRepository != nil {
		diaristaPendingCount, err := s.diaristaRepository.SyncStats(ctx)
		if err != nil {
			return usecase.SyncStatus{}, err
		}
		stats.PendingCount += diaristaPendingCount
	}
	if s.keyRepository != nil {
		keyPendingCount, err := s.keyRepository.SyncStats(ctx)
		if err != nil {
			return usecase.SyncStatus{}, err
		}
		stats.PendingCount += keyPendingCount
	}
	if s.scheduledServiceRepository != nil {
		scheduledPendingCount, err := s.scheduledServiceRepository.SyncStats(ctx)
		if err != nil {
			return usecase.SyncStatus{}, err
		}
		stats.PendingCount += scheduledPendingCount
	}
	if s.shoppingRepository != nil {
		shoppingPendingCount, err := s.shoppingRepository.SyncStats(ctx)
		if err != nil {
			return usecase.SyncStatus{}, err
		}
		stats.PendingCount += shoppingPendingCount
	}

	pingErr := s.client.Ping(ctx)
	lastError := stats.LastError
	if pingErr != nil {
		lastError = pingErr.Error()
	}

	return usecase.SyncStatus{
		Online:       pingErr == nil,
		PendingCount: stats.PendingCount,
		LastSyncedAt: stats.LastSyncedAt,
		LastError:    lastError,
	}, nil
}
