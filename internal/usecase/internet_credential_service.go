package usecase

import (
	"context"
	"errors"
	"fmt"
	"html"
	"net/mail"
	"strings"
	"time"
)

var ErrInternetCredentialEmailRequired = errors.New("e-mail do apartamento nao cadastrado")

type EmailSender interface {
	Send(ctx context.Context, to, subject, htmlBody string) error
}
type InternetCredentialService struct {
	residents *ResidentService
	mailer    EmailSender
}

func NewInternetCredentialService(residents *ResidentService, mailer EmailSender) *InternetCredentialService {
	return &InternetCredentialService{residents: residents, mailer: mailer}
}

func (s *InternetCredentialService) Send(ctx context.Context, apartment, username, password string, expiresAt time.Time) (string, error) {
	apartment = strings.TrimSpace(apartment)
	if apartment == "" || strings.TrimSpace(username) == "" || password == "" {
		return "", fmt.Errorf("dados da credencial incompletos: %w", ErrInternetCredentialEmailRequired)
	}
	rows, err := s.residents.List(ctx)
	if err != nil {
		return "", err
	}
	for _, resident := range rows {
		if !strings.EqualFold(strings.TrimSpace(resident.Unit), apartment) {
			continue
		}
		to, name := "", "Morador"
		if strings.TrimSpace(resident.Tenant) != "" && validEmail(resident.TenantEmail) {
			to, name = strings.TrimSpace(resident.TenantEmail), strings.TrimSpace(resident.Tenant)
		} else if validEmail(resident.Email) {
			to, name = strings.TrimSpace(resident.Email), strings.TrimSpace(resident.Owner)
		}
		if to == "" {
			return "", fmt.Errorf("cadastre um e-mail valido para o inquilino ou morador do apartamento %s: %w", apartment, ErrInternetCredentialEmailRequired)
		}
		body := fmt.Sprintf(`<p>Olá, %s.</p><p>O acesso à internet do apartamento <strong>%s</strong> foi atualizado.</p><p>Usuário: <strong>%s</strong><br>Senha: <strong>%s</strong><br>Validade: <strong>%s</strong></p><p>Condomínio Edifício Chateauneuf</p>`, html.EscapeString(name), html.EscapeString(apartment), html.EscapeString(username), html.EscapeString(password), expiresAt.In(time.Local).Format("02/01/2006"))
		if err := s.mailer.Send(ctx, to, "Acesso a internet - Apartamento "+apartment, body); err != nil {
			return "", err
		}
		return to, nil
	}
	return "", fmt.Errorf("apartamento %s nao encontrado: %w", apartment, ErrInternetCredentialEmailRequired)
}

func validEmail(value string) bool {
	value = strings.TrimSpace(value)
	address, err := mail.ParseAddress(value)
	return err == nil && strings.EqualFold(address.Address, value)
}
