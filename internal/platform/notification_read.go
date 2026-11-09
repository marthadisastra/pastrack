package platform

import (
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) notificationReadState(w http.ResponseWriter, r *http.Request, user *Identity, path string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		fail(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	if path == "notifications/read-all" {
		result, err := s.DB.ExecContext(r.Context(), "UPDATE notifications SET read_at=COALESCE(read_at,UTC_TIMESTAMP(3)) WHERE user_id=? AND read_at IS NULL", user.ID)
		if err != nil {
			databaseError(w, err)
			return
		}
		count, _ := result.RowsAffected()
		if count > 0 {
			s.Notify()
		}
		respond(w, http.StatusOK, map[string]any{"data": map[string]any{"updated": count}})
		return
	}
	parts := strings.Split(path, "/")
	if len(parts) != 3 || parts[0] != "notifications" || parts[2] != "read" {
		fail(w, http.StatusNotFound, "endpoint_not_found")
		return
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || id <= 0 {
		fail(w, http.StatusBadRequest, "invalid_notification")
		return
	}
	result, err := s.DB.ExecContext(r.Context(), "UPDATE notifications SET read_at=COALESCE(read_at,UTC_TIMESTAMP(3)) WHERE id=? AND user_id=?", id, user.ID)
	if err != nil {
		databaseError(w, err)
		return
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		var exists int
		if err = s.DB.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM notifications WHERE id=? AND user_id=?", id, user.ID).Scan(&exists); err != nil {
			databaseError(w, err)
			return
		}
		if exists == 0 {
			fail(w, http.StatusNotFound, "notification_not_found")
			return
		}
	}
	if count > 0 {
		s.Notify()
	}
	respond(w, http.StatusOK, map[string]any{"data": map[string]any{"id": strconv.FormatInt(id, 10), "read": true}})
}
