package platform

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestFuelMonitoringWorkspaceContract(t *testing.T) {
	dsn := os.Getenv("PASTRACK_TEST_DSN")
	if dsn == "" {
		t.Skip("isolated database not configured")
	}
	if !strings.Contains(dsn, "127.0.0.1:33307)/pastrack_test_local") {
		t.Fatal("isolated database required")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db)
	request := func(method, target string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, "/api/v1/"+target, nil)
		if !s.fuelMonitoringOperations(w, r, strings.Split(target, "?")[0]) {
			t.Fatal("workspace route was not handled")
		}
		return w
	}
	w := request("GET", "fuel/monitoring/workspace?from=2026-11-01T00:00:00Z&to=2026-11-06T00:00:00Z")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var response struct {
		Data struct {
			DriverAvailable bool             `json:"driver_dimension_available"`
			Ignition        string           `json:"physical_ignition"`
			Fleet           []map[string]any `json:"fleet"`
			Rankings        []map[string]any `json:"rankings"`
			Heatmap         []map[string]any `json:"heatmap"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.DriverAvailable || response.Data.Ignition != "NOT_INSTALLED" || len(response.Data.Fleet) != 1 || response.Data.Rankings == nil || response.Data.Heatmap == nil {
		t.Fatalf("unexpected contract: %s", w.Body.String())
	}
	if w = request("GET", "fuel/monitoring/workspace?from=2026-01-01T00:00:00Z&to=2026-03-01T00:00:00Z"); w.Code != 400 {
		t.Fatal("period longer than 31 days accepted", w.Code)
	}
	if w = request("POST", "fuel/monitoring/workspace?from=2026-11-01T00:00:00Z&to=2026-11-02T00:00:00Z"); w.Code != 405 {
		t.Fatal("write method accepted", w.Code)
	}
}
