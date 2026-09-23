package usecase

import (
	"context"
	"errors"
	"strings"
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

type recordingEmailSender struct {
	to   string
	body string
}

func (s *recordingEmailSender) Send(_ context.Context, to, _, body string) error {
	s.to = to
	s.body = body
	return nil
}

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

func TestInternetCredentialEmailUsesPortalLayoutAndListsSSIDs(t *testing.T) {
	sender := &recordingEmailSender{}
	service := NewInternetCredentialService(NewResidentService(credentialResidentRepository{rows: []domain.Resident{{Unit: "13", Owner: "Maria & João", Email: "resident@example.com"}}}), sender)
	if _, err := service.Send(context.Background(), "13", "apto<13>", "A&B123", time.Date(2026, 12, 1, 12, 0, 0, 0, time.Local)); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"Condomínio Edifício Chateauneuf", "CHATEAUNEUF", "CHATEAUNEUF_5GHz", "Maria &amp; João", "apto&lt;13&gt;", "A&amp;B123", "90 dias", "01/12/2026"} {
		if !strings.Contains(sender.body, expected) {
			t.Errorf("email body does not contain %q", expected)
		}
	}
}

func TestInternetCredentialRequiresValidEmail(t *testing.T) {
	service := NewInternetCredentialService(NewResidentService(credentialResidentRepository{rows: []domain.Resident{{Unit: "13", Owner: "Owner"}}}), &recordingEmailSender{})
	_, err := service.Send(context.Background(), "13", "apto13", "ABC123", time.Now())
	if !errors.Is(err, ErrInternetCredentialEmailRequired) {
		t.Fatalf("expected email required error, got %v", err)
	}
}
