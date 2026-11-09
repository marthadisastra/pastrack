package controllers

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

func TestReportDateBoundaries(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/reports?type=positions&start=2026-10-28&end=2026-10-28&device=1", nil)
	f, e := parseReportFilter(r)
	if e != nil {
		t.Fatal(e)
	}
	if f.Start.UTC().Format(time.RFC3339) != "2026-10-27T17:00:00Z" || f.End.UTC().Format(time.RFC3339) != "2026-10-28T17:00:00Z" {
		t.Fatal(f)
	}
	for _, s := range []string{"start=2026-02-30&end=2026-03-01", "start=2026-10-29&end=2026-10-28", "start=2024-01-01&end=2026-01-01", "type=bad", "device=1%20OR%201=1", "vehicle=-1"} {
		if _, e := parseReportFilter(httptest.NewRequest("GET", "/api/reports?"+s, nil)); e == nil {
			t.Errorf("accepted invalid report filter %s", s)
		}
	}
}
func fixtureReport() ReportData {
	now := time.Date(2026, 10, 28, 10, 0, 0, 0, time.UTC)
	r := ReportData{Actor: "qa-admin", ID: "PTR-QA-REPORT", Title: "Laporan Posisi GPS", Generated: now, Filter: ReportFilter{Kind: "positions", Start: now.Add(-time.Hour), End: now.Add(time.Hour)}, Columns: []Column{{"plate_number", "Kendaraan", "text", 25}, {"imei", "IMEI", "text", 30}, {"gps_time", "Waktu GPS (WIB)", "date", 40}, {"latitude", "Latitude", "number", 30}, {"longitude", "Longitude", "number", 30}, {"speed_kmh", "Kecepatan (km/h)", "number", 33}}, Note: "QA fixture. Waktu WIB. Nilai kosong tidak disamakan dengan nol.", Summary: [][2]string{{"Jumlah baris", "120"}, {"IMEI unik", "1"}, {"Maksimum (km/h)", "119.50"}}}
	for i := 0; i < 120; i++ {
		r.Rows = append(r.Rows, map[string]any{"plate_number": "B 1307 KNQ", "imei": "001252080093813", "gps_time": now.Add(time.Duration(i) * time.Second), "latitude": "-6.1339000", "longitude": "106.8725160", "speed_kmh": fmt.Sprintf("%d.50", i)})
	}
	return r
}
func TestExcelTypingAndStructure(t *testing.T) {

	r := fixtureReport()
	r.Rows[0]["plate_number"] = "=HYPERLINK(\"https://example.invalid\")"
	b, e := makeXLSX(r)
	if e != nil {
		t.Fatal(e)
	}
	f, e := excelize.OpenReader(bytes.NewReader(b))
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	if len(f.GetSheetList()) != 3 {
		t.Fatal(f.GetSheetList())
	}
	if value, _ := f.GetCellValue("Detail", "B5"); value != "001252080093813" {
		t.Fatalf("IMEI lost: %q", value)
	}
	if formula, _ := f.GetCellFormula("Detail", "A5"); formula != "" {
		t.Fatal("formula injection")
	}
	if value, _ := f.GetCellValue("Detail", "C5"); !strings.Contains(value, "17:00:00") {
		t.Fatalf("incorrect WIB date %q", value)
	}
	tables, e := f.GetTables("Detail")
	if e != nil || len(tables) != 1 || tables[0].Range != "A4:F124" {
		t.Fatal(tables, e)
	}
	panes, e := f.GetPanes("Detail")
	if e != nil || !panes.Freeze || panes.YSplit != 4 {
		t.Fatal(panes, e)
	}
	if dir := os.Getenv("PASTRACK_QA_OUTPUT"); dir != "" {
		os.MkdirAll(dir, 0700)
		os.WriteFile(filepath.Join(dir, "qa-positions.xlsx"), b, 0600)
	}
}
func TestPDFMultiPageAndLongPayload(t *testing.T) {

	r := fixtureReport()
	b, e := makePDF(r)
	if e != nil || !bytes.HasPrefix(b, []byte("%PDF")) {
		t.Fatal(e)
	}
	if dir := os.Getenv("PASTRACK_QA_OUTPUT"); dir != "" {
		os.MkdirAll(dir, 0700)
		os.WriteFile(filepath.Join(dir, "qa-positions.pdf"), b, 0600)
	}
	r.Title = "Laporan Penerimaan Data"
	r.Filter.Kind = "messages"
	r.Columns = []Column{{"id", "ID", "number", 15}, {"imei", "IMEI", "text", 30}, {"raw_payload", "Raw Payload", "text", 100}}
	r.Rows = []map[string]any{{"id": 1, "imei": "001252080093813", "raw_payload": strings.Repeat("+RESP:GTTEST,123,", 1200)}}
	b, e = makePDF(r)
	if e != nil {
		t.Fatal(e)
	}
	if dir := os.Getenv("PASTRACK_QA_OUTPUT"); dir != "" {
		os.WriteFile(filepath.Join(dir, "qa-long-payload.pdf"), b, 0600)
	}
}
func TestMasterValidation(t *testing.T) {
	good := MasterInput{Plate: " b  1234 xx ", Name: "Armada", Type: "truck", Active: true}
	if e := validateMaster("vehicles", &good); e != nil || good.Plate != "B 1234 XX" {
		t.Fatal(good, e)
	}
	bad := MasterInput{IMEI: "1 OR 1=1", Model: "GV30CEU", Protocol: "queclink"}
	if validateMaster("devices", &bad) == nil {
		t.Fatal("invalid IMEI accepted")
	}
	bad.IMEI = "861252080093813"
	bad.Metadata = []byte(`[]`)
	if validateMaster("devices", &bad) == nil {
		t.Fatal("non-object metadata accepted")
	}
}
