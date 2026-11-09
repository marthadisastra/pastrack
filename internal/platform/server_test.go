package platform

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUnauthenticatedAPI(t *testing.T) {
	s := New(nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/vehicles", nil))
	if w.Code != 401 || !strings.Contains(w.Body.String(), "authentication_required") {
		t.Fatal(w.Code, w.Body.String())
	}
}
func TestBoundedJSON(t *testing.T) {
	for _, value := range []string{`{"name":"a"} {"name":"b"}`, `{"unknown":1}`} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", strings.NewReader(value))
		var dst struct {
			Name string `json:"name"`
		}
		if body(w, r, &dst) == nil {
			t.Fatal("invalid JSON accepted")
		}
	}
}
func TestGeofenceValidation(t *testing.T) {
	if validateGeofence(map[string]any{"shape": "circle", "center_lat": -6.0, "center_lon": 106.0, "radius_m": 100.0}) != nil {
		t.Fatal("circle rejected")
	}
	if validateGeofence(map[string]any{"shape": "circle", "center_lat": -96.0, "center_lon": 106.0, "radius_m": 100.0}) == nil {
		t.Fatal("invalid accepted")
	}
}

func TestLiveUpstreamRejectsUnknownDestination(t *testing.T) {
	t.Setenv("PASTRACK_LIVE_UPSTREAM", "https://untrusted.invalid")
	s := New(nil)
	w := httptest.NewRecorder()
	if !s.proxyLive(w, httptest.NewRequest("POST", "/api/v1/vehicles", strings.NewReader(`{}`))) || w.Code != 503 {
		t.Fatal("unexpected live proxy destination accepted")
	}
	w = httptest.NewRecorder()
	if s.proxyLive(w, httptest.NewRequest("GET", "/api/v1/vehicles", nil)) {
		t.Fatal("normal reads should use live SQL directly")
	}
}
