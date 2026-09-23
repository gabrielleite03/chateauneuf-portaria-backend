package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const DeliveryPhotoTTL = 3 * time.Minute

var ErrDeliveryPhotoCodeInvalid = errors.New("codigo de foto invalido, expirado ou utilizado")

type DeliveryPhotoService struct{ db *sql.DB }

type DeliveryPhotoSession struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type DeliveryPhotoStatus struct {
	Status string `json:"status"`
	Photo  string `json:"photo,omitempty"`
}

func NewDeliveryPhotoService(db *sql.DB) *DeliveryPhotoService { return &DeliveryPhotoService{db: db} }

func deliveryPhotoHash(code string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(code))) }

func (s *DeliveryPhotoService) Create(ctx context.Context) (DeliveryPhotoSession, error) {
	var random [4]byte
	if _, err := rand.Read(random[:]); err != nil {
		return DeliveryPhotoSession{}, err
	}
	code := fmt.Sprintf("%06d", (uint32(random[0])<<24|uint32(random[1])<<16|uint32(random[2])<<8|uint32(random[3]))%1000000)
	now := time.Now().Round(0)
	expires := now.Add(DeliveryPhotoTTL)
	result, err := s.db.ExecContext(ctx, `INSERT INTO delivery_photo_sessions(code_hash,expires_at,created_at) VALUES(?,?,?)`, deliveryPhotoHash(code), expires, now)
	if err != nil {
		return DeliveryPhotoSession{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return DeliveryPhotoSession{}, err
	}
	return DeliveryPhotoSession{ID: strconv.FormatInt(id, 10), Code: code, ExpiresAt: expires}, nil
}

func (s *DeliveryPhotoService) Upload(ctx context.Context, code, photo string) error {
	code, photo = strings.TrimSpace(code), strings.TrimSpace(photo)
	if len(code) != 6 || !strings.HasPrefix(photo, "data:image/") || len(photo) > 2_500_000 {
		return ErrDeliveryPhotoCodeInvalid
	}
	comma := strings.IndexByte(photo, ',')
	if comma < 0 {
		return ErrDeliveryPhotoCodeInvalid
	}
	if _, err := base64.StdEncoding.DecodeString(photo[comma+1:]); err != nil {
		return ErrDeliveryPhotoCodeInvalid
	}
	now := time.Now().Round(0)
	result, err := s.db.ExecContext(ctx, `UPDATE delivery_photo_sessions SET photo_data=?,uploaded_at=? WHERE code_hash=? AND expires_at>? AND uploaded_at IS NULL AND consumed_at IS NULL`, photo, now, deliveryPhotoHash(code), now)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows != 1 {
		return ErrDeliveryPhotoCodeInvalid
	}
	return nil
}

func (s *DeliveryPhotoService) Status(ctx context.Context, sessionID string) (DeliveryPhotoStatus, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(sessionID), 10, 64)
	if err != nil {
		return DeliveryPhotoStatus{}, ErrDeliveryPhotoCodeInvalid
	}
	var photo string
	var expires time.Time
	var consumed sql.NullTime
	if err = s.db.QueryRowContext(ctx, `SELECT photo_data,expires_at,consumed_at FROM delivery_photo_sessions WHERE id=?`, id).Scan(&photo, &expires, &consumed); err != nil {
		return DeliveryPhotoStatus{}, ErrDeliveryPhotoCodeInvalid
	}
	if consumed.Valid {
		return DeliveryPhotoStatus{Status: "consumed"}, nil
	}
	if !expires.After(time.Now()) {
		return DeliveryPhotoStatus{Status: "expired"}, nil
	}
	if photo == "" {
		return DeliveryPhotoStatus{Status: "waiting"}, nil
	}
	return DeliveryPhotoStatus{Status: "ready", Photo: photo}, nil
}
