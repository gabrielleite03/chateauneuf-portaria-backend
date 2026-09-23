package usecase

import (
	"context"
	"strings"

	"chateauneuf-portaria-backend/internal/domain"
)

type ShoppingRepository interface {
	List(ctx context.Context) ([]domain.ShoppingDelivery, error)
	Create(ctx context.Context, delivery domain.ShoppingDelivery) (*domain.ShoppingDelivery, error)
	Withdraw(ctx context.Context, id string) (*domain.ShoppingDelivery, error)
}

type ShoppingService struct {
	repository      ShoppingRepository
	arrivalNotifier interface {
		NotifyDeliveryArrival(context.Context, domain.ShoppingDelivery, string) error
	}
}

type CreateShoppingInput struct {
	Unit        string `json:"unit"`
	Recipient   string `json:"recipient"`
	CourierName string `json:"courierName"`
	Document    string `json:"document"`
	Store       string `json:"store"`
	Product     string `json:"product"`
	Notes       string `json:"notes"`
	Photo       string `json:"photo"`
	EmailPhoto  string `json:"-"`
}

type WithdrawShoppingInput struct {
	ID string `json:"id"`
}

func NewShoppingService(repository ShoppingRepository) *ShoppingService {
	return &ShoppingService{repository: repository}
}

func (s *ShoppingService) SetArrivalNotifier(notifier interface {
	NotifyDeliveryArrival(context.Context, domain.ShoppingDelivery, string) error
}) {
	s.arrivalNotifier = notifier
}

func (s *ShoppingService) List(ctx context.Context) ([]domain.ShoppingDelivery, error) {
	return s.repository.List(ctx)
}

func (s *ShoppingService) Create(ctx context.Context, input CreateShoppingInput) (*domain.ShoppingDelivery, error) {
	if strings.TrimSpace(input.Unit) == "" {
		return nil, domain.ErrInvalidInput
	}
	if strings.TrimSpace(input.Recipient) == "" {
		return nil, domain.ErrInvalidInput
	}

	delivery, err := s.repository.Create(ctx, domain.ShoppingDelivery{
		Unit:        strings.TrimSpace(input.Unit),
		Recipient:   strings.TrimSpace(input.Recipient),
		CourierName: strings.TrimSpace(input.CourierName),
		Document:    strings.TrimSpace(input.Document),
		Store:       strings.TrimSpace(input.Store),
		Product:     strings.TrimSpace(input.Product),
		Notes:       strings.TrimSpace(input.Notes),
		Photo:       strings.TrimSpace(input.Photo),
		Status:      domain.ShoppingStatusWaiting,
		SyncStatus:  domain.SyncStatusPending,
	})
	if err != nil {
		return nil, err
	}
	if s.arrivalNotifier != nil && strings.TrimSpace(input.EmailPhoto) != "" {
		_ = s.arrivalNotifier.NotifyDeliveryArrival(ctx, *delivery, input.EmailPhoto)
	}
	return delivery, nil
}

func (s *ShoppingService) Withdraw(ctx context.Context, input WithdrawShoppingInput) (*domain.ShoppingDelivery, error) {
	if strings.TrimSpace(input.ID) == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.Withdraw(ctx, strings.TrimSpace(input.ID))
}
