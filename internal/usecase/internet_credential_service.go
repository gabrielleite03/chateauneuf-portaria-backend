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

var internetSSIDs = []string{"CHATEAUNEUF", "CHATEAUNEUF_5GHz"}

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
		body := internetCredentialEmailBody(name, apartment, username, password, expiresAt)
		if err := s.mailer.Send(ctx, to, "Acesso a internet - Apartamento "+apartment, body); err != nil {
			return "", err
		}
		return to, nil
	}
	return "", fmt.Errorf("apartamento %s nao encontrado: %w", apartment, ErrInternetCredentialEmailRequired)
}

func internetCredentialEmailBody(name, apartment, username, password string, expiresAt time.Time) string {
	ssidItems := ""
	for _, ssid := range internetSSIDs {
		ssidItems += fmt.Sprintf(`<tr><td style="padding:7px 0;color:#314750;font-size:14px;"><span style="display:inline-block;width:8px;height:8px;margin-right:9px;border-radius:50%%;background:#c9a45d;"></span><strong>%s</strong></td></tr>`, html.EscapeString(ssid))
	}

	return fmt.Sprintf(`<!doctype html><html lang="pt-BR"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Acesso à Internet</title></head>
<body style="margin:0;padding:0;background:#102d3a;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Arial,sans-serif;color:#172b35;">
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="background:#102d3a;"><tr><td align="center" style="padding:28px 12px;">
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="max-width:520px;background:#fff;border-radius:22px;overflow:hidden;box-shadow:0 20px 50px rgba(0,0,0,.25);">
<tr><td align="center" style="padding:30px 28px 26px;background:#17495b;color:#fff;"><div style="display:inline-block;width:62px;height:62px;line-height:62px;border:1px solid rgba(255,255,255,.45);border-radius:50%%;color:#c9a45d;font-family:Georgia,serif;font-size:24px;letter-spacing:-2px;">CH</div><h1 style="margin:15px 0 0;font-family:Georgia,'Times New Roman',serif;font-size:25px;font-weight:500;">Condomínio Edifício Chateauneuf</h1><p style="margin:8px 0 0;color:#c8d5da;font-size:12px;font-weight:700;letter-spacing:2px;text-transform:uppercase;">Acesso à Internet</p></td></tr>
<tr><td style="padding:30px 30px 32px;"><p style="margin:0 0 10px;font-size:16px;">Olá, <strong>%s</strong>.</p><p style="margin:0 0 24px;color:#677981;font-size:14px;line-height:1.6;">O acesso à internet do apartamento <strong>%s</strong> foi atualizado e terá validade de <strong>90 dias</strong>. Conecte-se a uma das redes abaixo e informe suas credenciais no portal.</p>
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="margin-bottom:18px;background:#f4f7f8;border:1px solid #dbe4e7;border-radius:12px;"><tr><td style="padding:18px 20px;"><p style="margin:0 0 9px;color:#677981;font-size:11px;font-weight:700;letter-spacing:1.5px;text-transform:uppercase;">Redes Wi-Fi disponíveis</p><table role="presentation" cellspacing="0" cellpadding="0" border="0">%s</table></td></tr></table>
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="background:#f4f7f8;border:1px solid #dbe4e7;border-radius:12px;"><tr><td style="padding:18px 20px;border-bottom:1px solid #dbe4e7;color:#677981;font-size:12px;">USUÁRIO<br><strong style="display:inline-block;margin-top:4px;color:#172b35;font-size:17px;">%s</strong></td></tr><tr><td style="padding:18px 20px;border-bottom:1px solid #dbe4e7;color:#677981;font-size:12px;">SENHA<br><strong style="display:inline-block;margin-top:4px;color:#172b35;font-size:17px;">%s</strong></td></tr><tr><td style="padding:18px 20px;color:#677981;font-size:12px;">VALIDADE<br><strong style="display:inline-block;margin-top:4px;color:#172b35;font-size:17px;">%s</strong></td></tr></table>
<p style="margin:22px 0 0;text-align:center;color:#819097;font-size:12px;line-height:1.5;">Credenciais pessoais. Não compartilhe seu acesso com terceiros.</p></td></tr></table></td></tr></table></body></html>`, html.EscapeString(name), html.EscapeString(apartment), ssidItems, html.EscapeString(username), html.EscapeString(password), expiresAt.In(time.Local).Format("02/01/2006"))
}

func validEmail(value string) bool {
	value = strings.TrimSpace(value)
	address, err := mail.ParseAddress(value)
	return err == nil && strings.EqualFold(address.Address, value)
}
