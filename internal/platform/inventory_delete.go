package platform

import (
	"database/sql"
	"errors"
	"net/http"
	"time"
)

func (s *Server) deleteInventory(w http.ResponseWriter, r *http.Request, u *Identity, table string, id int64) {
	if u.Role != "admin" {
		fail(w, 403, "permission_denied")
		return
	}
	if id <= 0 {
		fail(w, 400, "invalid_mutation_target")
		return
	}
	ctx := r.Context()
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		databaseError(w, err)
		return
	}
	defer tx.Rollback()
	var locked int64
	err = tx.QueryRowContext(ctx, "SELECT id FROM "+table+" WHERE id=? AND deleted_at IS NULL FOR UPDATE", id).Scan(&locked)
	if errors.Is(err, sql.ErrNoRows) {
		fail(w, 404, "record_not_found")
		return
	}
	if err != nil {
		databaseError(w, err)
		return
	}
	now := time.Now().UTC().Truncate(time.Millisecond)
	column := "vehicle_id"
	relations := []string{"vehicle_assignments", "ble_sensor_bindings", "geofence_vehicle_assignments"}
	update := "deleted_at=?,row_version=row_version+1"
	if table == "gps_devices" {
		column = "device_id"
		relations = relations[:2]
		update = "deleted_at=?,is_active=0"
	}
	for _, relation := range relations {
		var future bool
		if err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM "+relation+" WHERE "+column+"=? AND starts_at>? AND (ends_at IS NULL OR ends_at>starts_at))", id, now).Scan(&future); err != nil {
			databaseError(w, err)
			return
		}
		if future {
			fail(w, 409, "Masih ada penugasan terjadwal. Batalkan penugasan tersebut sebelum menghapus.")
			return
		}
		if _, err = tx.ExecContext(ctx, "UPDATE "+relation+" SET ends_at=? WHERE "+column+"=? AND starts_at<=? AND (ends_at IS NULL OR ends_at>?)", now, id, now, now); err != nil {
			databaseError(w, err)
			return
		}
	}
	if _, err = tx.ExecContext(ctx, "UPDATE "+table+" SET "+update+" WHERE id=?", now, id); err != nil {
		databaseError(w, err)
		return
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO audit_log(actor_id,action,entity,entity_id,request_id,occurred_at) VALUES(?,'delete',?,?,?,?)", u.ID, table, id, token(), now); err != nil {
		databaseError(w, err)
		return
	}
	if err = tx.Commit(); err != nil {
		databaseError(w, err)
		return
	}
	s.Notify()
	respond(w, 200, map[string]any{"data": map[string]any{"id": id}})
}
