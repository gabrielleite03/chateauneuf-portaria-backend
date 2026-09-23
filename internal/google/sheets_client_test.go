package google

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"chateauneuf-portaria-backend/internal/domain"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

func TestFindAccessLogRowsIncludesAllDuplicates(t *testing.T) {
	for _, externalID := range []string{"visit-test", "not-found", ""} {
		t.Run(externalID, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				values := [][]interface{}{{"ID"}, {"37"}, {"37"}, {"99"}}
				if strings.HasSuffix(r.URL.Path, "T:T") {
					values = [][]interface{}{{"external_id"}, {"visit-test"}, {"visit-test"}, {"other-visit"}}
				}
				json.NewEncoder(w).Encode(&sheets.ValueRange{Values: values})
			}))
			defer server.Close()
			service, err := sheets.NewService(context.Background(), option.WithEndpoint(server.URL+"/"), option.WithoutAuthentication())
			if err != nil {
				t.Fatal(err)
			}
			client := &SheetsClient{service: service, spreadsheetID: "test", sheetName: "Entradas"}
			rows, err := client.findAccessLogRows(context.Background(), domain.AccessLog{ID: 37, ExternalID: externalID})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(rows, []int{2, 3}) {
				t.Fatalf("expected both duplicate rows, got %v", rows)
			}
		})
	}
}
