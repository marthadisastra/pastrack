package platform

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestVehicleInventoryAtomicPhotoAndRevision(t *testing.T) {
	dsn := os.Getenv("PASTRACK_TEST_DSN")
	if dsn == "" {
		t.Skip("isolated database required")
	}
	if !strings.Contains(dsn, "127.0.0.1:33307)/pastrack_test_local") {
		t.Fatal("isolated database required")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	suffix := fmt.Sprint(time.Now().UnixNano())
	result, err := db.Exec("INSERT INTO users(username,full_name,password_hash,password_algo,auth_version,is_active,row_version) VALUES(?,'QA','unused','bcrypt',1,1,1)", "inventory-"+suffix)
	if err != nil {
		t.Fatal(err)
	}
	actor, _ := result.LastInsertId()
	s := New(db)
	user := &Identity{ID: actor, Role: "admin"}
	picture := image.NewRGBA(image.Rect(0, 0, 2, 2))
	picture.Set(0, 0, color.White)
	var encoded bytes.Buffer
	png.Encode(&encoded, picture)
	payload := map[string]any{"vehicle_name": "Inventory QA", "vehicle_code": "QA-" + suffix, "plate_number": "QA-" + suffix, "tank_capacity_l": 120, "photo_base64": base64.StdEncoding.EncodeToString(encoded.Bytes())}
	call := func(method, path string, want int) *httptest.ResponseRecorder {
		t.Helper()
		data, _ := json.Marshal(payload)
		w := httptest.NewRecorder()
		s.api(w, httptest.NewRequest(method, "/api/v1/"+path, bytes.NewReader(data)), user)
		if w.Code != want {
			t.Fatalf("%s status %d want %d: %s", path, w.Code, want, w.Body)
		}
		return w
	}
	w := call("POST", "vehicles/inventory", 201)
	var out struct {
		Data struct {
			ID int64 `json:"id"`
		}
	}
	json.Unmarshal(w.Body.Bytes(), &out)
	photo := call("GET", fmt.Sprintf("vehicles/%d/photo", out.Data.ID), 200)
	if !bytes.Equal(photo.Body.Bytes(), encoded.Bytes()) {
		t.Fatal("photo round trip changed")
	}
	payload["vehicle_name"] = "Changed"
	payload["row_version"] = 1
	delete(payload, "photo_base64")
	path := fmt.Sprintf("vehicles/%d/inventory", out.Data.ID)
	call("PUT", path, 200)
	call("PUT", path, 409)
	var capacity float64
	var version int
	if err = db.QueryRow("SELECT i.tank_capacity_l,v.row_version FROM vehicles v JOIN vehicle_inventory i ON i.vehicle_id=v.id WHERE v.id=?", out.Data.ID).Scan(&capacity, &version); err != nil || capacity != 120 || version != 2 {
		t.Fatalf("atomic inventory %v %v %v", capacity, version, err)
	}
	payload["row_version"] = 2
	payload["photo_base64"] = base64.StdEncoding.EncodeToString([]byte("not an image"))
	call("PUT", path, 400)
}
