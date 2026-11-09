# Inventaris layar lokal dan kontrak route

Semua 21 pasangan screen.png/code.html ditemukan. Nama folder adalah identitas sumber; bukan klaim implementasi UI. Screenshot Fleet Overview diperiksa visual sebelumnya, seluruh HTML diperiksa secara struktural untuk inventaris ini; audit visual detail seluruh layar tetap wajib saat implementasi. Kontrak API berikut PROPOSED dengan prefix `/api/v1`, bukan endpoint yang sudah tersedia. Route frontend mempertahankan hash routing framework vanilla JS/Beego yang ada.

| Sumber Desain/ | Route target | API target (prefix /api/v1) | Database | Interaksi | Permission |
|---|---|---|---|---|---|
| fleet_overview | /#overview | GET /fleet/overview; /fleet/live; /fleet/statistics | vehicles, latest_positions, gps_device_status, latest_fuel_readings, alert_events | filter, pilih kendaraan, map, ekspor | fleet.read |
| vehicles | /#vehicles | GET/POST /vehicles; GET/PUT /vehicles/{id} | vehicles, vehicle_assets, vehicle_assignments | search, pagination, create/edit, assignment | vehicles.read/write |
| vehicles_telematics_overview_b_9421_uei | /#vehicles/{id}/overview | GET /vehicles/{id}; /vehicles/{id}/telemetry | vehicles, latest_positions, device_status_logs | detail, tab, rentang waktu | vehicles.read |
| vehicles_trip_history_b_9421_uei | /#vehicles/{id}/trips | GET /vehicles/{id}/trips; /trips/{id} | trips, trip_segments, trip_events | filter, pagination, playback | trips.read |
| vehicles_fuel_ble_cfs300b_b_9421_uei | /#vehicles/{id}/fuel | GET /vehicles/{id}/fuel; /vehicles/{id}/sensors | fuel_readings, sensor_telemetry, ble_sensor_status | grafik, sensor, freshness | fuel.read |
| vehicles_device_info_b_9421_uei | /#vehicles/{id}/device | GET /vehicles/{id}/devices; /devices/{id} | vehicle_assignments, gps_devices, gps_device_models, gps_device_status | detail, histori pemasangan | devices.read |
| vehicles_sensor_calibration_b_9421_uei | /#vehicles/{id}/calibration | GET/POST /sensors/{id}/calibrations; PUT /calibrations/{id}; POST /calibrations/{id}/approve | fuel_sensors, fuel_tank_profiles, fuel_calibrations, fuel_calibration_points | input titik, preview, validasi, approve | fuel.calibrate/approve |
| live_tracking_maps_telemetri_real_time | /#tracking | GET /fleet/live; /vehicles/{id}/telemetry; GET /realtime | latest_positions, gps_device_status, ble_sensor_status | map, pencarian, pilih marker, fit bounds | fleet.read |
| rute_playback_riwayat_telemetri_b_9421_uei | /#tracking/playback/{id} | GET /vehicles/{id}/positions; /vehicles/{id}/trips | gps_positions, trips, tracking_places | tanggal, play/pause, speed, scrub | trips.read |
| fuel_monitoring | /#fuel/monitoring | GET /fuel/overview; /fuel/readings | fuel_readings, latest_fuel_readings, fuel_tank_profiles | filter armada, grafik, sort/page | fuel.read |
| fuel_analytics_alert_center | /#fuel/analytics | GET /fuel/analytics; /fuel/events | fuel_consumption_summaries, fuel_events, fuel_readings | banding kendaraan, rentang, ekspor | fuel.read |
| device_data_health | /#health/gps | GET /devices/gps-quality; /devices/health | gps_quality_logs, gps_device_status, latest_positions | filter, diagnostics, freshness | devices.diagnostics |
| device_data_health_device_connectivity | /#health/connectivity | GET /devices/connectivity | gps_device_status, network_status_logs, ble_connection_logs | timeline, search/filter/page | devices.diagnostics |
| device_data_health_data_activity | /#health/activity | GET /devices/data-activity | device_raw_messages, protocol_parse_logs | interval, statistik ingest, error drilldown | devices.diagnostics |
| device_data_health_raw_protocol_logs | /#health/raw | GET /devices/raw-messages; /devices/raw-messages/{id} | device_raw_messages, protocol_parse_logs, protocol_parser_errors | search, cursor, payload inspect berizin | protocol.inspect |
| alert_center_emergency_triage | /#alerts | GET /alerts; /alerts/{id}; POST /alerts/{id}/acknowledge; /alerts/{id}/resolve | alert_events, device_events, fuel_events, event_acknowledgements | filter severity, detail, acknowledge/resolve | alerts.read/acknowledge |
| geofences_management | /#geofences | GET/POST /geofences; PUT/DELETE /geofences/{id}; PUT /geofences/{id}/vehicles | geofences, geofence_vehicle_assignments, geofence_events | circle/polygon editor, assignment, soft deactivate | geofences.read/write |
| reports_executive_telematics_analytics | /#reports | GET/POST /reports; GET /reports/{id}; GET /reports/{id}/download; GET/POST /report-schedules | report_jobs, report_schedules, trips, fuel_consumption_summaries | rentang/filter, export, jadwal | reports.read/create |
| device_management_inventaris_gps_gateway_iot | /#devices | GET/POST /devices; PUT /devices/{id}; GET/POST /vehicles/{id}/assignments | gps_devices, gps_device_models, vehicle_assignments, gps_device_configurations | inventory, edit, assignment; command hanya terverifikasi | devices.read/write |
| sensor_management_kalibrasi_telemetri_sensor | /#sensors | GET/POST /sensors; PUT /sensors/{id}; GET/POST /sensors/{id}/bindings; GET /sensors/{id}/telemetry | ble_sensors, ble_sensor_bindings, sensor_telemetry, fuel_sensors | inventory, binding, calibration | sensors.read/write |
| user_management_manajemen_pengguna_hak_akses | /#users | GET/POST /users; PUT /users/{id}; PUT /users/{id}/roles; POST /users/{id}/password-reset | users, roles, permissions, user_roles, sessions, audit_log | search, CRUD soft, roles, reset | users.manage |

## Layar pendukung di luar 21 ekspor

- Fuel Events: /#fuel/events → GET /api/v1/fuel/events; navigasi ditemukan pada HTML, tetapi tidak ada folder layar terpisah. Layout perlu disusun dari pola ekspor dan diperiksa, bukan diklaim ada screenshot khusus.
- Login /login, profile /#profile, inbox /#notifications, audit /#audit, settings bila dibutuhkan: pertahankan kemampuan sekarang, sesuaikan shell/tokens; tidak ada screenshot khusus yang telah diidentifikasi.
- DESIGN.md ada pada pastrack_fleet_telematics; folder stitch_pastrack_fleet_monitoring_dashboard dan ZIP tidak dihitung sebagai layar tambahan. Duplikasi arsip harus diperiksa sebelum ada penambahan inventory.

## Kompatibilitas route/API lama

- Saat ini Beego melayani shell / dan hash navigation; /api/tracking, /api/dashboards/:kind, /api/master/:entity, /api/fuel-calibration/:id, /api/users, /api/reports, /api/notifications, /api/realtime tetap tersedia melalui adapter sampai pengujian parity lulus.
- Alias hash lama vehicles-master→vehicles, devices-master→devices, sensors-master→sensors, messages→health/raw, positions→tracking/playback, dash-fuel→fuel/monitoring. ID/rentang yang belum ada harus memunculkan selection state, bukan memilih kendaraan palsu.
- SSE /api/realtime dan Unix-socket invalidation dipakai ulang; tidak menambahkan MQTT hanya karena label mockup. /api/v1/realtime adapter SSE, auth dan permission sama dengan query.

## Komponen, data state, akses

Shell/sidebar/header/search, KPI, StatusBadge, DataFreshness, LiveMap Leaflet, filter/date range, tabel cursor, chart units, raw viewer, timeline, empty/error/loading/stale dipakai ulang. Inter menggunakan tokens ekspor; screenshot per halaman mengatasi perbedaan warna antara YAML dan prose DESIGN.md. Jumlah 168, kendaraan, alert dan grafik pada mockup tidak menjadi seed produksi. Semua aksi diuji; aksi tanpa kemampuan backend tidak ditampilkan sebagai tombol aktif.

Viewer hanya read yang diberi izin, operator mengelola fleet/sensor dan acknowledge sesuai policy, admin mengelola akun/permission. Raw protocol dan command memerlukan permission khusus; periksa server-side untuk setiap endpoint, bukan menyembunyikan tombol saja. Filter vehicle membership belum berarti tenant support.
