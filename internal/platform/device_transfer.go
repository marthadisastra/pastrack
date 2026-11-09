package platform

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var errAssignmentChanged = errors.New("Relasi GPS berubah atau masih ada penugasan terjadwal. Muat ulang sebelum mengonfirmasi.")

type deviceTransfer struct {
	FromVehicleID  int64 `json:"from_vehicle_id"`
	TargetDeviceID int64 `json:"target_device_id"`
}

func transferGPS(ctx context.Context, tx *sql.Tx, device, target int64, expected deviceTransfer) (int64, string, error) {
	source, other := expected.FromVehicleID, expected.TargetDeviceID
	if source <= 0 || source == target || other < 0 || other == device {
		return 0, "", errAssignmentChanged
	}
	var locked int64
	for _, item := range []struct {
		table string
		id    int64
	}{{"vehicles", source}, {"gps_devices", other}} {
		if item.id == 0 {
			continue
		}
		if err := tx.QueryRowContext(ctx, "SELECT id FROM "+item.table+" WHERE id=? AND deleted_at IS NULL FOR UPDATE", item.id).Scan(&locked); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, "", errAssignmentChanged
			}
			return 0, "", err
		}
	}
	now := time.Now().UTC().Truncate(time.Millisecond)
	rows, err := tx.QueryContext(ctx, `SELECT id,device_id,vehicle_id,starts_at FROM vehicle_assignments WHERE (device_id IN (?,?) OR vehicle_id IN (?,?)) AND (ends_at IS NULL OR ends_at>?) FOR UPDATE`, device, other, source, target, now)
	if err != nil {
		return 0, "", err
	}
	var current, destination int64
	changed := false
	for rows.Next() {
		var id, d, v int64
		var starts time.Time
		if err = rows.Scan(&id, &d, &v, &starts); err != nil {
			rows.Close()
			return 0, "", err
		}
		switch {
		case !starts.Before(now):
			changed = true
		case d == device && v == source && current == 0:
			current = id
		case other > 0 && d == other && v == target && destination == 0:
			destination = id
		default:
			changed = true
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, "", err
	}
	if changed || current == 0 || (other > 0 && destination == 0) {
		return 0, "", errAssignmentChanged
	}
	if _, err = tx.ExecContext(ctx, "UPDATE vehicle_assignments SET ends_at=? WHERE id IN (?,?)", now, current, destination); err != nil {
		return 0, "", err
	}
	action := "move_gps"
	if other > 0 {
		action = "swap_gps"
		if _, err = tx.ExecContext(ctx, "INSERT INTO vehicle_assignments(device_id,vehicle_id,starts_at,provenance) VALUES(?,?,?,'device_management')", other, source, now); err != nil {
			return 0, "", err
		}
	}
	result, err := tx.ExecContext(ctx, "INSERT INTO vehicle_assignments(device_id,vehicle_id,starts_at,provenance) VALUES(?,?,?,'device_management')", device, target, now)
	if err != nil {
		return 0, "", err
	}
	id, err := result.LastInsertId()
	return id, action, err
}
