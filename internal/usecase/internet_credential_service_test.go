package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"chateauneuf-portaria-backend/internal/domain"
)

type credentialResidentRepository struct{ rows []domain.Resident }

func (r credentialResidentRepository) List(context.Context) ([]domain.Resident, error) {
	return r.rows, nil
}
func (r credentialResidentRepository) Upsert(context.Context, domain.Resident) (*domain.Resident, error) {
	return nil, nil
}

type recordingEmailSender struct{ to string }

func (s *recordingEmailSender) Send(_ context.Context, to, _, _ string) error { s.to = to; return nil }

func TestInternetCredentialRecipientPriority(t *testing.T) {
	tests := []struct {
		name     string
		resident domain.Resident
		want     string
	}{
		{"tenant", domain.Resident{Unit: "13", Owner: "Owner", Email: "owner@example.com", Tenant: "Tenant", TenantEmail: "tenant@example.com"}, "tenant@example.com"},
		{"owner fallback", domain.Resident{Unit: "13", Owner: "Owner", Email: "owner@example.com", Tenant: "Tenant", TenantEmail: "invalid"}, "owner@example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &recordingEmailSender{}
			service := NewInternetCredentialService(NewResidentService(credentialResidentRepository{rows: []domain.Resident{tt.resident}}), sender)
			if _, err := service.Send(context.Background(), "13", "apto13", "ABC123", time.Now().Add(24*time.Hour)); err != nil {
				t.Fatal(err)
			}
			if sender.to != tt.want {
				t.Fatalf("got %q want %q", sender.to, tt.want)
			}
		})
	}
}

func TestInternetCredentialRequiresValidEmail(t *testing.T) {
	service := NewInternetCredentialService(NewResidentService(credentialResidentRepository{rows: []domain.Resident{{Unit: "13", Owner: "Owner"}}}), &recordingEmailSender{})
	_, err := service.Send(context.Background(), "13", "apto13", "ABC123", time.Now())
	if !errors.Is(err, ErrInternetCredentialEmailRequired) {
		t.Fatalf("expected email required error, got %v", err)
	}
}
