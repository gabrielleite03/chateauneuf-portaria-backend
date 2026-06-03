package google

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"chateauneuf-portaria-backend/internal/domain"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var (
	ErrSpreadsheetNotConfigured = errors.New("google spreadsheet nao configurado")
	ErrCredentialsNotConfigured = errors.New("credenciais google nao configuradas")
)

type SheetsClient struct {
	spreadsheetID   string
	sheetName       string
	credentialsFile string
	service         *sheets.Service
}

const (
	diaristaSheetName         = "Diaristas"
	keySheetName              = "Controle de Chaves"
	scheduledServiceSheetName = "Servicos Agendados"
)

var accessLogHeaders = []interface{}{
	"ID Local",
	"Data Entrada",
	"Hora Entrada",
	"Data Saida",
	"Hora Saida",
	"Nome Prestador",
	"Documento",
	"Empresa",
	"Telefone",
	"Unidade",
	"Morador Responsavel",
	"Tipo Servico",
	"Placa Veiculo",
	"Autorizado Por",
	"Porteiro",
	"Status",
	"Criado Em",
	"Sincronizado Em",
	"Foto",
}

var diaristaHeaders = []interface{}{
	"ID Local",
	"Data",
	"Nome",
	"RG",
	"Apto",
	"Autorizado Por",
	"Hora Entrada",
	"Hora Saida",
	"Porteiro",
	"Foto",
	"Status",
	"Criado Em",
	"Atualizado Em",
	"Sincronizado Em",
}

var scheduledServiceHeaders = []interface{}{
	"ID Local",
	"Data Agendada",
	"Nome Prestador",
	"Documento",
	"Empresa",
	"Unidade",
	"Autorizado Por",
	"Hora Check-In",
	"Observacoes",
	"Status",
	"Foto",
	"Criado Em",
	"Atualizado Em",
	"Sincronizado Em",
}

var keyHeaders = []interface{}{
	"ID Local",
	"Data",
	"Local",
	"Morador",
	"Apto",
	"Hora Retirada",
	"Hora Devolucao",
	"Porteiro",
	"Status",
	"Criado Em",
	"Atualizado Em",
	"Sincronizado Em",
}

func NewSheetsClient(ctx context.Context, spreadsheetID, sheetName, credentialsFile string) (*SheetsClient, error) {
	client := &SheetsClient{
		spreadsheetID:   spreadsheetID,
		sheetName:       sheetName,
		credentialsFile: credentialsFile,
	}

	if spreadsheetID == "" {
		return client, nil
	}
	if credentialsFile == "" {
		return client, nil
	}

	service, err := sheets.NewService(ctx, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, fmt.Errorf("create sheets service: %w", err)
	}
	client.service = service
	return client, nil
}

func (c *SheetsClient) Ping(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if c.spreadsheetID == "" {
		return ErrSpreadsheetNotConfigured
	}
	if c.service == nil {
		return ErrCredentialsNotConfigured
	}

	_, err := c.service.Spreadsheets.Get(c.spreadsheetID).Fields("spreadsheetId").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("google sheets ping: %w", err)
	}
	return nil
}

func (c *SheetsClient) AppendAccessLog(ctx context.Context, accessLog domain.AccessLog) error {
	if c.spreadsheetID == "" {
		return ErrSpreadsheetNotConfigured
	}
	if c.service == nil {
		return ErrCredentialsNotConfigured
	}
	if err := c.ensureHeaders(ctx); err != nil {
		return err
	}

	row := accessLogRow(accessLog, time.Now())
	rowIndex, err := c.findRowByLocalID(ctx, c.sheetName, strconv.FormatInt(accessLog.ID, 10))
	if err != nil {
		return err
	}
	if rowIndex > 0 {
		updateRange := fmt.Sprintf("%s!A%d:S%d", quoteSheetName(c.sheetName), rowIndex, rowIndex)
		_, err := c.service.Spreadsheets.Values.Update(c.spreadsheetID, updateRange, &sheets.ValueRange{
			Values: [][]interface{}{row},
		}).ValueInputOption("USER_ENTERED").Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("update access log in sheets: %w", err)
		}
		return nil
	}

	appendRange := fmt.Sprintf("%s!A:S", quoteSheetName(c.sheetName))
	_, err = c.service.Spreadsheets.Values.Append(c.spreadsheetID, appendRange, &sheets.ValueRange{
		Values: [][]interface{}{row},
	}).ValueInputOption("USER_ENTERED").InsertDataOption("INSERT_ROWS").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("append access log to sheets: %w", err)
	}

	return nil
}

func (c *SheetsClient) AppendDiaristaEntry(ctx context.Context, entry domain.DiaristaEntry) error {
	if c.spreadsheetID == "" {
		return ErrSpreadsheetNotConfigured
	}
	if c.service == nil {
		return ErrCredentialsNotConfigured
	}
	if err := c.ensureDiaristaHeaders(ctx); err != nil {
		return err
	}

	row := diaristaRow(entry, time.Now())
	rowIndex, err := c.findRowByLocalID(ctx, diaristaSheetName, entry.ID)
	if err != nil {
		return err
	}
	if rowIndex > 0 {
		updateRange := fmt.Sprintf("%s!A%d:N%d", quoteSheetName(diaristaSheetName), rowIndex, rowIndex)
		_, err := c.service.Spreadsheets.Values.Update(c.spreadsheetID, updateRange, &sheets.ValueRange{
			Values: [][]interface{}{row},
		}).ValueInputOption("USER_ENTERED").Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("update diarista entry in sheets: %w", err)
		}
		return nil
	}

	appendRange := fmt.Sprintf("%s!A:N", quoteSheetName(diaristaSheetName))
	_, err = c.service.Spreadsheets.Values.Append(c.spreadsheetID, appendRange, &sheets.ValueRange{
		Values: [][]interface{}{row},
	}).ValueInputOption("USER_ENTERED").InsertDataOption("INSERT_ROWS").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("append diarista entry to sheets: %w", err)
	}

	return nil
}

func (c *SheetsClient) AppendKeyRecord(ctx context.Context, key domain.KeyRecord) error {
	if c.spreadsheetID == "" {
		return ErrSpreadsheetNotConfigured
	}
	if c.service == nil {
		return ErrCredentialsNotConfigured
	}
	if err := c.ensureKeyHeaders(ctx); err != nil {
		return err
	}

	row := keyRow(key, time.Now())
	rowIndex, err := c.findRowByLocalID(ctx, keySheetName, key.ID)
	if err != nil {
		return err
	}
	if rowIndex > 0 {
		updateRange := fmt.Sprintf("%s!A%d:L%d", quoteSheetName(keySheetName), rowIndex, rowIndex)
		_, err := c.service.Spreadsheets.Values.Update(c.spreadsheetID, updateRange, &sheets.ValueRange{
			Values: [][]interface{}{row},
		}).ValueInputOption("USER_ENTERED").Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("update key record in sheets: %w", err)
		}
		return nil
	}

	appendRange := fmt.Sprintf("%s!A:L", quoteSheetName(keySheetName))
	_, err = c.service.Spreadsheets.Values.Append(c.spreadsheetID, appendRange, &sheets.ValueRange{
		Values: [][]interface{}{row},
	}).ValueInputOption("USER_ENTERED").InsertDataOption("INSERT_ROWS").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("append key record to sheets: %w", err)
	}

	return nil
}

func (c *SheetsClient) AppendScheduledService(ctx context.Context, service domain.ScheduledService) error {
	if c.spreadsheetID == "" {
		return ErrSpreadsheetNotConfigured
	}
	if c.service == nil {
		return ErrCredentialsNotConfigured
	}
	if err := c.ensureScheduledServiceHeaders(ctx); err != nil {
		return err
	}

	row := scheduledServiceRow(service, time.Now())
	rowIndex, err := c.findRowByLocalID(ctx, scheduledServiceSheetName, service.ID)
	if err != nil {
		return err
	}
	if rowIndex > 0 {
		updateRange := fmt.Sprintf("%s!A%d:N%d", quoteSheetName(scheduledServiceSheetName), rowIndex, rowIndex)
		_, err := c.service.Spreadsheets.Values.Update(c.spreadsheetID, updateRange, &sheets.ValueRange{
			Values: [][]interface{}{row},
		}).ValueInputOption("USER_ENTERED").Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("update scheduled service in sheets: %w", err)
		}
		return nil
	}

	appendRange := fmt.Sprintf("%s!A:N", quoteSheetName(scheduledServiceSheetName))
	_, err = c.service.Spreadsheets.Values.Append(c.spreadsheetID, appendRange, &sheets.ValueRange{
		Values: [][]interface{}{row},
	}).ValueInputOption("USER_ENTERED").InsertDataOption("INSERT_ROWS").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("append scheduled service to sheets: %w", err)
	}

	return nil
}

func (c *SheetsClient) ensureHeaders(ctx context.Context) error {
	return c.ensureSheetHeaders(ctx, c.sheetName, accessLogHeaders, "A1:S1")
}

func (c *SheetsClient) ensureDiaristaHeaders(ctx context.Context) error {
	return c.ensureSheetHeaders(ctx, diaristaSheetName, diaristaHeaders, "A1:N1")
}

func (c *SheetsClient) ensureKeyHeaders(ctx context.Context) error {
	return c.ensureSheetHeaders(ctx, keySheetName, keyHeaders, "A1:L1")
}

func (c *SheetsClient) ensureScheduledServiceHeaders(ctx context.Context) error {
	return c.ensureSheetHeaders(ctx, scheduledServiceSheetName, scheduledServiceHeaders, "A1:N1")
}

func (c *SheetsClient) ensureSheetHeaders(ctx context.Context, sheetName string, headers []interface{}, headerCells string) error {
	if err := c.ensureSheetExists(ctx, sheetName); err != nil {
		return err
	}

	headerRange := fmt.Sprintf("%s!%s", quoteSheetName(sheetName), headerCells)
	response, err := c.service.Spreadsheets.Values.Get(c.spreadsheetID, headerRange).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("read sheets headers: %w", err)
	}
	if len(response.Values) > 0 && headersMatch(response.Values[0], headers) {
		return nil
	}

	_, err = c.service.Spreadsheets.Values.Update(c.spreadsheetID, headerRange, &sheets.ValueRange{
		Values: [][]interface{}{headers},
	}).ValueInputOption("USER_ENTERED").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("write sheets headers: %w", err)
	}
	return nil
}

func (c *SheetsClient) ensureSheetExists(ctx context.Context, sheetName string) error {
	spreadsheet, err := c.service.Spreadsheets.Get(c.spreadsheetID).Fields("sheets.properties.title").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("read spreadsheet sheets: %w", err)
	}
	for _, sheet := range spreadsheet.Sheets {
		if sheet.Properties != nil && sheet.Properties.Title == sheetName {
			return nil
		}
	}

	_, err = c.service.Spreadsheets.BatchUpdate(c.spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				AddSheet: &sheets.AddSheetRequest{
					Properties: &sheets.SheetProperties{Title: sheetName},
				},
			},
		},
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("create sheet %s: %w", sheetName, err)
	}
	return nil
}

func headersMatch(current []interface{}, expected []interface{}) bool {
	if len(current) < len(expected) {
		return false
	}
	for index, header := range expected {
		if strings.TrimSpace(fmt.Sprint(current[index])) != fmt.Sprint(header) {
			return false
		}
	}
	return true
}

func (c *SheetsClient) findRowByLocalID(ctx context.Context, sheetName string, targetID string) (int, error) {
	readRange := fmt.Sprintf("%s!A:A", quoteSheetName(sheetName))
	response, err := c.service.Spreadsheets.Values.Get(c.spreadsheetID, readRange).Context(ctx).Do()
	if err != nil {
		return 0, fmt.Errorf("read sheets local ids: %w", err)
	}

	for index, row := range response.Values {
		if len(row) == 0 {
			continue
		}
		if strings.TrimSpace(fmt.Sprint(row[0])) == targetID {
			return index + 1, nil
		}
	}

	return 0, nil
}

func accessLogRow(accessLog domain.AccessLog, syncedAt time.Time) []interface{} {
	return []interface{}{
		accessLog.ID,
		formatDate(accessLog.EntryAt),
		formatTime(accessLog.EntryAt),
		formatOptionalDate(accessLog.ExitAt),
		formatOptionalTime(accessLog.ExitAt),
		accessLog.VisitorName,
		accessLog.Document,
		accessLog.Company,
		accessLog.Phone,
		accessLog.Unit,
		accessLog.ResidentName,
		accessLog.ServiceType,
		accessLog.VehiclePlate,
		accessLog.AuthorizedBy,
		accessLog.Doorman,
		string(accessLog.VisitStatus),
		formatDateTime(accessLog.CreatedAt),
		formatDateTime(syncedAt),
		accessLog.Photo,
	}
}

func diaristaRow(entry domain.DiaristaEntry, syncedAt time.Time) []interface{} {
	return []interface{}{
		entry.ID,
		entry.Date,
		entry.Name,
		entry.RG,
		entry.Unit,
		entry.AuthorizedBy,
		entry.EntryTime,
		entry.ExitTime,
		entry.Gatekeeper,
		entry.Photo,
		"Sincronizado",
		formatDateTime(entry.CreatedAt),
		formatDateTime(entry.UpdatedAt),
		formatDateTime(syncedAt),
	}
}

func keyRow(key domain.KeyRecord, syncedAt time.Time) []interface{} {
	return []interface{}{
		key.ID,
		key.Date,
		key.Local,
		key.ResidentName,
		key.Unit,
		key.PickupTime,
		key.ReturnTime,
		key.Gatekeeper,
		string(key.Status),
		formatDateTime(key.CreatedAt),
		formatDateTime(key.UpdatedAt),
		formatDateTime(syncedAt),
	}
}

func scheduledServiceRow(service domain.ScheduledService, syncedAt time.Time) []interface{} {
	return []interface{}{
		service.ID,
		service.Date,
		service.Name,
		service.Document,
		service.Company,
		service.Unit,
		service.AuthorizedBy,
		service.ArrivalTime,
		service.Notes,
		string(service.Status),
		service.Photo,
		formatDateTime(service.CreatedAt),
		formatDateTime(service.UpdatedAt),
		formatDateTime(syncedAt),
	}
}

func quoteSheetName(name string) string {
	return "'" + strings.ReplaceAll(name, "'", "''") + "'"
}

func formatOptionalDate(value *time.Time) string {
	if value == nil {
		return ""
	}
	return formatDate(*value)
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return formatTime(*value)
}

func formatDate(value time.Time) string {
	return value.Format("02/01/2006")
}

func formatTime(value time.Time) string {
	return value.Format("15:04:05")
}

func formatDateTime(value time.Time) string {
	return value.Format("02/01/2006 15:04:05")
}
