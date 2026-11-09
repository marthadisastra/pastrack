package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Metric struct {
	Label string `json:"label"`
	Value any    `json:"value"`
	Unit  string `json:"unit,omitempty"`
}
type DashboardColumn struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Kind  string `json:"kind,omitempty"`
}
type DashboardTable struct {
	Title   string            `json:"title"`
	Columns []DashboardColumn `json:"columns"`
	Rows    []map[string]any  `json:"rows"`
}
type DashboardData struct {
	Kind        string           `json:"kind"`
	GeneratedAt time.Time        `json:"generated_at"`
	Since       time.Time        `json:"since"`
	Metrics     []Metric         `json:"metrics"`
	Tables      []DashboardTable `json:"tables"`
	Trend       []map[string]any `json:"trend"`
	Notes       []string         `json:"notes"`
}

func col(key, label string) DashboardColumn { return DashboardColumn{Key: key, Label: label} }
func datecol(key, label string) DashboardColumn {
	return DashboardColumn{Key: key, Label: label, Kind: "date"}
}
func Analytics(ctx context.Context, kind string, hours int) (DashboardData, error) {
	switch kind {
	case "vehicles", "gps", "activity", "fuel":
	default:
		return DashboardData{}, fmt.Errorf("unknown dashboard")
	}
	tx, txErr := DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if txErr != nil {
		return DashboardData{}, txErr
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	since := now.Add(-time.Duration(hours) * time.Hour)
	out := DashboardData{Kind: kind, GeneratedAt: now, Since: since, Metrics: []Metric{}, Tables: []DashboardTable{}, Trend: []map[string]any{}, Notes: []string{}}
	var err error
	addTable := func(title string, columns []DashboardColumn, sql string, args ...any) {
		if err != nil {
			return
		}
		var rows []map[string]any
		rows, err = QueryOn(ctx, tx, sql, args...)
		out.Tables = append(out.Tables, DashboardTable{title, columns, rows})
	}
	metrics := func(sql string, args []any, labels ...string) {
		if err != nil {
			return
		}
		var rows []map[string]any
		rows, err = QueryOn(ctx, tx, sql, args...)
		if err != nil || len(rows) == 0 {
			return
		}
		for i, label := range labels {
			out.Metrics = append(out.Metrics, Metric{Label: label, Value: rows[0][fmt.Sprintf("m%d", i)]})
		}
	}
	switch kind {
	case "vehicles":
		err = vehicleAnalytics(ctx, tx, &out)
	case "gps":
		valid := `latitude BETWEEN -90 AND 90 AND longitude BETWEEN -180 AND 180 AND NOT(latitude=0 AND longitude=0) AND gps_time IS NOT NULL`
		metrics(`SELECT COUNT(*) m0,COALESCE(SUM(`+valid+`),0) m1,COUNT(DISTINCT CASE WHEN `+valid+` THEN gps_time END,CASE WHEN `+valid+` THEN latitude END,CASE WHEN `+valid+` THEN longitude END) m2,COALESCE(SUM(TIMESTAMPDIFF(SECOND,gps_time,created_at)>900),0) m3 FROM gps_positions WHERE device_id IN (SELECT id FROM gps_devices WHERE deleted_at IS NULL) AND created_at>=?`, []any{since}, "Laporan posisi", "Koordinat valid", "Fix unik lintas device*", "Terlambat >15 mnt")
		// Count unique fixes per device, so two vehicles at the same location are distinct.
		if err == nil {
			var unique int
			err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM (SELECT DISTINCT device_id,gps_time,latitude,longitude FROM gps_positions WHERE device_id IN (SELECT id FROM gps_devices WHERE deleted_at IS NULL) AND created_at>=? AND `+valid+`) fixes`, since).Scan(&unique)
			if err == nil {
				out.Metrics[2].Value = unique
				out.Metrics[2].Label = "Fix GPS unik"
			}
		}
		addTable("Kualitas GPS per device", []DashboardColumn{col("plate_number", "Kendaraan"), col("imei", "IMEI"), col("samples", "Laporan"), col("unique_fixes", "Fix unik"), col("late_samples", "Terlambat >15 mnt"), datecol("latest_gps", "GPS terbaru (WIB)"), datecol("last_received", "Diterima terakhir (WIB)")}, `SELECT d.id,d.imei,v.plate_number,COUNT(p.id) samples,COUNT(DISTINCT CASE WHEN p.latitude BETWEEN -90 AND 90 AND p.longitude BETWEEN -180 AND 180 AND NOT(p.latitude=0 AND p.longitude=0) THEN p.gps_time END,CASE WHEN p.latitude BETWEEN -90 AND 90 AND p.longitude BETWEEN -180 AND 180 AND NOT(p.latitude=0 AND p.longitude=0) THEN p.latitude END,CASE WHEN p.latitude BETWEEN -90 AND 90 AND p.longitude BETWEEN -180 AND 180 AND NOT(p.latitude=0 AND p.longitude=0) THEN p.longitude END) unique_fixes,COALESCE(SUM(TIMESTAMPDIFF(SECOND,p.gps_time,p.created_at)>900),0) late_samples,MAX(p.gps_time) latest_gps,MAX(p.created_at) last_received FROM gps_devices d LEFT JOIN vehicles v ON v.id=d.vehicle_id LEFT JOIN gps_positions p ON p.device_id=d.id AND p.created_at>=? WHERE d.deleted_at IS NULL GROUP BY d.id,d.imei,v.plate_number ORDER BY samples DESC,d.id LIMIT 501`, since)
		out.Notes = append(out.Notes, "Periode berdasarkan waktu penerimaan; fix unik dibedakan per device, waktu GPS, dan koordinat.")
	case "activity":
		metrics(`SELECT COUNT(*) m0,COALESCE(SUM(parse_status='success'),0) m1,COALESCE(SUM(parse_status='partial'),0) m2,COALESCE(SUM(parse_status='failed'),0) m3 FROM device_raw_messages WHERE received_at>=?`, []any{since}, "Pesan diterima", "Parsing berhasil", "Parsing parsial", "Parsing gagal")
		addTable("Jenis laporan", []DashboardColumn{col("message_type", "Jenis report"), col("total", "Pesan"), col("success", "Berhasil"), col("partial", "Parsial"), col("failed", "Gagal"), datecol("latest", "Terakhir (WIB)")}, `SELECT COALESCE(message_type,'Unknown') message_type,COUNT(*) total,SUM(parse_status='success') success,SUM(parse_status='partial') partial,SUM(parse_status='failed') failed,MAX(received_at) latest FROM device_raw_messages WHERE received_at>=? GROUP BY message_type ORDER BY total DESC,message_type LIMIT 501`, since)
		if err == nil {
			out.Trend, err = QueryOn(ctx, tx, `SELECT DATE_FORMAT(DATE_ADD(received_at,INTERVAL 7 HOUR),'%Y-%m-%d %H:00') bucket,COUNT(*) total,SUM(parse_status='failed') failed FROM device_raw_messages WHERE received_at>=? GROUP BY bucket ORDER BY bucket`, since)
		}
		out.Notes = append(out.Notes, "Berhasil/parsial/gagal adalah status parser; bukan status konektivitas device.")
	case "fuel":
		candidate := `r.message_type IN ('GTBAR','GTERI','GTBAA')`
		metrics(`SELECT COUNT(*) m0,COUNT(DISTINCT r.imei) m1 FROM device_raw_messages r WHERE r.received_at>=? AND `+candidate, []any{since}, "Laporan BLE", "GPS pelapor")
		metrics(`SELECT COUNT(*) m0,COALESCE(SUM(EXISTS(SELECT 1 FROM fuel_calibration_points p WHERE p.sensor_pk=s.id AND p.level_percent=0) AND EXISTS(SELECT 1 FROM fuel_calibration_points p WHERE p.sensor_pk=s.id AND p.level_percent=100)),0) m1 FROM device_fuel_sensors s JOIN gps_devices d ON d.id=s.device_id WHERE d.deleted_at IS NULL AND s.deleted_at IS NULL`, nil, "Sensor terhubung", "Profil lengkap")
		addTable("Penerimaan aksesori per IMEI", []DashboardColumn{col("imei", "IMEI"), col("plate_number", "Kendaraan"), col("candidates", "Laporan BLE"), datecol("latest", "Diterima terakhir (WIB)")}, `SELECT r.imei,v.plate_number,COUNT(r.id) candidates,MAX(r.received_at) latest FROM device_raw_messages r LEFT JOIN gps_devices d ON d.imei=r.imei AND d.deleted_at IS NULL LEFT JOIN vehicles v ON v.id=d.vehicle_id WHERE r.received_at>=? AND `+candidate+` GROUP BY r.imei,v.plate_number ORDER BY candidates DESC,r.imei LIMIT 501`, since)
		out.Notes = append(out.Notes, "Persentase ditentukan oleh titik Kosong dan Penuh per sensor. Perhitungan liter memerlukan kapasitas dan profil tangki kendaraan.")
	default:
		return out, fmt.Errorf("unknown dashboard")
	}
	for i := range out.Tables {
		if kind != "vehicles" && len(out.Tables[i].Rows) > 500 {
			out.Tables[i].Rows = out.Tables[i].Rows[:500]
			out.Notes = append(out.Notes, "Tabel dibatasi 500 baris; gunakan laporan untuk ekspor data lengkap.")
		}
	}
	if err == nil {
		err = tx.Commit()
	}
	return out, err
}
