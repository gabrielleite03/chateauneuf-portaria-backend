package google

import (
	"chateauneuf-portaria-backend/internal/domain"
	"context"
	"fmt"
	"google.golang.org/api/sheets/v4"
	"time"
)

const reservationSheetName = "Reservas"

var reservationHeaders = []interface{}{
	"ID Local", "Area", "Apartamento", "Morador", "Data Reserva", "Inicio", "Fim",
	"Convidados", "Observacoes", "Status", "Criado Em", "Atualizado Em", "Sincronizado Em",
}

func (c *SheetsClient) AppendReservation(ctx context.Context, reservation domain.CommonAreaReservation) error {
	if c.spreadsheetID == "" {
		return ErrSpreadsheetNotConfigured
	}
	if c.service == nil {
		return ErrCredentialsNotConfigured
	}
	if err := c.ensureSheetHeaders(ctx, reservationSheetName, reservationHeaders, "A1:M1"); err != nil {
		return err
	}
	row := []interface{}{reservation.ID, reservation.Area, reservation.Unit, reservation.ResidentName,
		reservation.ReservationDate, reservation.StartTime, reservation.EndTime, reservation.Guests,
		reservation.Notes, string(reservation.Status), formatDateTime(reservation.CreatedAt),
		formatDateTime(reservation.UpdatedAt), formatDateTime(time.Now())}
	indexes, err := c.findRowsByColumnValue(ctx, reservationSheetName, "A", reservation.ID)
	if err != nil {
		return err
	}
	values := &sheets.ValueRange{Values: [][]interface{}{row}}
	if len(indexes) == 0 {
		_, err = c.service.Spreadsheets.Values.Append(c.spreadsheetID, quoteSheetName(reservationSheetName)+"!A:M", values).
			ValueInputOption("RAW").InsertDataOption("INSERT_ROWS").Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("append reservation to sheets: %w", err)
		}
		return nil
	}
	for _, index := range indexes {
		target := fmt.Sprintf("%s!A%d:M%d", quoteSheetName(reservationSheetName), index, index)
		if _, err = c.service.Spreadsheets.Values.Update(c.spreadsheetID, target, values).ValueInputOption("RAW").Context(ctx).Do(); err != nil {
			return fmt.Errorf("update reservation in sheets: %w", err)
		}
	}
	return nil
}
