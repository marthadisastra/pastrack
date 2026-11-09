package platform

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Server) reportOperations(w http.ResponseWriter, r *http.Request, u *Identity, path string) bool {
	parts := strings.Split(path, "/")
	if len(parts) == 3 && parts[0] == "reports" && parts[2] == "download" {
		if r.Method != "GET" {
			fail(w, 405, "method_not_allowed")
			return true
		}
		id, e := parseID(parts[1])
		if e != nil {
			fail(w, 400, "invalid_id")
			return true
		}
		var owner int64
		var state string
		var filename *string
		e = s.DB.QueryRowContext(r.Context(), "SELECT requested_by,state,artifact_path FROM report_jobs WHERE id=? AND (expires_at IS NULL OR expires_at>UTC_TIMESTAMP(3))", id).Scan(&owner, &state, &filename)
		if e != nil {
			fail(w, 404, "report_not_found")
			return true
		}
		if owner != u.ID && u.Role != "admin" {
			fail(w, 403, "permission_denied")
			return true
		}
		if state != "completed" || filename == nil {
			fail(w, 409, "report_not_ready")
			return true
		}
		if filepath.Base(*filename) != *filename {
			fail(w, 500, "invalid_artifact")
			return true
		}
		extension := filepath.Ext(*filename)
		contentType := map[string]string{".csv": "text/csv; charset=utf-8", ".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", ".pdf": "application/pdf"}[extension]
		if contentType == "" {
			fail(w, 500, "unsupported_artifact")
			return true
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="pastrack-report-%d%s"`, id, extension))
		http.ServeFile(w, r, filepath.Join(s.reportDir(), *filename))
		return true
	}
	if path != "reports" || r.Method == "GET" {
		return false
	}
	if r.Method != "POST" {
		fail(w, 405, "method_not_allowed")
		return true
	}
	if u.Role == "viewer" {
		fail(w, 403, "permission_denied")
		return true
	}
	var input struct {
		Type      string    `json:"report_type"`
		Format    string    `json:"format"`
		From      time.Time `json:"from"`
		To        time.Time `json:"to"`
		VehicleID *int64    `json:"vehicle_id"`
	}
	if body(w, r, &input) != nil || input.From.IsZero() || !input.To.After(input.From) || input.To.Sub(input.From) > 31*24*time.Hour {
		fail(w, 400, "invalid_report_range")
		return true
	}
	if input.Format == "" {
		input.Format = "csv"
	}
	if input.Format != "csv" && input.Format != "xlsx" && input.Format != "pdf" {
		fail(w, 400, "unsupported_report_format")
		return true
	}
	target := map[string]string{"positions": "gps/positions", "trips": "trips", "fuel": "fuel/readings", "events": "events", "crashes": "crashes/history"}[input.Type]
	spec, ok := resources[target]
	if !ok {
		fail(w, 400, "unsupported_report_type")
		return true
	}
	select {
	case s.reportSlots <- struct{}{}:
	default:
		fail(w, 429, "report_capacity_reached")
		return true
	}
	filter, _ := json.Marshal(input)
	result, e := s.DB.ExecContext(r.Context(), "INSERT INTO report_jobs(requested_by,report_type,filters,state,expires_at) VALUES(?,?,?,'queued',?)", u.ID, input.Type, string(filter), time.Now().UTC().Add(24*time.Hour))
	if e != nil {
		<-s.reportSlots
		databaseError(w, e)
		return true
	}
	id, _ := result.LastInsertId()
	go func() {
		defer func() { <-s.reportSlots }()
		s.writeReport(id, spec, input.From, input.To, input.VehicleID, input.Format)
	}()
	respond(w, 202, map[string]any{"data": map[string]any{"id": id, "state": "queued"}})
	return true
}
func (s *Server) reportDir() string {
	if value := os.Getenv("LOCAL_REPORT_DIR"); value != "" {
		return value
	}
	return ".local/reports"
}
func (s *Server) writeReport(id int64, spec resource, from, to time.Time, vehicle *int64, formats ...string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	state := "failed"
	var artifact any
	defer func() {
		end, c := context.WithTimeout(context.Background(), 5*time.Second)
		defer c()
		s.DB.ExecContext(end, "UPDATE report_jobs SET state=?,artifact_path=? WHERE id=?", state, artifact, id)
		s.Notify()
	}()
	if e := os.MkdirAll(s.reportDir(), 0700); e != nil {
		return
	}
	name := token() + ".csv"
	file, e := os.OpenFile(filepath.Join(s.reportDir(), name+".partial"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return
	}
	defer file.Close()
	query := "SELECT " + spec.Select + " FROM " + spec.Table + " a" + spec.Join + " WHERE " + spec.Time + ">=? AND " + spec.Time + "<=?"
	args := []any{from.UTC(), to.UTC()}
	if vehicle != nil && spec.Vehicle != "" {
		query += " AND " + spec.Vehicle + "=?"
		args = append(args, *vehicle)
	}
	query += " ORDER BY a.id LIMIT 50001"
	rows, e := s.DB.QueryContext(ctx, query, args...)
	if e != nil {
		return
	}
	defer rows.Close()
	columns, e := rows.Columns()
	if e != nil {
		return
	}
	writer := csv.NewWriter(file)
	writer.Write(columns)
	count := 0
	for rows.Next() {
		count++
		if count > 50000 {
			return
		}
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if e = rows.Scan(pointers...); e != nil {
			return
		}
		line := make([]string, len(values))
		for i, v := range values {
			switch value := v.(type) {
			case nil:
				line[i] = ""
			case []byte:
				line[i] = string(value)
			case time.Time:
				line[i] = value.UTC().Format(time.RFC3339Nano)
			default:
				line[i] = fmt.Sprint(value)
			}
			if strings.ContainsAny(firstChar(line[i]), "=+-@\t\r") {
				line[i] = "'" + line[i]
			}
		}
		if e = writer.Write(line); e != nil {
			return
		}
	}
	if rows.Err() != nil {
		return
	}
	writer.Flush()
	if writer.Error() != nil {
		return
	}
	if file.Sync() != nil {
		return
	}
	if file.Close() != nil {
		return
	}
	if os.Rename(filepath.Join(s.reportDir(), name+".partial"), filepath.Join(s.reportDir(), name)) != nil {
		return
	}
	format := "csv"
	if len(formats) > 0 {
		format = formats[0]
	}
	if format != "csv" {
		converted := strings.TrimSuffix(name, ".csv") + "." + format
		if convertReport(filepath.Join(s.reportDir(), name), filepath.Join(s.reportDir(), converted), format) != nil {
			return
		}
		name = converted
	}
	artifact = name
	state = "completed"
}
func firstChar(s string) string {
	if s == "" {
		return ""
	}
	return s[:1]
}
