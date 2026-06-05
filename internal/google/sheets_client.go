package google

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"
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
	shoppingSheetName         = "Compras"
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

var shoppingHeaders = []interface{}{
	"ID Local",
	"Apartamento",
	"Entregador",
	"Documento",
	"Loja/Transportadora",
	"Mercadoria",
	"Observacoes",
	"Data Recebimento",
	"Hora Recebimento",
	"Data Retirada",
	"Hora Retirada",
	"Status",
	"Foto",
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

func (c *SheetsClient) ReadAccessLogs(ctx context.Context) ([]domain.AccessLog, error) {
	if c.spreadsheetID == "" {
		return nil, ErrSpreadsheetNotConfigured
	}
	if c.service == nil {
		return nil, ErrCredentialsNotConfigured
	}
	if err := c.ensureHeaders(ctx); err != nil {
		return nil, err
	}

	readRange := fmt.Sprintf("%s!A2:S", quoteSheetName(c.sheetName))
	response, err := c.service.Spreadsheets.Values.Get(c.spreadsheetID, readRange).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("read access logs from sheets: %w", err)
	}

	accessLogs := make([]domain.AccessLog, 0, len(response.Values))
	for _, row := range response.Values {
		accessLog, ok := accessLogFromSheetRow(row)
		if !ok {
			continue
		}
		accessLogs = append(accessLogs, accessLog)
	}

	return accessLogs, nil
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

func (c *SheetsClient) AppendShoppingDelivery(ctx context.Context, delivery domain.ShoppingDelivery) error {
	if c.spreadsheetID == "" {
		return ErrSpreadsheetNotConfigured
	}
	if c.service == nil {
		return ErrCredentialsNotConfigured
	}
	if err := c.ensureShoppingHeaders(ctx); err != nil {
		return err
	}

	row := shoppingRow(delivery, time.Now())
	rowIndex, err := c.findRowByLocalID(ctx, shoppingSheetName, delivery.ID)
	if err != nil {
		return err
	}
	if rowIndex > 0 {
		updateRange := fmt.Sprintf("%s!A%d:P%d", quoteSheetName(shoppingSheetName), rowIndex, rowIndex)
		_, err := c.service.Spreadsheets.Values.Update(c.spreadsheetID, updateRange, &sheets.ValueRange{
			Values: [][]interface{}{row},
		}).ValueInputOption("USER_ENTERED").Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("update shopping delivery in sheets: %w", err)
		}
		return nil
	}

	appendRange := fmt.Sprintf("%s!A:P", quoteSheetName(shoppingSheetName))
	_, err = c.service.Spreadsheets.Values.Append(c.spreadsheetID, appendRange, &sheets.ValueRange{
		Values: [][]interface{}{row},
	}).ValueInputOption("USER_ENTERED").InsertDataOption("INSERT_ROWS").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("append shopping delivery to sheets: %w", err)
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

func (c *SheetsClient) ensureShoppingHeaders(ctx context.Context) error {
	return c.ensureSheetHeaders(ctx, shoppingSheetName, shoppingHeaders, "A1:P1")
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
		sheetPhotoValue(accessLog.Photo),
	}
}

func accessLogFromSheetRow(row []interface{}) (domain.AccessLog, bool) {
	id, err := strconv.ParseInt(cell(row, 0), 10, 64)
	if err != nil || id <= 0 {
		return domain.AccessLog{}, false
	}

	entryAt, err := parseSheetDateTime(cell(row, 1), cell(row, 2))
	if err != nil {
		return domain.AccessLog{}, false
	}

	var exitAt *time.Time
	if parsedExitAt, err := parseSheetDateTime(cell(row, 3), cell(row, 4)); err == nil {
		exitAt = &parsedExitAt
	}

	visitStatus := domain.VisitStatus(cell(row, 15))
	if !visitStatus.IsValid() {
		if exitAt != nil {
			visitStatus = domain.VisitStatusFinished
		} else {
			visitStatus = domain.VisitStatusInProgress
		}
	}

	createdAt := parseSheetDateTimeOrDefault(cell(row, 16), "", entryAt)
	syncedAtValue := parseSheetDateTimeOrDefault(cell(row, 17), "", time.Now())
	updatedAt := syncedAtValue
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}

	return domain.AccessLog{
		ID:           id,
		ExternalID:   "",
		VisitorName:  cell(row, 5),
		Document:     cell(row, 6),
		Company:      cell(row, 7),
		Phone:        cell(row, 8),
		Unit:         cell(row, 9),
		ResidentName: cell(row, 10),
		ServiceType:  cell(row, 11),
		VehiclePlate: cell(row, 12),
		AuthorizedBy: cell(row, 13),
		Doorman:      cell(row, 14),
		Photo:        normalizeSheetPhoto(cell(row, 18)),
		EntryAt:      entryAt,
		ExitAt:       exitAt,
		VisitStatus:  visitStatus,
		SyncStatus:   domain.SyncStatusSynced,
		SyncError:    "",
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
		SyncedAt:     &syncedAtValue,
	}, true
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
		sheetPhotoValue(entry.Photo),
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
		sheetPhotoValue(service.Photo),
		formatDateTime(service.CreatedAt),
		formatDateTime(service.UpdatedAt),
		formatDateTime(syncedAt),
	}
}

func shoppingRow(delivery domain.ShoppingDelivery, syncedAt time.Time) []interface{} {
	return []interface{}{
		delivery.ID,
		delivery.Unit,
		delivery.CourierName,
		delivery.Document,
		delivery.Store,
		delivery.Product,
		delivery.Notes,
		formatDate(delivery.ReceivedAt),
		formatTime(delivery.ReceivedAt),
		formatOptionalDate(delivery.WithdrawnAt),
		formatOptionalTime(delivery.WithdrawnAt),
		string(delivery.Status),
		sheetPhotoValue(delivery.Photo),
		formatDateTime(delivery.CreatedAt),
		formatDateTime(delivery.UpdatedAt),
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

func cell(row []interface{}, index int) string {
	if index >= len(row) {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(row[index]))
}

func normalizeSheetPhoto(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "data:image/") || strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	return ""
}

func sheetPhotoValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 45000 {
		return value
	}
	if compressed := compressSheetPhoto(value); compressed != "" {
		return compressed
	}
	return fmt.Sprintf("Foto salva localmente no sistema (base64 com %d caracteres excede o limite do Google Sheets).", len(value))
}

func compressSheetPhoto(value string) string {
	header, encoded, found := strings.Cut(value, ",")
	if !found || !strings.HasPrefix(header, "data:image/") {
		return ""
	}

	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return ""
	}

	source, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return ""
	}

	for maxSide := 240; maxSide >= 80; maxSide -= 40 {
		resized := resizeImage(source, maxSide)
		for quality := 70; quality >= 35; quality -= 10 {
			var out bytes.Buffer
			if err := jpeg.Encode(&out, resized, &jpeg.Options{Quality: quality}); err != nil {
				continue
			}
			result := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(out.Bytes())
			if len(result) <= 45000 {
				return result
			}
		}
	}
	return ""
}

func resizeImage(source image.Image, maxSide int) image.Image {
	bounds := source.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return source
	}
	if width <= maxSide && height <= maxSide {
		return flattenImage(source, width, height)
	}

	newWidth := maxSide
	newHeight := maxSide
	if width >= height {
		newHeight = max(1, height*maxSide/width)
	} else {
		newWidth = max(1, width*maxSide/height)
	}

	target := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	for y := 0; y < newHeight; y++ {
		sourceY := bounds.Min.Y + y*height/newHeight
		for x := 0; x < newWidth; x++ {
			sourceX := bounds.Min.X + x*width/newWidth
			target.Set(x, y, source.At(sourceX, sourceY))
		}
	}
	return flattenImage(target, newWidth, newHeight)
}

func flattenImage(source image.Image, width int, height int) image.Image {
	target := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(target, target.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(target, target.Bounds(), source, source.Bounds().Min, draw.Over)
	return target
}

func parseSheetDateTimeOrDefault(dateValue string, timeValue string, fallback time.Time) time.Time {
	parsed, err := parseSheetDateTime(dateValue, timeValue)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseSheetDateTime(dateValue string, timeValue string) (time.Time, error) {
	dateValue = strings.TrimSpace(dateValue)
	timeValue = strings.TrimSpace(timeValue)
	if dateValue == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}

	if timeValue != "" {
		for _, layout := range []string{
			"02/01/2006 15:04:05",
			"02/01/2006 15:04",
			"2006-01-02 15:04:05",
			"2006-01-02 15:04",
		} {
			parsed, err := time.ParseInLocation(layout, dateValue+" "+timeValue, time.Local)
			if err == nil {
				return parsed, nil
			}
		}
	}

	for _, layout := range []string{
		time.RFC3339,
		"02/01/2006 15:04:05",
		"02/01/2006 15:04",
		"02/01/2006",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	} {
		parsed, err := time.ParseInLocation(layout, dateValue, time.Local)
		if err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid date time: %s %s", dateValue, timeValue)
}
