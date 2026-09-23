package google

import (
	"chateauneuf-portaria-backend/internal/domain"
	"context"
	"encoding/json"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReservationSheetAppendAndUpdateDuplicates(t *testing.T) {
	appends, updates := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case !strings.Contains(r.URL.Path, "/values/"):
			json.NewEncoder(w).Encode(&sheets.Spreadsheet{Sheets: []*sheets.Sheet{{Properties: &sheets.SheetProperties{Title: reservationSheetName}}}})
		case r.Method != http.MethodGet:
			var body sheets.ValueRange
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if r.URL.Query().Get("valueInputOption") != "RAW" {
				t.Error("reservation values must be literal")
			}
			if len(body.Values) != 1 || len(body.Values[0]) != 13 || body.Values[0][0] != "26" {
				t.Errorf("unexpected row: %+v", body.Values)
			}
			if r.Method == http.MethodPost {
				appends++
			} else {
				updates++
			}
			w.Write([]byte(`{}`))
		case strings.HasSuffix(r.URL.Path, "A1:M1"):
			json.NewEncoder(w).Encode(&sheets.ValueRange{Values: [][]interface{}{reservationHeaders}})
		default:
			values := [][]interface{}{{"ID Local"}}
			if appends > 0 {
				values = append(values, []interface{}{"26"}, []interface{}{"26"})
			}
			json.NewEncoder(w).Encode(&sheets.ValueRange{Values: values})
		}
	}))
	defer server.Close()
	svc, err := sheets.NewService(context.Background(), option.WithEndpoint(server.URL+"/"), option.WithoutAuthentication())
	if err != nil {
		t.Fatal(err)
	}
	client := &SheetsClient{spreadsheetID: "test", service: svc}
	r := domain.CommonAreaReservation{ID: "26", Area: "Churrasqueira", Unit: "101", Status: domain.ReservationStatusBooked}
	if err := client.AppendReservation(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	r.Status = domain.ReservationStatusCanceled
	if err := client.AppendReservation(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if appends != 1 || updates != 2 {
		t.Fatalf("appends=%d updates=%d", appends, updates)
	}
}
