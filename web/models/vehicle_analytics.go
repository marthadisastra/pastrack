package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"
)

var ErrVehicleAnalysisLimit = errors.New("Analisis melebihi 250.000 titik GPS. Pilih periode lebih pendek.")

type vehicleFix struct {
	at              time.Time
	lat, lon, speed float64
	valid           bool
}
type vehicleMotion struct {
	samples, rejected, duplicates, gaps, inferred              int
	km, moving, stopped, weightedSpeed, speedSeconds, maxSpeed float64
	previous                                                   *vehicleFix
	daily                                                      map[string]float64
	hourly                                                     [24]float64
	latest                                                     time.Time
}

func newVehicleMotion() *vehicleMotion { return &vehicleMotion{daily: map[string]float64{}} }
func (m *vehicleMotion) add(p vehicleFix) {
	if !p.valid || math.IsNaN(p.lat) || math.IsNaN(p.lon) || math.IsInf(p.lat, 0) || math.IsInf(p.lon, 0) || math.Abs(p.lat) > 90 || math.Abs(p.lon) > 180 || (p.lat == 0 && p.lon == 0) || math.IsNaN(p.speed) || math.IsInf(p.speed, 0) || p.speed < 0 || p.speed > 200 {
		m.rejected++
		m.previous = nil
		return
	}
	if m.previous != nil && p.at.Equal(m.previous.at) {
		m.duplicates++
		return
	}
	m.samples++
	m.latest = p.at
	if p.speed > m.maxSpeed {
		m.maxSpeed = p.speed
	}
	if prev := m.previous; prev != nil {
		seconds := p.at.Sub(prev.at).Seconds()
		km := coordinateDistance(prev.lat, prev.lon, p.lat, p.lon) / 1000
		if seconds <= 0 || seconds > 1800 || km/seconds*3600 > 200 {
			m.gaps++
		} else {
			// Distance does not depend on reported speed. Match Tracking's 30-minute continuity rule.
			m.km += km
			inferred := km / seconds * 3600
			moving := seconds <= 300 && ((prev.speed > 3 && p.speed > 3) || (inferred > 3 && km >= 0.02))
			stopped := seconds <= 300 && prev.speed <= 3 && p.speed <= 3 && inferred <= 3
			if seconds > 300 {
				m.gaps++
			}
			if moving {
				m.moving += seconds
				effective := (prev.speed + p.speed) / 2
				if !(prev.speed > 3 && p.speed > 3) {
					effective = inferred
					m.inferred++
				}
				m.weightedSpeed += effective * seconds
				m.speedSeconds += seconds
			}
			if stopped {
				m.stopped += seconds
			}
			for cursor := prev.at; cursor.Before(p.at); {
				local := cursor.In(time.FixedZone("WIB", 7*3600))
				next := cursor.Truncate(time.Hour).Add(time.Hour)
				if next.After(p.at) {
					next = p.at
				}
				duration := next.Sub(cursor).Seconds()
				m.daily[local.Format("2006-01-02")] += km * duration / seconds
				if moving {
					m.hourly[local.Hour()] += duration
				}
				cursor = next
			}
		}
	}
	m.previous = &p
}
func rounded(n float64) float64 { return math.Round(n*100) / 100 }
func vehicleAnalytics(ctx context.Context, tx *sql.Tx, out *DashboardData) error {
	masters, e := QueryOn(ctx, tx, `SELECT id,plate_number,vehicle_name,fleet_code FROM vehicles WHERE deleted_at IS NULL ORDER BY plate_number`)
	if e != nil {
		return e
	}
	rows, e := tx.QueryContext(ctx, `SELECT d.vehicle_id,p.device_id,p.gps_time,p.latitude,p.longitude,p.speed_kmh FROM gps_positions p JOIN gps_devices d ON d.id=p.device_id JOIN vehicles v ON v.id=d.vehicle_id WHERE d.deleted_at IS NULL AND v.deleted_at IS NULL AND p.gps_time>=? AND p.gps_time<=? ORDER BY d.vehicle_id,p.device_id,p.gps_time,p.id LIMIT 250001`, out.Since, out.GeneratedAt)
	if e != nil {
		return e
	}
	defer rows.Close()
	devices := map[int64]map[int64]*vehicleMotion{}
	count := 0
	for rows.Next() {
		count++
		if count > 250000 {
			return ErrVehicleAnalysisLimit
		}
		var vehicle, device int64
		var at time.Time
		var lat, lon, speed sql.NullFloat64
		if e = rows.Scan(&vehicle, &device, &at, &lat, &lon, &speed); e != nil {
			return e
		}
		if devices[vehicle] == nil {
			devices[vehicle] = map[int64]*vehicleMotion{}
		}
		if devices[vehicle][device] == nil {
			devices[vehicle][device] = newVehicleMotion()
		}
		devices[vehicle][device].add(vehicleFix{at: at, lat: lat.Float64, lon: lon.Float64, speed: speed.Float64, valid: lat.Valid && lon.Valid && speed.Valid})
	}
	if e = rows.Err(); e != nil {
		return e
	}
	details := []map[string]any{}
	daily := map[string]float64{}
	var hourly [24]float64
	var km, moving, weighted, speedSeconds float64
	active, withData := 0, 0
	for _, v := range masters {
		id, _ := number(v["id"])
		selected := newVehicleMotion()
		var chosen int64
		for device, m := range devices[int64(id)] {
			if chosen == 0 || m.samples > selected.samples || (m.samples == selected.samples && device < chosen) {
				selected = m
				chosen = device
			}
		}
		m := selected
		km += m.km
		moving += m.moving
		weighted += m.weightedSpeed
		speedSeconds += m.speedSeconds
		if m.moving > 0 {
			active++
		}
		if m.samples > 0 {
			withData++
		}
		for day, value := range m.daily {
			daily[day] += value
		}
		for h, value := range m.hourly {
			hourly[h] += value
		}
		average := any(nil)
		if m.speedSeconds > 0 {
			average = rounded(m.weightedSpeed / m.speedSeconds)
		}
		coverage := m.moving + m.stopped
		latest := any(nil)
		if !m.latest.IsZero() {
			latest = m.latest.UTC().Format(time.RFC3339)
		}
		status := "Tidak ada data"
		if m.samples > 0 {
			status = "Data terbatas"
		}
		if coverage > 0 {
			status = "Teramati"
		}
		details = append(details, map[string]any{"id": v["id"], "plate_number": v["plate_number"], "vehicle_name": v["vehicle_name"], "fleet_code": v["fleet_code"], "distance_km": rounded(m.km), "moving_minutes": rounded(m.moving / 60), "stopped_minutes": rounded(m.stopped / 60), "average_speed": average, "max_speed": func() any {
			if m.samples == 0 {
				return nil
			}
			return rounded(m.maxSpeed)
		}(), "samples": m.samples, "rejected": m.rejected, "duplicates": m.duplicates, "gaps": m.gaps, "inferred_intervals": m.inferred, "coverage_percent": rounded(coverage / out.GeneratedAt.Sub(out.Since).Seconds() * 100), "latest": latest, "status": status, "device_id": chosen, "device_count": len(devices[int64(id)])})
	}
	sort.SliceStable(details, func(i, j int) bool { return details[i]["distance_km"].(float64) > details[j]["distance_km"].(float64) })
	avg := any("—")
	if speedSeconds > 0 {
		avg = rounded(weighted / speedSeconds)
	}
	out.Metrics = []Metric{{Label: "Kendaraan bergerak", Value: active}, {Label: "Estimasi jarak GPS", Value: rounded(km), Unit: "km"}, {Label: "Durasi bergerak teramati", Value: rounded(moving / 3600), Unit: "jam"}, {Label: "Rata-rata saat bergerak", Value: avg, Unit: "km/jam"}, {Label: "Kendaraan dengan data", Value: withData}, {Label: "Total kendaraan", Value: len(masters)}}
	out.Tables = []DashboardTable{{Title: "Analisis perilaku kendaraan", Columns: []DashboardColumn{col("plate_number", "Nomor polisi"), col("vehicle_name", "Kendaraan"), col("fleet_code", "Fleet"), col("distance_km", "Estimasi km"), col("moving_minutes", "Bergerak (mnt)"), col("stopped_minutes", "Berhenti teramati (mnt)"), col("average_speed", "Rata-rata bergerak (km/jam)"), col("max_speed", "Maks. (km/jam)"), col("coverage_percent", "Cakupan teramati (%)"), col("samples", "Titik valid"), col("rejected", "Tidak valid"), col("duplicates", "Duplikat"), col("gaps", "Jeda/loncatan durasi"), col("inferred_intervals", "Gerak dari koordinat"), col("status", "Data"), datecol("latest", "GPS terakhir (WIB)")}, Rows: details}}
	zone := time.FixedZone("WIB", 7*3600)
	start := out.Since.In(zone)
	day := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, zone)
	for !day.After(out.GeneratedAt.In(zone)) {
		key := day.Format("2006-01-02")
		out.Trend = append(out.Trend, map[string]any{"kind": "daily", "bucket": key, "total": rounded(daily[key])})
		day = day.AddDate(0, 0, 1)
	}
	for h, value := range hourly {
		out.Trend = append(out.Trend, map[string]any{"kind": "hourly", "bucket": fmt.Sprintf("%02d:00", h), "total": rounded(value / 3600)})
	}
	out.Notes = []string{"Periode berdasarkan waktu GPS (WIB). Sampel di luar periode tidak ditampilkan; gunakan 7/30 hari untuk riwayat lama. Tidak ada data berbeda dari kendaraan tidak bergerak. Kilometer merupakan estimasi garis lurus GPS, bukan odometer/jarak jalan.", "Jarak menjumlah perpindahan GPS valid dengan jeda ≤30 menit dan laju perpindahan ≤200 km/jam, sama dengan Tracking. Jarak tidak bergantung pada kecepatan yang dilaporkan perangkat.", "Durasi hanya untuk interval ≤5 menit. Bergerak bila kedua kecepatan >3 km/jam, atau perpindahan ≥20 m dengan laju GPS >3 km/jam. Berhenti bila kedua kecepatan dan laju GPS ≤3 km/jam. Interval lain tidak diklasifikasikan.", "Rata-rata bergerak dibobot durasi; bila kecepatan perangkat rendah tetapi koordinat menunjukkan gerak, dipakai laju perpindahan GPS. Kolom gerak dari koordinat menunjukkan jumlah interval tersebut. Maksimum adalah kecepatan sampel perangkat.", "Cakupan = durasi bergerak + berhenti teramati dibagi periode. Jeda >5 menit tidak berarti kendaraan berhenti; tanpa ignition tidak disimpulkan mesin mati/idle. Duplikat waktu dihitung sekali per perangkat.", "Jika beberapa perangkat melapor, dipilih perangkat dengan titik valid terbanyak per kendaraan (ID terkecil jika sama). Label kendaraan mengikuti asosiasi master saat ini."}
	return nil
}
