package platform

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

var discoveryHeader = regexp.MustCompile(`^\+(?:RESP|BUFF|ACK):GT[A-Z0-9]{3},([0-9A-Fa-f]{6,32}),([0-9]{15}),([A-Za-z0-9-]{0,40}),`)

func observedDevice(payload string) (version, imei, model string) {
	payload = strings.TrimSpace(payload)
	if !strings.HasSuffix(payload, "$") {
		return "", "", ""
	}
	parts := discoveryHeader.FindStringSubmatch(payload)
	if parts == nil {
		return "", "", ""
	}
	return parts[1], parts[2], parts[3]
}

type discoveryQuery interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// A report may omit the model. Resolve it only from a catalog model explicitly
// reported by the same IMEI, including earlier acknowledgements.
func observedModel(ctx context.Context, db discoveryQuery, imei, model string) (int64, string, error) {
	var id int64
	if model != "" {
		err := db.QueryRowContext(ctx, "SELECT id FROM gps_device_models WHERE protocol_family='queclink' AND BINARY model=? ORDER BY id LIMIT 1", model).Scan(&id)
		return id, model, err
	}
	rows, err := db.QueryContext(ctx, `SELECT m.id,m.model,CONCAT(CONVERT(LEFT(r.payload,128) USING utf8mb4),RIGHT(TRIM(CONVERT(RIGHT(r.payload,16) USING utf8mb4)),1))
 FROM device_raw_messages r JOIN gps_device_models m ON m.protocol_family='queclink'
 WHERE r.received_at>=UTC_TIMESTAMP()-INTERVAL 30 DAY
 AND CONVERT(LEFT(r.payload,128) USING utf8mb4) LIKE CONCAT('+%:GT%,%,',?,',',m.model,',%')
 ORDER BY r.received_at DESC,r.id DESC,m.id LIMIT 50`, imei)
	if err != nil {
		return 0, "", err
	}
	defer rows.Close()
	for rows.Next() {
		var payload string
		if err := rows.Scan(&id, &model, &payload); err != nil {
			return 0, "", err
		}
		_, observedIMEI, observedName := observedDevice(payload)
		if observedIMEI == imei && observedName == model {
			return id, model, nil
		}
	}
	if err := rows.Err(); err != nil {
		return 0, "", err
	}
	return 0, "", sql.ErrNoRows
}

func (s *Server) deviceDiscovery(w http.ResponseWriter, r *http.Request, u *Identity, path string) bool {
	if path != "devices/discovered" {
		return false
	}
	if u.Role != "admin" && u.Role != "operator" {
		fail(w, 403, "permission_denied")
		return true
	}
	switch r.Method {
	case http.MethodGet:
		rows, e := queryRows(r.Context(), s.DB, `SELECT id, CONCAT(CONVERT(LEFT(payload,128) USING utf8mb4), RIGHT(TRIM(CONVERT(RIGHT(payload,16) USING utf8mb4)),1)) payload,received_at FROM device_raw_messages WHERE device_id IS NULL AND received_at>=UTC_TIMESTAMP()-INTERVAL 7 DAY ORDER BY received_at DESC,id DESC LIMIT 2000`)
		if e != nil {
			databaseError(w, e)
			return true
		}
		known, e := queryRows(r.Context(), s.DB, "SELECT imei FROM gps_devices")
		if e != nil {
			databaseError(w, e)
			return true
		}
		seen := make(map[string]bool, len(known))
		for _, row := range known {
			if imei, ok := row["imei"].(string); ok {
				seen[imei] = true
			}
		}
		candidates := []map[string]any{}
		for _, row := range rows {
			payload, _ := row["payload"].(string)
			version, imei, model := observedDevice(payload)
			if imei == "" || seen[imei] {
				continue
			}
			seen[imei] = true
			_, resolved, err := observedModel(r.Context(), s.DB, imei, model)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				databaseError(w, err)
				return true
			}
			candidates = append(candidates, map[string]any{"receipt_id": row["id"], "identifier": "••••" + imei[11:], "model": resolved, "registerable": err == nil, "protocol_version": version, "received_at": row["received_at"]})
		}
		respond(w, 200, map[string]any{"data": candidates})
	case http.MethodPost:
		if s.ReadOnly {
			fail(w, 403, "live_data_read_only")
			return true
		}
		var input struct {
			ReceiptID int64 `json:"receipt_id"`
		}
		if body(w, r, &input) != nil || input.ReceiptID <= 0 {
			fail(w, 400, "invalid_receipt_id")
			return true
		}
		tx, e := s.DB.BeginTx(r.Context(), nil)
		if e != nil {
			databaseError(w, e)
			return true
		}
		defer tx.Rollback()
		var payload string
		e = tx.QueryRowContext(r.Context(), `SELECT CONVERT(payload USING utf8mb4) FROM device_raw_messages WHERE id=? AND device_id IS NULL AND received_at>=UTC_TIMESTAMP()-INTERVAL 7 DAY`, input.ReceiptID).Scan(&payload)
		if errors.Is(e, sql.ErrNoRows) {
			fail(w, 404, "device_receipt_not_found")
			return true
		}
		if e != nil {
			databaseError(w, e)
			return true
		}
		version, imei, model := observedDevice(payload)
		if imei == "" {
			fail(w, 400, "invalid_device_header")
			return true
		}
		modelID, model, e := observedModel(r.Context(), tx, imei, model)
		if errors.Is(e, sql.ErrNoRows) {
			fail(w, 422, "device_model_not_supported")
			return true
		}
		if e != nil {
			databaseError(w, e)
			return true
		}
		result, e := tx.ExecContext(r.Context(), "INSERT INTO gps_devices(imei,model_id,protocol_version,is_active) VALUES(?,?,?,1)", imei, modelID, version)
		var duplicate *mysql.MySQLError
		if errors.As(e, &duplicate) && duplicate.Number == 1062 {
			fail(w, 409, "device_already_registered")
			return true
		}
		if e != nil {
			databaseError(w, e)
			return true
		}
		id, e := result.LastInsertId()
		if e != nil {
			databaseError(w, e)
			return true
		}
		provenance, _ := json.Marshal(map[string]any{"receipt_id": input.ReceiptID, "model": model, "protocol_version": version})
		if _, e = tx.ExecContext(r.Context(), "INSERT INTO audit_log(actor_id,action,entity,entity_id,after_data,request_id,occurred_at) VALUES(?,'register_observed','gps_devices',?,?,?,?)", u.ID, id, string(provenance), token(), time.Now().UTC()); e != nil {
			databaseError(w, e)
			return true
		}
		if e = tx.Commit(); e != nil {
			databaseError(w, e)
			return true
		}
		s.Notify()
		respond(w, 201, map[string]any{"data": map[string]any{"id": id}})
	default:
		fail(w, 405, "method_not_allowed")
	}
	return true
}
