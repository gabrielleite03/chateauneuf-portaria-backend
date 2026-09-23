package usecase

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"image"
	_ "image/png"
	"strconv"
	"strings"
	"time"

	"chateauneuf-portaria-backend/internal/domain"
	"github.com/phpdave11/gofpdf"
)

const SignatureCodeTTL = 3 * time.Minute

var ErrSignatureCodeInvalid = errors.New("codigo invalido ou expirado")

type PDFEmailSender interface {
	Send(context.Context, string, string, string) error
	SendAttachment(context.Context, string, string, string, string, []byte) error
	SendInlineImage(context.Context, string, string, string, string, []byte) error
}
type ReservationSignatureService struct {
	db        *sql.DB
	residents *ResidentService
	mailer    PDFEmailSender
}
type SignatureForm struct {
	Area, ResidentName, Unit, ReservationDate, StartTime, EndTime string
	Fee                                                           string    `json:"fee"`
	ExpiresAt                                                     time.Time `json:"expiresAt"`
	SignedAt                                                      time.Time `json:"-"`
}
type SignatureCode struct {
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func NewReservationSignatureService(db *sql.DB, residents *ResidentService, mailer PDFEmailSender) *ReservationSignatureService {
	return &ReservationSignatureService{db: db, residents: residents, mailer: mailer}
}

func codeHash(code string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(code))) }
func (s *ReservationSignatureService) CreateCode(ctx context.Context, reservationID string) (SignatureCode, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(reservationID), 10, 64)
	if err != nil {
		return SignatureCode{}, domain.ErrInvalidInput
	}
	var status, unit string
	if err = s.db.QueryRowContext(ctx, `SELECT status,unit FROM common_area_reservations WHERE id=?`, id).Scan(&status, &unit); err != nil {
		return SignatureCode{}, domain.ErrNotFound
	}
	if status != string(domain.ReservationStatusBooked) {
		return SignatureCode{}, domain.ErrInvalidInput
	}
	if _, err = s.residentEmail(ctx, unit); err != nil {
		return SignatureCode{}, err
	}
	var random [4]byte
	if _, err = rand.Read(random[:]); err != nil {
		return SignatureCode{}, err
	}
	code := fmt.Sprintf("%06d", (uint32(random[0])<<24|uint32(random[1])<<16|uint32(random[2])<<8|uint32(random[3]))%1000000)
	now, expires := time.Now().Round(0), time.Now().Add(SignatureCodeTTL).Round(0)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return SignatureCode{}, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `UPDATE reservation_signatures SET expires_at=? WHERE reservation_id=? AND consumed_at IS NULL`, now, id)
	if err != nil {
		return SignatureCode{}, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO reservation_signatures(reservation_id,code_hash,expires_at,created_at) VALUES(?,?,?,?)`, id, codeHash(code), expires, now)
	if err != nil {
		return SignatureCode{}, err
	}
	if err = tx.Commit(); err != nil {
		return SignatureCode{}, err
	}
	return SignatureCode{Code: code, ExpiresAt: expires}, nil
}

func (s *ReservationSignatureService) Lookup(ctx context.Context, code string) (SignatureForm, error) {
	return s.lookup(ctx, code, time.Now())
}
func (s *ReservationSignatureService) lookup(ctx context.Context, code string, now time.Time) (SignatureForm, error) {
	var f SignatureForm
	err := s.db.QueryRowContext(ctx, `SELECT r.area,r.resident_name,r.unit,r.reservation_date,r.start_time,r.end_time,s.expires_at FROM reservation_signatures s JOIN common_area_reservations r ON r.id=s.reservation_id WHERE s.code_hash=? AND s.consumed_at IS NULL AND s.expires_at>?`, codeHash(strings.TrimSpace(code)), now).Scan(&f.Area, &f.ResidentName, &f.Unit, &f.ReservationDate, &f.StartTime, &f.EndTime, &f.ExpiresAt)
	if err != nil {
		return SignatureForm{}, ErrSignatureCodeInvalid
	}
	if f.Area == "Salão de festas" || f.Area == "Salao de festas" {
		f.Fee = "R$ 200,00"
	} else {
		f.Fee = "R$ 80,00"
	}
	return f, nil
}

func (s *ReservationSignatureService) Confirm(ctx context.Context, code, dataURL string) error {
	f, err := s.lookup(ctx, code, time.Now())
	if err != nil {
		return err
	}
	comma := strings.IndexByte(dataURL, ',')
	if comma < 0 {
		return domain.ErrInvalidInput
	}
	png, err := base64.StdEncoding.DecodeString(dataURL[comma+1:])
	if err != nil || len(png) < 100 || len(png) > 2_000_000 {
		return domain.ErrInvalidInput
	}
	now := time.Now().Round(0)
	f.SignedAt = now
	pdf, err := reservationPDF(f, png)
	if err != nil {
		return err
	}
	email, err := s.residentEmail(ctx, f.Unit)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE reservation_signatures SET consumed_at=?,signature_png=?,pdf_data=?,recipient_email=?,email_status='pending' WHERE code_hash=? AND consumed_at IS NULL AND expires_at>?`, now, png, pdf, email, codeHash(strings.TrimSpace(code)), now)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return ErrSignatureCodeInvalid
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	err = s.mailer.SendAttachment(ctx, email, "Termo de reserva assinado - "+f.Area, "<p>Olá. Segue anexo o termo de reserva assinado digitalmente.</p>", "termo-reserva.pdf", pdf)
	status, detail := "sent", ""
	if err != nil {
		status, detail = "failed", err.Error()
	}
	_, _ = s.db.ExecContext(context.Background(), `UPDATE reservation_signatures SET email_status=?,email_error=? WHERE code_hash=?`, status, detail, codeHash(strings.TrimSpace(code)))
	return nil
}

func (s *ReservationSignatureService) residentEmail(ctx context.Context, unit string) (string, error) {
	rows, err := s.residents.List(ctx)
	if err != nil {
		return "", err
	}
	for _, r := range rows {
		if normalizeApartmentUnit(r.Unit) == normalizeApartmentUnit(unit) {
			if validEmail(r.TenantEmail) {
				return strings.TrimSpace(r.TenantEmail), nil
			}
			if validEmail(r.Email) {
				return strings.TrimSpace(r.Email), nil
			}
		}
	}
	return "", fmt.Errorf("cadastre um e-mail valido para o inquilino ou morador do apartamento %s: %w", unit, ErrInternetCredentialEmailRequired)
}

func normalizeApartmentUnit(unit string) string {
	value := strings.ToLower(strings.TrimSpace(unit))
	for _, prefix := range []string{"apartamento", "apto.", "apto"} {
		if strings.HasPrefix(value, prefix) {
			value = strings.TrimSpace(strings.TrimPrefix(value, prefix))
			break
		}
	}
	value = strings.Trim(value, " -–—:nº°.")
	if number, err := strconv.Atoi(value); err == nil {
		return strconv.Itoa(number)
	}
	return strings.Join(strings.Fields(value), "")
}

func (s *ReservationSignatureService) NotifyCancellation(ctx context.Context, r domain.CommonAreaReservation) error {
	email, err := s.residentEmail(ctx, r.Unit)
	if err != nil {
		return err
	}
	date := r.ReservationDate
	if parsed, parseErr := time.Parse("2006-01-02", r.ReservationDate); parseErr == nil {
		date = parsed.Format("02/01/2006")
	}
	body := fmt.Sprintf(`<!doctype html><html><body style="margin:0;background:#102d3a;font-family:Arial,sans-serif;color:#172b35"><table width="100%%" role="presentation"><tr><td align="center" style="padding:28px 12px"><table width="100%%" style="max-width:520px;background:#fff;border-radius:22px;overflow:hidden"><tr><td align="center" style="padding:28px;background:#17495b"><h1 style="font:24px Georgia;margin:0;color:#c9a45d">Condomínio Edifício Chateauneuf</h1><p style="letter-spacing:2px;font-size:12px;color:#fff">CANCELAMENTO DE RESERVA</p></td></tr><tr><td style="padding:30px"><p>Olá, <strong>%s</strong>.</p><p style="color:#677981;line-height:1.6">A reserva abaixo foi cancelada pela portaria.</p><div style="padding:18px;background:#f4f7f8;border:1px solid #dbe4e7;border-radius:12px"><strong>%s</strong><br>Apartamento %s<br>Data: %s<br>Horário: %s–%s</div><p style="margin-top:22px;color:#a8554b;font-weight:bold">Reserva cancelada</p></td></tr></table></td></tr></table></body></html>`, html.EscapeString(r.ResidentName), html.EscapeString(r.Area), html.EscapeString(r.Unit), html.EscapeString(date), html.EscapeString(r.StartTime), html.EscapeString(r.EndTime))
	return s.mailer.Send(ctx, email, "Reserva cancelada - "+r.Area, body)
}

func (s *ReservationSignatureService) NotifyDeliveryArrival(ctx context.Context, delivery domain.ShoppingDelivery, photoDataURL string) error {
	email, err := s.residentEmail(ctx, delivery.Unit)
	if errors.Is(err, ErrInternetCredentialEmailRequired) {
		return nil
	}
	if err != nil {
		return err
	}
	photoDataURL = strings.TrimSpace(photoDataURL)
	comma := strings.IndexByte(photoDataURL, ',')
	if comma < 0 || !strings.HasPrefix(photoDataURL, "data:image/") {
		return nil
	}
	contentType := strings.TrimPrefix(strings.Split(photoDataURL[:comma], ";")[0], "data:")
	photo, err := base64.StdEncoding.DecodeString(photoDataURL[comma+1:])
	if err != nil || len(photo) == 0 {
		return nil
	}
	body := fmt.Sprintf(`<!doctype html><html><body style="margin:0;background:#102d3a;font-family:Arial,sans-serif;color:#172b35"><table width="100%%" role="presentation"><tr><td align="center" style="padding:28px 12px"><table width="100%%" style="max-width:520px;background:#fff;border-radius:22px;overflow:hidden"><tr><td align="center" style="padding:28px;background:#17495b"><h1 style="font:24px Georgia;margin:0;color:#c9a45d">Condomínio Edifício Chateauneuf</h1><p style="letter-spacing:2px;font-size:12px;color:#fff">ENTREGA RECEBIDA</p></td></tr><tr><td style="padding:30px"><p>Olá, <strong>%s</strong>.</p><p style="color:#677981;line-height:1.6">Uma entrega destinada ao apartamento <strong>%s</strong> chegou à portaria.</p><div style="padding:18px;background:#f4f7f8;border:1px solid #dbe4e7;border-radius:12px"><strong>%s</strong><br>Origem: %s<br>Recebida em: %s</div><p style="margin:20px 0 8px;font-weight:bold">Foto da entrega:</p><img src="cid:delivery-photo" alt="Foto da entrega" style="display:block;max-width:100%%;height:auto;border-radius:12px;border:1px solid #dbe4e7"></td></tr></table></td></tr></table></body></html>`, html.EscapeString(delivery.Recipient), html.EscapeString(delivery.Unit), html.EscapeString(delivery.Product), html.EscapeString(delivery.Store), delivery.ReceivedAt.In(time.Local).Format("02/01/2006 15:04"))
	return s.mailer.SendInlineImage(ctx, email, "Entrega recebida - Apartamento "+delivery.Unit, body, contentType, photo)
}

func (s *ReservationSignatureService) Document(ctx context.Context, reservationID string) ([]byte, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(reservationID), 10, 64)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}
	var f SignatureForm
	var signature []byte
	if err = s.db.QueryRowContext(ctx, `
		SELECT r.area,r.resident_name,r.unit,r.reservation_date,r.start_time,r.end_time,
			s.signature_png,s.consumed_at
		FROM reservation_signatures s
		JOIN common_area_reservations r ON r.id=s.reservation_id
		WHERE s.reservation_id=? AND s.consumed_at IS NOT NULL
		ORDER BY s.consumed_at DESC LIMIT 1
	`, id).Scan(&f.Area, &f.ResidentName, &f.Unit, &f.ReservationDate, &f.StartTime, &f.EndTime, &signature, &f.SignedAt); err != nil || len(signature) == 0 {
		return nil, domain.ErrNotFound
	}
	if f.Area == "Salão de festas" || f.Area == "Salao de festas" {
		f.Fee = "R$ 200,00"
	} else {
		f.Fee = "R$ 80,00"
	}
	pdf, err := reservationPDF(f, signature)
	if err != nil {
		return nil, err
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE reservation_signatures SET pdf_data=? WHERE reservation_id=? AND consumed_at=?`, pdf, id, f.SignedAt)
	return pdf, nil
}

func reservationPDF(f SignatureForm, signature []byte) ([]byte, error) {
	p := gofpdf.New("P", "mm", "A4", "")
	tr := p.UnicodeTranslatorFromDescriptor("")
	p.SetMargins(18, 18, 18)
	p.AddPage()
	p.SetFillColor(23, 73, 91)
	p.Rect(0, 0, 210, 42, "F")
	p.SetY(12)
	p.SetTextColor(201, 164, 93)
	p.SetFont("Helvetica", "B", 15)
	p.CellFormat(0, 8, tr("CONDOMÍNIO EDIFÍCIO CHATEAUNEUF"), "", 1, "C", false, 0, "")
	p.SetTextColor(255, 255, 255)
	p.SetFont("Helvetica", "B", 10)
	p.CellFormat(0, 6, tr("TERMO DE REQUISIÇÃO E RESPONSABILIDADE"), "", 1, "C", false, 0, "")
	p.SetY(49)
	p.SetTextColor(23, 43, 53)
	p.SetFillColor(244, 247, 248)
	p.SetDrawColor(219, 228, 231)
	p.SetFont("Helvetica", "B", 11)
	p.CellFormat(0, 9, tr(strings.ToUpper(f.Area)), "1", 1, "C", true, 0, "")
	p.SetFont("Helvetica", "", 9)
	p.CellFormat(87, 8, tr("Morador: "+f.ResidentName), "LB", 0, "L", true, 0, "")
	p.CellFormat(45, 8, tr("Apto: "+f.Unit), "B", 0, "L", true, 0, "")
	p.CellFormat(42, 8, tr("Taxa: "+f.Fee), "BR", 1, "L", true, 0, "")
	eventDate := f.ReservationDate
	if parsed, err := time.Parse("2006-01-02", f.ReservationDate); err == nil {
		eventDate = parsed.Format("02/01/2006")
	}
	p.CellFormat(87, 8, tr("Data do evento: "+eventDate), "LB", 0, "L", true, 0, "")
	p.CellFormat(87, 8, tr("Horário: "+f.StartTime+" às "+f.EndTime), "BR", 1, "L", true, 0, "")
	p.Ln(6)
	text := "Na qualidade de morador(a) do Condomínio Chateauneuf, na Rua Vigário João Álvares, 157 – São Paulo, sirvo-me da presente para requisitar para meu uso privativo o espaço acima, estando ciente do Regulamento Interno e dos procedimentos para seu uso. Declaro ter pleno conhecimento das regras, responsabilizando-me por danos causados nas dependências por mim ou por meus convidados. Não será permitida aparelhagem de som ou instrumentos musicais fora da área reservada; excessos de barulho poderão causar o encerramento da festa. Os convidados deverão permanecer exclusivamente na área reservada, sem acesso às demais áreas exclusivas dos condôminos, principalmente piscinas. Declaro receber em ordem o espaço, móveis e objetos instalados. Comprometo-me a cumprir os horários previstos no regulamento interno."
	if f.Area == "Salão de festas" || f.Area == "Salao de festas" {
		text += " Autorizo a cobrança da taxa de utilização de R$ 200,00 juntamente com a taxa condominial do mês subsequente."
	} else {
		text += " Os equipamentos e utensílios deverão ser devolvidos limpos após o uso. Autorizo a cobrança da taxa de utilização da churrasqueira de R$ 80,00 juntamente com a taxa condominial do mês subsequente."
	}
	text = strings.ReplaceAll(text, ". ", ".\n\n")
	p.SetFont("Helvetica", "", 9.5)
	p.MultiCell(0, 5, tr(text), "", "J", false)
	p.Ln(6)
	p.SetDrawColor(201, 164, 93)
	p.Line(18, p.GetY(), 192, p.GetY())
	p.Ln(5)
	p.SetFont("Helvetica", "B", 10)
	p.SetTextColor(23, 73, 91)
	p.Cell(0, 6, tr("Assinatura do condômino requisitante:"))
	p.Ln(7)
	imageConfig, _, err := image.DecodeConfig(bytes.NewReader(signature))
	if err != nil || imageConfig.Width <= 0 || imageConfig.Height <= 0 {
		return nil, fmt.Errorf("invalid signature image")
	}
	const maxSignatureWidth = 100.0
	const maxSignatureHeight = 24.0
	imageWidth := maxSignatureWidth
	imageHeight := imageWidth * float64(imageConfig.Height) / float64(imageConfig.Width)
	if imageHeight > maxSignatureHeight {
		imageHeight = maxSignatureHeight
		imageWidth = imageHeight * float64(imageConfig.Width) / float64(imageConfig.Height)
	}
	signatureY := p.GetY()
	if signatureY+imageHeight+12 > 280 {
		p.AddPage()
		signatureY = 22
	}
	signatureX := (210.0 - imageWidth) / 2
	opts := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
	p.RegisterImageOptionsReader("signature", opts, bytes.NewReader(signature))
	p.ImageOptions("signature", signatureX, signatureY, imageWidth, imageHeight, false, opts, 0, "")
	p.SetY(signatureY + maxSignatureHeight + 4)
	p.SetTextColor(103, 121, 129)
	p.SetFont("Helvetica", "", 8)
	signedAt := f.SignedAt
	if signedAt.IsZero() {
		signedAt = time.Now()
	}
	p.Cell(0, 5, tr("Assinado eletronicamente em "+signedAt.Format("02/01/2006 15:04:05")))
	var out bytes.Buffer
	if err := p.Output(&out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func SignatureEmailBody(name string) string { return "<p>" + html.EscapeString(name) + "</p>" }
