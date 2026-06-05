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
	repository ShoppingRepository
}

type CreateShoppingInput struct {
	Unit        string `json:"unit"`
	CourierName string `json:"courierName"`
	Document    string `json:"document"`
	Store       string `json:"store"`
	Product     string `json:"product"`
	Notes       string `json:"notes"`
	Photo       string `json:"photo"`
}

type WithdrawShoppingInput struct {
	ID string `json:"id"`
}

func NewShoppingService(repository ShoppingRepository) *ShoppingService {
	return &ShoppingService{repository: repository}
}

func (s *ShoppingService) List(ctx context.Context) ([]domain.ShoppingDelivery, error) {
	return s.repository.List(ctx)
}

func (s *ShoppingService) Create(ctx context.Context, input CreateShoppingInput) (*domain.ShoppingDelivery, error) {
	if strings.TrimSpace(input.Unit) == "" ||
		strings.TrimSpace(input.CourierName) == "" ||
		strings.TrimSpace(input.Document) == "" ||
		strings.TrimSpace(input.Store) == "" ||
		strings.TrimSpace(input.Product) == "" ||
		strings.TrimSpace(input.Photo) == "" {
		return nil, domain.ErrInvalidInput
	}

	return s.repository.Create(ctx, domain.ShoppingDelivery{
		Unit:        strings.TrimSpace(input.Unit),
		CourierName: strings.TrimSpace(input.CourierName),
		Document:    strings.TrimSpace(input.Document),
		Store:       strings.TrimSpace(input.Store),
		Product:     strings.TrimSpace(input.Product),
		Notes:       strings.TrimSpace(input.Notes),
		Photo:       strings.TrimSpace(input.Photo),
		Status:      domain.ShoppingStatusWaiting,
		SyncStatus:  domain.SyncStatusPending,
	})
}

func (s *ShoppingService) Withdraw(ctx context.Context, input WithdrawShoppingInput) (*domain.ShoppingDelivery, error) {
	if strings.TrimSpace(input.ID) == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.repository.Withdraw(ctx, strings.TrimSpace(input.ID))
}
