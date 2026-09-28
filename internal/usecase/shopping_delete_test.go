package usecase

import (
	"context"
	"errors"
	"testing"
)

type deletionRepo struct {
	ShoppingRepository
	calls int
}

func (r *deletionRepo) Delete(ctx context.Context, id string) error { r.calls++; return nil }
func TestShoppingDeletePassword(t *testing.T) {
	repo := &deletionRepo{}
	service := NewShoppingService(repo)
	service.passwordVerifier = "bab5121981ecfb09398732ffdf7a488d:8f90271e50d9a4f6f6cfbafd02625c9c0db0e5317c6c68d9c9001647bae35e02"
	for _, password := range []string{"", "wrong"} {
		if err := service.Delete(context.Background(), "1", password); !errors.Is(err, ErrInventoryUnauthorized) {
			t.Fatal(err)
		}
	}
	if repo.calls != 0 {
		t.Fatal("unauthorized delete")
	}
	if err := service.Delete(context.Background(), "1", "inventory-test-secret"); err != nil {
		t.Fatal(err)
	}
	if repo.calls != 1 {
		t.Fatal("authorized deletion not executed")
	}
}
