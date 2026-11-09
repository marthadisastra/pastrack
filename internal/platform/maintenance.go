package platform

import (
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) maintenance(w http.ResponseWriter, r *http.Request, u *Identity, path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 1 || parts[0] != "maintenance" || len(parts) > 2 {
		return false
	}
	if r.Method == http.MethodGet {
		return s.maintenanceList(w, r, parts)
	}
	if u.Role != "admin" && u.Role != "operator" {
		fail(w, 403, "permission_denied")
		return true
	}
	if r.Method == http.MethodPost && len(parts) == 1 {
		return s.maintenanceCreate(w, r, u)
	}
	if r.Method == http.MethodPatch && len(parts) == 2 {
		return s.maintenanceUpdate(w, r, u, parts[1])
	}
	if r.Method == http.MethodDelete && len(parts) == 2 {
		return s.maintenanceDelete(w, r, u, parts[1])
	}
	fail(w, 405, "method_not_allowed")
	return true
}

func (s *Server) maintenanceList(w http.ResponseWriter, r *http.Request, parts []string) bool {
	q := `SELECT m.id,m.vehicle_id,COALESCE(i.vehicle_code,CAST(m.vehicle_id AS CHAR)) vehicle_code,v.plate_number,v.vehicle_name,m.scope,m.title,m.description,m.trigger_value,m.trigger_unit,m.current_value,m.facility,m.bay,m.budget,m.scheduled_date,m.priority,m.status,m.row_version FROM maintenance_work_orders m JOIN vehicles v ON v.id=m.vehicle_id LEFT JOIN vehicle_inventory i ON i.vehicle_id=v.id WHERE v.deleted_at IS NULL`
	args := []any{}
	if len(parts) == 2 {
		id, e := strconv.ParseInt(parts[1], 10, 64)
		if e != nil {
			fail(w, 400, "invalid_id")
			return true
		}
		q += " AND m.id=?"
		args = append(args, id)
	}
	q += " ORDER BY m.id DESC"
	rows, e := queryRows(r.Context(), s.DB, q, args...)
	if e != nil {
		databaseError(w, e)
		return true
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, maintenanceRow(row))
	}
	if len(parts) == 2 {
		if len(out) == 0 {
			fail(w, 404, "record_not_found")
		} else {
			respond(w, 200, map[string]any{"data": out[0]})
		}
		return true
	}
	respond(w, 200, map[string]any{"data": out, "page": map[string]any{"has_more": false, "next_cursor": nil}})
	return true
}

func maintenanceRow(row map[string]any) map[string]any {
	return map[string]any{"id": "WO-" + strconv.FormatInt(numberInt(row["id"]), 10), "backend_id": row["id"], "vehicleId": row["vehicle_code"], "scope": row["scope"], "title": row["title"], "description": row["description"], "trigger": row["trigger_value"], "unit": row["trigger_unit"], "current": row["current_value"], "facility": row["facility"], "bay": row["bay"], "budget": row["budget"], "date": row["scheduled_date"], "priority": row["priority"], "status": row["status"], "row_version": row["row_version"]}
}
func numberInt(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case []byte:
		i, _ := strconv.ParseInt(string(n), 10, 64)
		return i
	case string:
		i, _ := strconv.ParseInt(n, 10, 64)
		return i
	}
	return 0
}

type maintenanceInput struct {
	VehicleID   string  `json:"vehicleId"`
	Scope       string  `json:"scope"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Trigger     float64 `json:"trigger"`
	Unit        string  `json:"unit"`
	Facility    string  `json:"facility"`
	Bay         string  `json:"bay"`
	Budget      float64 `json:"budget"`
	Date        string  `json:"date"`
	Priority    string  `json:"priority"`
	Status      string  `json:"status"`
}

func (s *Server) maintenanceCreate(w http.ResponseWriter, r *http.Request, u *Identity) bool {
	var in maintenanceInput
	if body(w, r, &in) != nil {
		fail(w, 400, "invalid_json")
		return true
	}
	vehicle, e := s.maintenanceVehicle(r, in.VehicleID)
	if e != nil || in.Title == "" || in.Trigger <= 0 || in.Date == "" {
		fail(w, 400, "invalid_work_order")
		return true
	}
	if in.Bay == "" {
		in.Bay = "Pending allocation"
	}
	if in.Priority == "" {
		in.Priority = "Normal"
	}
	if in.Status == "" {
		in.Status = "Scheduled"
	}
	res, e := s.DB.ExecContext(r.Context(), `INSERT INTO maintenance_work_orders(vehicle_id,scope,title,description,trigger_value,trigger_unit,current_value,facility,bay,budget,scheduled_date,priority,status,created_by) VALUES(?,?,?,?,?,?,0,?,?,?,?,?,?,?)`, vehicle, in.Scope, in.Title, in.Description, in.Trigger, in.Unit, in.Facility, in.Bay, in.Budget, in.Date, in.Priority, in.Status, u.ID)
	if e != nil {
		databaseError(w, e)
		return true
	}
	id, _ := res.LastInsertId()
	s.Notify()
	respond(w, 201, map[string]any{"data": map[string]any{"id": "WO-" + strconv.FormatInt(id, 10)}})
	return true
}
func (s *Server) maintenanceVehicle(r *http.Request, value string) (int64, error) {
	id, e := strconv.ParseInt(value, 10, 64)
	if e == nil {
		return id, nil
	}
	var out int64
	e = s.DB.QueryRowContext(r.Context(), "SELECT v.id FROM vehicles v LEFT JOIN vehicle_inventory i ON i.vehicle_id=v.id WHERE v.deleted_at IS NULL AND (i.vehicle_code=? OR CAST(v.id AS CHAR)=?) LIMIT 1", value, value).Scan(&out)
	return out, e
}
func (s *Server) maintenanceUpdate(w http.ResponseWriter, r *http.Request, u *Identity, raw string) bool {
	id, e := strconv.ParseInt(strings.TrimPrefix(raw, "WO-"), 10, 64)
	if e != nil {
		fail(w, 400, "invalid_id")
		return true
	}
	var in maintenanceInput
	if body(w, r, &in) != nil {
		fail(w, 400, "invalid_json")
		return true
	}
	sets := []string{}
	args := []any{}
	if in.Status != "" {
		sets = append(sets, "status=?")
		args = append(args, in.Status)
	}
	if in.Title != "" {
		sets = append(sets, "title=?")
		args = append(args, in.Title)
	}
	if in.Description != "" {
		sets = append(sets, "description=?")
		args = append(args, in.Description)
	}
	if in.Scope != "" {
		sets = append(sets, "scope=?")
		args = append(args, in.Scope)
	}
	if in.Unit != "" {
		sets = append(sets, "trigger_unit=?")
		args = append(args, in.Unit)
	}
	if in.Trigger > 0 {
		sets = append(sets, "trigger_value=?")
		args = append(args, in.Trigger)
	}
	if in.Facility != "" {
		sets = append(sets, "facility=?")
		args = append(args, in.Facility)
	}
	if in.Bay != "" {
		sets = append(sets, "bay=?")
		args = append(args, in.Bay)
	}
	if in.Budget >= 0 {
		sets = append(sets, "budget=?")
		args = append(args, in.Budget)
	}
	if in.Date != "" {
		sets = append(sets, "scheduled_date=?")
		args = append(args, in.Date)
	}
	if in.Priority != "" {
		sets = append(sets, "priority=?")
		args = append(args, in.Priority)
	}
	if len(sets) == 0 {
		fail(w, 400, "no_changes")
		return true
	}
	args = append(args, id)
	res, e := s.DB.ExecContext(r.Context(), "UPDATE maintenance_work_orders SET "+strings.Join(sets, ",")+",row_version=row_version+1 WHERE id=?", args...)
	if e != nil {
		databaseError(w, e)
		return true
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		fail(w, 404, "record_not_found")
		return true
	}
	s.Notify()
	respond(w, 200, map[string]any{"data": map[string]any{"id": raw}})
	return true
}

func (s *Server) maintenanceDelete(w http.ResponseWriter, r *http.Request, u *Identity, raw string) bool {
	id, e := strconv.ParseInt(strings.TrimPrefix(raw, "WO-"), 10, 64)
	if e != nil {
		fail(w, 400, "invalid_id")
		return true
	}
	res, e := s.DB.ExecContext(r.Context(), "DELETE FROM maintenance_work_orders WHERE id=?", id)
	if e != nil {
		databaseError(w, e)
		return true
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		fail(w, 404, "record_not_found")
		return true
	}
	s.Notify()
	respond(w, 200, map[string]any{"data": map[string]any{"id": raw, "deleted": true}})
	return true
}
