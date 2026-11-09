package controllers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/server/web"
	"github.com/go-pdf/fpdf"
	"github.com/xuri/excelize/v2"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

var wib = time.FixedZone("WIB", 7*3600)
var exportSlots = make(chan struct{}, 2)

type Report struct{ web.Controller }
type AuditView struct{ web.Controller }
type ReportFilter struct {
	Kind, Status        string
	Start, End          time.Time
	DeviceID, VehicleID int
}
type Column struct {
	Key, Label, Kind string
	Width            float64
}
type ReportData struct {
	Actor     string
	ID, Title string
	Generated time.Time
	Filter    ReportFilter
	Columns   []Column
	Rows      []map[string]any
	Summary   [][2]string
	Note      string
}

func parseReportFilter(r *http.Request) (f ReportFilter, e error) {
	q := r.URL.Query()
	f.Kind = q.Get("type")
	if f.Kind == "" {
		f.Kind = "positions"
	}
	switch f.Kind {
	case "positions", "events", "messages", "vehicles", "devices":
	default:
		return f, errors.New("Jenis laporan tidak valid.")
	}
	now := time.Now().In(wib)
	start := q.Get("start")
	end := q.Get("end")
	if start == "" {
		start = now.AddDate(0, 0, -6).Format("2006-01-02")
	}
	if end == "" {
		end = now.Format("2006-01-02")
	}
	f.Start, e = time.ParseInLocation("2006-01-02", start, wib)
	if e != nil {
		return f, errors.New("Tanggal awal tidak valid.")
	}
	f.End, e = time.ParseInLocation("2006-01-02", end, wib)
	if e != nil {
		return f, errors.New("Tanggal akhir tidak valid.")
	}
	f.End = f.End.AddDate(0, 0, 1)
	if !f.End.After(f.Start) || f.End.Sub(f.Start) > 366*24*time.Hour {
		return f, errors.New("Periode maksimal 366 hari dan tanggal akhir tidak boleh sebelum tanggal awal.")
	}
	for key, dest := range map[string]*int{"device": &f.DeviceID, "vehicle": &f.VehicleID} {
		s := q.Get(key)
		if s != "" {
			v, err := strconv.Atoi(s)
			if err != nil || v < 0 || v > 2147483647 {
				return f, errors.New("Filter master tidak valid.")
			}
			*dest = v
		}
	}
	f.Status = q.Get("status")
	if f.Kind == "messages" {
		if f.Status != "" && f.Status != "success" && f.Status != "failed" && f.Status != "partial" && f.Status != "pending" {
			return f, errors.New("Status parsing tidak valid.")
		}
	} else if f.Status != "" && f.Status != "active" && f.Status != "inactive" {
		return f, errors.New("Status master tidak valid.")
	}
	return
}
func queryTx(ctx context.Context, tx *sql.Tx, q string, args ...any) ([]map[string]any, error) {
	rows, e := tx.QueryContext(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	cols, e := rows.Columns()
	if e != nil {
		return nil, e
	}
	out := []map[string]any{}
	for rows.Next() {
		v := make([]any, len(cols))
		p := make([]any, len(cols))
		for i := range p {
			p[i] = &v[i]
		}
		if e = rows.Scan(p...); e != nil {
			return nil, e
		}
		row := map[string]any{}
		for i, k := range cols {
			switch x := v[i].(type) {
			case []byte:
				row[k] = string(x)
			default:
				row[k] = x
			}
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
func buildReport(ctx context.Context, f ReportFilter, limit int) (ReportData, error) {
	report := ReportData{Actor: actorName(ctx), ID: "PTR-" + time.Now().In(wib).Format("20060102") + "-" + strings.ToUpper(requestID()[:12]), Generated: time.Now().UTC(), Filter: f, Note: "Sumber: database Pastrack. Waktu WIB (UTC+7). Relasi kendaraan dan perangkat mengikuti master pada waktu laporan dibuat. Nilai kosong berarti belum tersedia. Tidak ada estimasi liter, jarak tempuh, atau interpretasi kode event yang belum diverifikasi."}
	tx, e := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if e != nil {
		return report, e
	}
	defer tx.Rollback()
	var q string
	var args []any
	switch f.Kind {
	case "positions":
		report.Title = "Laporan Posisi GPS"
		report.Columns = []Column{{"plate_number", "Kendaraan", "text", 24}, {"imei", "IMEI", "text", 27}, {"gps_time", "Waktu GPS (WIB)", "date", 34}, {"received_at", "Diterima (WIB)", "date", 34}, {"latitude", "Latitude", "number", 25}, {"longitude", "Longitude", "number", 25}, {"speed_kmh", "Kecepatan (km/h)", "number", 30}, {"heading_deg", "Arah (derajat)", "number", 27}, {"altitude_m", "Altitude (m)", "number", 27}}
		q = `SELECT p.id,d.imei,v.plate_number,p.gps_time,COALESCE(r.received_at,p.created_at) AS received_at,p.latitude,p.longitude,p.speed_kmh,p.heading_deg,p.altitude_m FROM gps_positions p JOIN gps_devices d ON d.id=p.device_id LEFT JOIN vehicles v ON v.id=d.vehicle_id LEFT JOIN device_raw_messages r ON r.id=p.raw_message_id WHERE COALESCE(r.received_at,p.created_at)>=? AND COALESCE(r.received_at,p.created_at)<?`
		args = []any{f.Start.UTC(), f.End.UTC()}
		q, args = reportDeviceWhere(q, args, f, "d")
		q += " ORDER BY received_at,p.id"
	case "events":
		report.Title = "Laporan Event Perangkat"
		report.Columns = []Column{{"plate_number", "Kendaraan", "text", 25}, {"imei", "IMEI", "text", 29}, {"event_time", "Waktu Event (WIB)", "date", 39}, {"received_at", "Diterima (WIB)", "date", 39}, {"message_type", "Report", "text", 25}, {"event_code", "Kode", "text", 20}, {"event_value", "Keterangan", "text", 76}}
		q = `SELECT e.id,d.imei,v.plate_number,e.event_time,COALESCE(r.received_at,e.created_at) AS received_at,e.message_type,e.event_code,e.event_value FROM device_events e JOIN gps_devices d ON d.id=e.device_id LEFT JOIN vehicles v ON v.id=d.vehicle_id LEFT JOIN device_raw_messages r ON r.id=e.raw_message_id WHERE COALESCE(r.received_at,e.created_at)>=? AND COALESCE(r.received_at,e.created_at)<?`
		args = []any{f.Start.UTC(), f.End.UTC()}
		q, args = reportDeviceWhere(q, args, f, "d")
		q += " ORDER BY received_at,e.id"
	case "messages":
		report.Title = "Laporan Penerimaan Data"
		report.Columns = []Column{{"id", "ID Pesan", "number", 18}, {"plate_number", "Kendaraan", "text", 23}, {"imei", "IMEI", "text", 28}, {"received_at", "Diterima (WIB)", "date", 36}, {"message_type", "Report", "text", 22}, {"parse_status", "Status", "text", 20}, {"parse_error", "Catatan Parsing", "text", 45}, {"raw_payload", "Raw Payload", "text", 72}}
		q = `SELECT r.id,r.imei,v.plate_number,r.received_at,COALESCE(r.message_type,SUBSTRING_INDEX(r.raw_payload,',',1)) AS message_type,r.parse_status,r.parse_error,r.raw_payload FROM device_raw_messages r LEFT JOIN gps_devices d ON d.id=r.device_id LEFT JOIN vehicles v ON v.id=d.vehicle_id WHERE r.received_at>=? AND r.received_at<?`
		args = []any{f.Start.UTC(), f.End.UTC()}
		q, args = reportDeviceWhere(q, args, f, "d")
		if f.Status != "" {
			q += " AND r.parse_status=?"
			args = append(args, f.Status)
		}
		q += " ORDER BY r.received_at,r.id"
	case "vehicles":
		report.Title = "Master Data Kendaraan"
		report.Columns = []Column{{"plate_number", "Nomor Polisi", "text", 32}, {"vehicle_name", "Nama Kendaraan", "text", 54}, {"vehicle_type", "Jenis", "text", 28}, {"fleet_code", "Fleet", "text", 30}, {"status", "Status", "text", 23}, {"device_count", "Perangkat", "number", 23}, {"updated_at", "Diperbarui (WIB)", "date", 44}}
		q = `SELECT v.id,v.plate_number,v.vehicle_name,v.vehicle_type,v.fleet_code,IF(v.is_active=1,'Aktif','Nonaktif') AS status,(SELECT COUNT(*) FROM gps_devices d WHERE d.vehicle_id=v.id AND d.deleted_at IS NULL) AS device_count,v.updated_at FROM vehicles v WHERE v.deleted_at IS NULL`
		if f.VehicleID > 0 {
			q += " AND v.id=?"
			args = append(args, f.VehicleID)
		}
		if f.DeviceID > 0 {
			q += " AND EXISTS (SELECT 1 FROM gps_devices d WHERE d.vehicle_id=v.id AND d.id=?)"
			args = append(args, f.DeviceID)
		}
		if f.Status != "" {
			q += " AND v.is_active=?"
			args = append(args, f.Status == "active")
		}
		q += " ORDER BY v.plate_number"
	case "devices":
		report.Title = "Master Data Perangkat"
		report.Columns = []Column{{"imei", "IMEI", "text", 29}, {"model", "Model", "text", 30}, {"plate_number", "Kendaraan", "text", 27}, {"protocol_name", "Protokol", "text", 27}, {"firmware_version", "Firmware", "text", 48}, {"status", "Status", "text", 23}, {"last_seen_at", "Terakhir Diterima (WIB)", "date", 50}}
		q = `SELECT d.id,d.imei,d.model,v.plate_number,d.protocol_name,d.firmware_version,IF(d.is_active=1,'Aktif','Nonaktif') AS status,d.last_seen_at FROM gps_devices d LEFT JOIN vehicles v ON v.id=d.vehicle_id WHERE d.deleted_at IS NULL`
		q, args = reportDeviceWhere(q, args, f, "d")
		if f.Status != "" {
			q += " AND d.is_active=?"
			args = append(args, f.Status == "active")
		}
		q += " ORDER BY d.imei"
	}
	q += " LIMIT ?"
	args = append(args, limit+1)
	report.Rows, e = queryTx(ctx, tx, q, args...)
	if e != nil {
		return report, e
	}
	if len(report.Rows) > limit {
		return report, fmt.Errorf("Laporan melebihi batas %d baris. Persempit periode atau filter; data tidak dipotong otomatis.", limit)
	}
	if e = tx.Commit(); e != nil {
		return report, e
	}
	report.Summary = [][2]string{{"Jumlah baris", strconv.Itoa(len(report.Rows))}}
	distinct := map[string]bool{}
	failed := 0
	var total, max float64
	var count int
	for _, r := range report.Rows {
		if s := text(r["imei"]); s != "-" {
			distinct[s] = true
		}
		if text(r["parse_status"]) == "failed" {
			failed++
		}
		if v, ok := asNumber(r["speed_kmh"]); ok {
			total += v
			count++
			if v > max {
				max = v
			}
		}
	}
	if f.Kind != "vehicles" {
		report.Summary = append(report.Summary, [2]string{"IMEI unik", strconv.Itoa(len(distinct))})
	}
	if f.Kind == "positions" {
		avg := 0.
		if count > 0 {
			avg = total / float64(count)
		}
		report.Summary = append(report.Summary, [2]string{"Rata-rata sampel (km/h)", fmt.Sprintf("%.2f", avg)}, [2]string{"Kecepatan maksimum (km/h)", fmt.Sprintf("%.2f", max)})
	}
	if f.Kind == "messages" {
		report.Summary = append(report.Summary, [2]string{"Gagal parsing", strconv.Itoa(failed)})
	}
	return report, nil
}
func reportDeviceWhere(q string, args []any, f ReportFilter, alias string) (string, []any) {
	if f.DeviceID > 0 {
		q += " AND " + alias + ".id=?"
		args = append(args, f.DeviceID)
	}
	if f.VehicleID > 0 {
		q += " AND " + alias + ".vehicle_id=?"
		args = append(args, f.VehicleID)
	}
	return q, args
}
func text(v any) string {
	if v == nil {
		return "-"
	}
	switch x := v.(type) {
	case time.Time:
		return x.In(wib).Format("02 Jan 2006 15:04:05")
	case []byte:
		return string(x)
	default:
		return fmt.Sprint(v)
	}
}
func asNumber(v any) (float64, bool) {
	if v == nil {
		return 0, false
	}
	n, e := strconv.ParseFloat(text(v), 64)
	return n, e == nil
}
func (c *Report) Get() {
	if !authorize(&c.Controller, false) {
		return
	}
	f, e := parseReportFilter(c.Ctx.Request)
	if e != nil {
		reply(&c.Controller, 422, map[string]string{"error": e.Error()})
		return
	}
	format := c.GetString("format")
	if format != "" && format != "xlsx" && format != "pdf" {
		reply(&c.Controller, 422, map[string]string{"error": "Format laporan tidak valid."})
		return
	}
	select {
	case exportSlots <- struct{}{}:
		defer func() { <-exportSlots }()
	default:
		reply(&c.Controller, 429, map[string]string{"error": "Dua laporan sedang diproses. Coba lagi sebentar."})
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 60*time.Second)
	defer cancel()
	limit := 50000
	if format == "pdf" {
		limit = 5000
	}
	r, e := buildReport(ctx, f, limit)
	if e != nil {
		log.Printf("report build: %v", e)
		msg := "Laporan belum dapat dibuat. Coba lagi sebentar."
		status := 503
		if strings.Contains(e.Error(), "batas") {
			msg = e.Error()
			status = 422
		}
		reply(&c.Controller, status, map[string]string{"error": msg})
		return
	}
	if format == "" {
		rows := r.Rows
		if len(rows) > 50 {
			rows = rows[:50]
		}
		reply(&c.Controller, 200, map[string]any{"report_id": r.ID, "title": r.Title, "columns": r.Columns, "data": rows, "total": len(r.Rows), "summary": r.Summary, "note": r.Note, "generated_at": r.Generated, "filters": reportFilters(r)})
		return
	}
	var data []byte
	if format == "xlsx" {
		data, e = makeXLSX(r)
	} else {
		data, e = makePDF(r)
	}
	if e != nil {
		log.Printf("report export: %v", e)
		reply(&c.Controller, 503, map[string]string{"error": "File laporan belum dapat dibuat."})
		return
	}
	digest := sha256.Sum256(data)
	checksum := hex.EncodeToString(digest[:])
	tx, e := db.BeginTx(ctx, nil)
	if e == nil {
		defer tx.Rollback()
		e = audit(ctx, tx, "export", "reports", nil, nil, map[string]any{"report_id": r.ID, "format": format, "type": f.Kind, "filters": reportFilters(r), "rows": len(r.Rows), "sha256": checksum, "summary": r.Summary}, r.ID)
		if e == nil {
			e = tx.Commit()
		}
	}
	if e != nil {
		log.Printf("report audit: %v", e)
		reply(&c.Controller, 503, map[string]string{"error": "Audit ekspor belum dapat disimpan. File belum dikirim."})
		return
	}
	mime := "application/pdf"
	if format == "xlsx" {
		mime = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}
	c.Ctx.Output.Header("Content-Type", mime)
	c.Ctx.Output.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="pastrack-%s-%s.%s"`, f.Kind, r.ID, format))
	c.Ctx.Output.Header("X-Report-ID", r.ID)
	c.Ctx.Output.Header("X-Content-SHA256", checksum)
	c.Ctx.Output.Body(data)
}
func reportFilters(r ReportData) [][2]string {
	period := r.Filter.Start.In(wib).Format("02 Jan 2006") + " s/d " + r.Filter.End.Add(-time.Second).In(wib).Format("02 Jan 2006")
	if r.Filter.Kind == "vehicles" || r.Filter.Kind == "devices" {
		period = "Snapshot master saat pembuatan laporan"
	}
	device := "Semua perangkat"
	if r.Filter.DeviceID > 0 {
		device = fmt.Sprintf("Device ID %d", r.Filter.DeviceID)
	}
	vehicle := "Semua kendaraan"
	if r.Filter.VehicleID > 0 {
		vehicle = fmt.Sprintf("Vehicle ID %d", r.Filter.VehicleID)
	}
	status := r.Filter.Status
	if status == "" {
		status = "Semua status"
	}
	return [][2]string{{"Periode", period}, {"Filter perangkat", device}, {"Filter kendaraan", vehicle}, {"Status", status}, {"Basis waktu", func() string {
		if r.Filter.Kind == "vehicles" || r.Filter.Kind == "devices" {
			return "Snapshot master; arsip dikecualikan"
		}
		return "Waktu penerimaan data, awal inklusif / akhir eksklusif"
	}()}}
}
func makeXLSX(r ReportData) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	f.SetSheetName("Sheet1", "Ringkasan")
	f.NewSheet("Detail")
	f.NewSheet("Definisi")
	f.SetDocProps(&excelize.DocProperties{Title: r.Title, Subject: "Pastrack operational report", Creator: r.Actor, Description: r.ID, Category: "Confidential - Internal"})
	title, e := f.NewStyle(&excelize.Style{Font: &excelize.Font{Size: 23, Bold: true, Color: "087F72"}})
	if e != nil {
		return nil, e
	}
	header, e := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11}, Fill: excelize.Fill{Type: "pattern", Color: []string{"087F72"}, Pattern: 1}, Alignment: &excelize.Alignment{Vertical: "center", WrapText: true}})
	if e != nil {
		return nil, e
	}
	body, e := f.NewStyle(&excelize.Style{Font: &excelize.Font{Size: 10, Color: "334B45"}, Alignment: &excelize.Alignment{Vertical: "top", WrapText: true}})
	if e != nil {
		return nil, e
	}
	dateStyle, e := f.NewStyle(&excelize.Style{NumFmt: 22, CustomNumFmt: strptr("dd mmm yyyy hh:mm:ss"), Font: &excelize.Font{Size: 10}, Alignment: &excelize.Alignment{Vertical: "top"}})
	if e != nil {
		return nil, e
	}
	number, e := f.NewStyle(&excelize.Style{CustomNumFmt: strptr("0.00;[Red](0.00);0.00"), Font: &excelize.Font{Size: 10}, Alignment: &excelize.Alignment{Vertical: "top"}})
	if e != nil {
		return nil, e
	}
	coordinate, e := f.NewStyle(&excelize.Style{CustomNumFmt: strptr("0.0000000"), Font: &excelize.Font{Size: 10}, Alignment: &excelize.Alignment{Vertical: "top"}})
	if e != nil {
		return nil, e
	}
	integral, e := f.NewStyle(&excelize.Style{CustomNumFmt: strptr("0"), Font: &excelize.Font{Size: 10}, Alignment: &excelize.Alignment{Vertical: "top"}})
	if e != nil {
		return nil, e
	}
	f.MergeCell("Ringkasan", "A1", "H2")
	f.SetCellStr("Ringkasan", "A1", "PASTRACK / OPERATIONS")
	f.SetCellStyle("Ringkasan", "A1", "H2", title)
	f.MergeCell("Ringkasan", "A4", "H4")
	f.SetCellStr("Ringkasan", "A4", r.Title)
	f.SetCellStyle("Ringkasan", "A4", "H4", header)
	f.SetRowHeight("Ringkasan", 4, 28)
	meta := [][2]string{{"Report ID", r.ID}, {"Dibuat (WIB)", r.Generated.In(wib).Format("02 Jan 2006 15:04:05")}, {"Dibuat oleh", r.Actor}, {"Klasifikasi", "INTERNAL / CONFIDENTIAL"}}
	meta = append(meta, reportFilters(r)...)
	meta = append(meta, r.Summary...)
	for i, p := range meta {
		row := i + 6
		f.SetCellStr("Ringkasan", fmt.Sprintf("A%d", row), p[0])
		f.MergeCell("Ringkasan", fmt.Sprintf("C%d", row), fmt.Sprintf("H%d", row))
		if n, e := strconv.ParseFloat(p[1], 64); e == nil {
			f.SetCellValue("Ringkasan", fmt.Sprintf("C%d", row), n)
		} else {
			f.SetCellStr("Ringkasan", fmt.Sprintf("C%d", row), p[1])
		}
		f.SetRowHeight("Ringkasan", row, 24)
	}
	f.SetColWidth("Ringkasan", "A", "B", 18)
	f.SetColWidth("Ringkasan", "C", "H", 14)
	f.SetCellStyle("Ringkasan", "A6", fmt.Sprintf("H%d", len(meta)+6), body)
	last, _ := excelize.ColumnNumberToName(len(r.Columns))
	f.MergeCell("Detail", "A1", last+"1")
	f.SetCellStr("Detail", "A1", r.Title)
	f.SetCellStyle("Detail", "A1", last+"1", title)
	f.SetRowHeight("Detail", 1, 34)
	f.MergeCell("Detail", "A2", last+"2")
	f.SetCellStr("Detail", "A2", r.ID+" | WIB (UTC+7) | "+strconv.Itoa(len(r.Rows))+" baris")
	headers := make([]any, len(r.Columns))
	for i, c := range r.Columns {
		headers[i] = c.Label
		col, _ := excelize.ColumnNumberToName(i + 1)
		width := c.Width * .8
		if width < 16 {
			width = 16
		}
		if width > 70 {
			width = 70
		}
		f.SetColWidth("Detail", col, col, width)
	}
	f.SetSheetRow("Detail", "A4", &headers)
	f.SetRowHeight("Detail", 4, 32)
	f.SetCellStyle("Detail", "A4", last+"4", header)
	for i, row := range r.Rows {
		for j, col := range r.Columns {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+5)
			v := row[col.Key]
			if v == nil {
				continue
			}
			if col.Kind == "date" {
				if t, ok := v.(time.Time); ok {
					local := t.In(wib)
					wall := time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), local.Minute(), local.Second(), 0, time.UTC)
					f.SetCellValue("Detail", cell, wall)
					f.SetCellStyle("Detail", cell, cell, dateStyle)
				}
			} else if col.Kind == "number" {
				if n, ok := asNumber(v); ok {
					f.SetCellValue("Detail", cell, n)
					style := number
					if col.Key == "latitude" || col.Key == "longitude" {
						style = coordinate
					}
					if col.Key == "id" || col.Key == "heading_deg" || col.Key == "device_count" {
						style = integral
					}
					f.SetCellStyle("Detail", cell, cell, style)
				}
			} else {
				value := text(v)
				if len([]rune(value)) > 32767 {
					return nil, errors.New("Payload melebihi batas satu sel Excel; ekspor dihentikan agar data tidak terpotong")
				}
				f.SetCellStr("Detail", cell, value)
				f.SetCellStyle("Detail", cell, cell, body)
			}
		}
		f.SetRowHeight("Detail", i+5, func() float64 {
			if r.Filter.Kind == "messages" {
				return 68
			}
			return 30
		}())
	}
	if len(r.Rows) > 0 {
		f.AddTable("Detail", &excelize.Table{Range: fmt.Sprintf("A4:%s%d", last, len(r.Rows)+4), Name: "PastrackDetail", StyleName: "TableStyleMedium2", ShowRowStripes: boolptr(true)})
	}
	f.SetPanes("Detail", &excelize.Panes{Freeze: true, YSplit: 4, TopLeftCell: "A5", ActivePane: "bottomLeft"})
	f.SetSheetView("Detail", 0, &excelize.ViewOptions{ShowGridLines: boolptr(false)})
	definitions := [][2]string{{"Catatan sumber", r.Note}, {"Online", "Master aktif dengan data diterima dalam 5 menit; tidak membuktikan posisi GPS baru."}, {"Waktu GPS", "Waktu fix perangkat; dapat lebih lama dari waktu penerimaan."}, {"Kecepatan rata-rata", "Rata-rata aritmetika sampel valid, bukan rata-rata perjalanan atau konsumsi."}, {"Status parsing", "Status tersimpan dari listener; failed pada ACK bukan bukti perintah ditolak tracker."}, {"Fuel", "Nilai liter dan konsumsi tidak dihitung sebelum protokol dan kalibrasi terverifikasi."}, {"Master snapshot", "Master kendaraan/perangkat menggunakan identitas dan relasi saat laporan dibuat; periode tidak membatasi snapshot."}, {"Batas ekspor", "Excel maksimal 50.000 baris, PDF 5.000 baris; permintaan lebih besar ditolak, tidak dipotong."}, {"Integritas", "SHA-256 file dicatat dalam jejak audit ekspor bersama Report ID."}}
	f.SetColWidth("Definisi", "A", "A", 28)
	f.SetColWidth("Definisi", "B", "B", 110)
	for i, p := range definitions {
		f.SetCellStr("Definisi", fmt.Sprintf("A%d", i+1), p[0])
		f.SetCellStr("Definisi", fmt.Sprintf("B%d", i+1), p[1])
		f.SetRowHeight("Definisi", i+1, 48)
	}
	f.SetCellStyle("Definisi", "A1", fmt.Sprintf("B%d", len(definitions)), body)
	orientation := "landscape"
	paper := 9
	fitWidth := 1
	fitHeight := 0
	for _, s := range []string{"Ringkasan", "Detail", "Definisi"} {
		f.SetPageLayout(s, &excelize.PageLayoutOptions{Orientation: &orientation, Size: &paper, FitToWidth: &fitWidth, FitToHeight: &fitHeight})
		f.SetHeaderFooter(s, &excelize.HeaderFooterOptions{OddHeader: "&LPASTRACK&C" + r.Title + "&RINTERNAL", OddFooter: "&L" + r.ID + "&CPage &P of &N&RWIB (UTC+7)"})
	}
	f.SetDefinedName(&excelize.DefinedName{Name: "_xlnm.Print_Titles", RefersTo: "'Detail'!$1:$4", Scope: "Detail"})
	buf, e := f.WriteToBuffer()
	if e != nil {
		return nil, e
	}
	return buf.Bytes(), nil
}
func strptr(s string) *string { return &s }
func boolptr(b bool) *bool    { return &b }
func makePDF(r ReportData) ([]byte, error) {
	p := fpdf.New("L", "mm", "A4", "")
	p.SetMargins(14, 15, 14)
	p.SetAutoPageBreak(false, 15)
	p.AddUTF8FontFromBytes("Go", "", goregular.TTF)
	p.AddUTF8FontFromBytes("Go", "B", gobold.TTF)
	p.SetTitle(r.Title, false)
	p.SetAuthor(r.Actor, false)
	p.SetSubject(r.ID, false)
	p.AliasNbPages("")
	p.SetFooterFunc(func() {
		p.SetY(-11)
		p.SetTextColor(120, 135, 130)
		p.SetFont("Go", "", 7)
		p.CellFormat(120, 5, r.ID+" | INTERNAL / CONFIDENTIAL", "", 0, "L", false, 0, "")
		p.CellFormat(149, 5, fmt.Sprintf("Halaman %d / {nb}  |  WIB (UTC+7)", p.PageNo()), "", 0, "R", false, 0, "")
	})
	heading := func(detail bool) {
		p.AddPage()
		p.SetFillColor(8, 127, 114)
		p.Rect(14, 14, 2, 14, "F")
		p.SetXY(20, 14)
		p.SetTextColor(8, 127, 114)
		p.SetFont("Go", "B", 17)
		p.Cell(170, 8, "PASTRACK")
		p.SetFont("Go", "", 8)
		p.SetTextColor(100, 120, 110)
		p.CellFormat(93, 7, "OPERATIONS / INTERNAL", "", 1, "R", false, 0, "")
		p.SetX(20)
		p.SetFont("Go", "B", 12)
		p.SetTextColor(30, 55, 45)
		p.Cell(240, 7, r.Title)
		p.Ln(12)
		if detail {
			p.SetFont("Go", "", 8)
			p.Cell(269, 5, r.ID+" | "+reportFilters(r)[0][1])
			p.Ln(9)
		}
	}
	heading(false)
	p.SetFont("Go", "", 9)
	p.SetTextColor(70, 90, 80)
	metadata := [][2]string{{"Report ID", r.ID}, {"Dibuat (WIB)", r.Generated.In(wib).Format("02 Jan 2006 15:04:05")}, {"Dibuat oleh", r.Actor}}
	metadata = append(metadata, reportFilters(r)...)
	for _, m := range metadata {
		p.SetFont("Go", "B", 8)
		p.Cell(45, 6, m[0])
		p.SetFont("Go", "", 8)
		p.MultiCell(224, 6, m[1], "", "L", false)
	}
	p.Ln(5)
	p.SetFont("Go", "B", 10)
	p.SetTextColor(8, 127, 114)
	p.Cell(269, 7, "RINGKASAN LAPORAN")
	p.Ln(9)
	width := 269 / float64(len(r.Summary))
	y := p.GetY()
	for i, s := range r.Summary {
		x := 14 + float64(i)*width
		p.SetFillColor(235, 246, 240)
		p.Rect(x, y, width-3, 23, "F")
		p.SetXY(x+4, y+3)
		p.SetFont("Go", "", 8)
		p.SetTextColor(85, 110, 95)
		p.Cell(width-9, 6, s[0])
		p.SetXY(x+4, y+10)
		p.SetFont("Go", "B", 16)
		p.SetTextColor(8, 127, 114)
		p.Cell(width-9, 9, s[1])
	}
	p.SetXY(14, y+29)
	p.SetFont("Go", "", 8)
	p.SetTextColor(90, 110, 100)
	p.MultiCell(269, 5, r.Note, "", "L", false)
	p.Ln(4)
	p.MultiCell(269, 5, "Dokumen ini menggunakan snapshot database yang konsisten. Periode data operasional difilter berdasarkan waktu penerimaan. Rata-rata kecepatan merupakan rata-rata sampel, bukan pengukuran perjalanan.", "", "L", false)
	heading(true)
	sumWidth := 0.
	for _, c := range r.Columns {
		sumWidth += c.Width
	}
	widths := make([]float64, len(r.Columns))
	for i, c := range r.Columns {
		widths[i] = 269 * c.Width / sumWidth
	}
	tableHeader := func() {
		p.SetFont("Go", "B", 7)
		p.SetTextColor(255, 255, 255)
		p.SetFillColor(8, 127, 114)
		x, y := 14., p.GetY()
		for i, c := range r.Columns {
			p.Rect(x, y, widths[i], 12, "F")
			p.SetXY(x+2, y+2)
			p.MultiCell(widths[i]-4, 4, c.Label, "", "L", false)
			x += widths[i]
		}
		p.SetXY(14, y+12)
	}
	tableHeader()
	p.SetFont("Go", "", 7)
	if len(r.Rows) == 0 {
		p.SetTextColor(100, 120, 110)
		p.Ln(10)
		p.Cell(269, 8, "Tidak ada data yang sesuai dengan filter laporan.")
	}
	for i, row := range r.Rows {
		p.SetFont("Go", "", 7)
		cells := make([]string, len(r.Columns))
		height := 9.
		for j, c := range r.Columns {
			cells[j] = text(row[c.Key])
			if c.Kind == "number" {
				if n, ok := asNumber(row[c.Key]); ok {
					if c.Key == "latitude" || c.Key == "longitude" {
						cells[j] = fmt.Sprintf("%.7f", n)
					} else if c.Key == "id" {
						cells[j] = fmt.Sprintf("%.0f", n)
					} else {
						cells[j] = fmt.Sprintf("%.2f", n)
					}
				}
			}
			lines := p.SplitText(cells[j], widths[j]-4)
			h := float64(len(lines))*3.8 + 5
			if h > height {
				height = h
			}
		}
		// Payloads are wrapped and continued across pages without truncation.
		if height > 140 {
			for j := range cells {
				lines := p.SplitText(cells[j], widths[j]-4)
				if len(lines) > 28 {
					cells[j] = strings.Join(lines[:26], "\n") + "\n[Lihat lampiran payload #" + text(row["id"]) + "]"
				}
			}
			height = 125
		}
		if p.GetY()+height > 192 {
			heading(true)
			tableHeader()
		}
		x, y := 14., p.GetY()
		for j, cell := range cells {
			if i%2 == 0 {
				p.SetFillColor(244, 248, 246)
			} else {
				p.SetFillColor(255, 255, 255)
			}
			p.Rect(x, y, widths[j], height, "F")
			p.SetXY(x+2, y+2)
			p.SetTextColor(55, 75, 65)
			p.SetFont("Go", "", 7)
			p.MultiCell(widths[j]-4, 3.8, cell, "", "L", false)
			x += widths[j]
		}
		p.SetDrawColor(225, 235, 229)
		p.Line(14, y+height, 283, y+height)
		p.SetXY(14, y+height)
		if height == 125 {
			heading(true)
			p.SetFont("Go", "B", 9)
			p.Cell(269, 7, "Lampiran payload #"+text(row["id"]))
			p.Ln(10)
			p.SetFont("Go", "", 7)
			for j, c := range r.Columns {
				if len(p.SplitText(text(row[c.Key]), widths[j]-4)) > 28 {
					p.SetFont("Go", "B", 8)
					p.Cell(269, 6, c.Label)
					p.Ln(7)
					p.SetFont("Go", "", 7)
					for _, line := range p.SplitText(text(row[c.Key]), 265) {
						if p.GetY() > 185 {
							heading(true)
							p.SetFont("Go", "", 7)
						}
						p.Cell(269, 4, line)
						p.Ln(4)
					}
				}
			}
			heading(true)
			tableHeader()
		}
	}
	var buf bytes.Buffer
	e := p.Output(&buf)
	return buf.Bytes(), e
}
func (c *AuditView) Get() {
	if !allowRole(&c.Controller, "admin") {
		return
	}
	if !authorize(&c.Controller, false) {
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 8*time.Second)
	defer cancel()
	page := integer(c.GetString("page"), 1, 1, 10000)
	rows, e := query(ctx, `SELECT id,occurred_at,actor,action,entity,entity_id,before_data,after_data,request_id FROM pastrack_audit_log ORDER BY id DESC LIMIT 51 OFFSET ?`, (page-1)*50)
	if e != nil {
		reply(&c.Controller, 503, map[string]string{"error": "Audit belum dapat dimuat."})
		return
	}
	reply(&c.Controller, 200, map[string]any{"data": rows, "page": page})
}
