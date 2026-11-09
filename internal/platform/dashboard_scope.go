package platform

import (
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) dashboardScopes(w http.ResponseWriter, r *http.Request, u *Identity) {
	if u.Role != "admin" {
		fail(w, http.StatusForbidden, "admin_required")
		return
	}
	if r.Method == http.MethodGet {
		users, err := queryRows(r.Context(), s.DB, "SELECT id,username,full_name FROM users WHERE is_active=1 AND deleted_at IS NULL ORDER BY username")
		if err != nil {
			databaseError(w, err)
			return
		}
		vehicles, err := queryRows(r.Context(), s.DB, "SELECT id,plate_number,vehicle_name FROM vehicles WHERE deleted_at IS NULL ORDER BY plate_number")
		if err != nil {
			databaseError(w, err)
			return
		}
		scopes, err := queryRows(r.Context(), s.DB, "SELECT user_id,vehicle_id FROM dashboard_vehicle_scopes ORDER BY user_id,vehicle_id")
		if err != nil {
			databaseError(w, err)
			return
		}
		respond(w, 200, map[string]any{"data": map[string]any{"users": users, "vehicles": vehicles, "scopes": scopes}})
		return
	}
	if r.Method != http.MethodPut {
		fail(w, 405, "method_not_allowed")
		return
	}
	var input struct {
		UserID     int64   `json:"userId"`
		VehicleIDs []int64 `json:"vehicleIds"`
	}
	if err := body(w, r, &input); err != nil || input.UserID < 1 {
		fail(w, 400, "invalid_scope")
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		databaseError(w, err)
		return
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(r.Context(), "DELETE FROM dashboard_vehicle_scopes WHERE user_id=?", input.UserID); err != nil {
		databaseError(w, err)
		return
	}
	for _, vehicleID := range input.VehicleIDs {
		if vehicleID > 0 {
			if _, err = tx.ExecContext(r.Context(), "INSERT INTO dashboard_vehicle_scopes(user_id,vehicle_id) VALUES(?,?)", input.UserID, vehicleID); err != nil {
				databaseError(w, err)
				return
			}
		}
	}
	if err = tx.Commit(); err != nil {
		databaseError(w, err)
		return
	}
	s.Notify()
	respond(w, 200, map[string]any{"data": true})
}

func parseScopeID(path string) (int64, bool) {
	p := strings.Trim(path, "/")
	id, err := strconv.ParseInt(p, 10, 64)
	return id, err == nil && id > 0
}
