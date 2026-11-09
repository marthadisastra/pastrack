package platform

import (
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strings"
)

func (s *Server) userMutation(w http.ResponseWriter, r *http.Request, u *Identity, path string) bool {
	if r.Method == "GET" {
		return false
	}
	parts := strings.Split(path, "/")
	if parts[0] != "users" {
		return false
	}
	if u.Role != "admin" {
		fail(w, 403, "permission_denied")
		return true
	}
	if len(parts) > 2 {
		fail(w, 404, "endpoint_not_found")
		return true
	}
	var id int64
	var e error
	if len(parts) == 2 {
		id, e = parseID(parts[1])
		if e != nil {
			fail(w, 400, "invalid_id")
			return true
		}
	}
	if r.Method == "POST" && id != 0 || r.Method == "PUT" && id == 0 || (r.Method != "POST" && r.Method != "PUT") {
		fail(w, 405, "method_not_allowed")
		return true
	}
	var v struct {
		Username string  `json:"username"`
		Name     string  `json:"full_name"`
		Email    *string `json:"email"`
		Password string  `json:"password"`
		Role     string  `json:"role"`
		Active   *bool   `json:"is_active"`
	}
	if body(w, r, &v) != nil || len(v.Username) < 3 || len(v.Username) > 64 || len(v.Name) == 0 || len(v.Name) > 100 || (v.Role != "admin" && v.Role != "operator" && v.Role != "viewer") || v.Email != nil && len(*v.Email) > 190 {
		fail(w, 400, "invalid_user")
		return true
	}
	if id == 0 || v.Password != "" {
		if len(v.Password) < 12 || len(v.Password) > 72 {
			fail(w, 400, "password_requires_12_to_72_bytes")
			return true
		}
	}
	var password []byte
	if v.Password != "" {
		password, e = bcrypt.GenerateFromPassword([]byte(v.Password), 12)
		if e != nil {
			fail(w, 400, "invalid_password")
			return true
		}
	}
	active := true
	if v.Active != nil {
		active = *v.Active
	}
	tx, e := s.DB.BeginTx(r.Context(), nil)
	if e != nil {
		databaseError(w, e)
		return true
	}
	defer tx.Rollback()
	// Lock the admin role to serialize changes that could remove the last admin.
	var roleID int64
	if e = tx.QueryRowContext(r.Context(), "SELECT id FROM roles WHERE code='admin' FOR UPDATE").Scan(&roleID); e != nil {
		databaseError(w, e)
		return true
	}
	if id > 0 {
		var exists int
		if e = tx.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM users WHERE id=?", id).Scan(&exists); e != nil || exists == 0 {
			fail(w, 404, "user_not_found")
			return true
		}
		if !active || v.Role != "admin" {
			var remaining int
			e = tx.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM users u JOIN user_roles ur ON ur.user_id=u.id WHERE ur.role_id=? AND u.id<>? AND u.is_active=1 AND u.deleted_at IS NULL", roleID, id).Scan(&remaining)
			if e != nil {
				databaseError(w, e)
				return true
			}
			if remaining == 0 {
				fail(w, 409, "last_admin_required")
				return true
			}
		}
		_, e = tx.ExecContext(r.Context(), "UPDATE users SET username=?,full_name=?,email=?,is_active=?,row_version=row_version+1,auth_version=auth_version+1 WHERE id=?", v.Username, v.Name, v.Email, active, id)
		if e == nil && len(password) > 0 {
			_, e = tx.ExecContext(r.Context(), "UPDATE users SET password_hash=?,password_changed_at=UTC_TIMESTAMP(3) WHERE id=?", string(password), id)
		}
	} else {
		result, err := tx.ExecContext(r.Context(), "INSERT INTO users(username,full_name,email,password_hash,password_algo,auth_version,is_active,row_version) VALUES(?,?,?,?,'bcrypt',1,?,1)", v.Username, v.Name, v.Email, string(password), active)
		e = err
		if e == nil {
			id, _ = result.LastInsertId()
		}
	}
	if e != nil {
		fail(w, 422, "invalid_or_conflicting_user")
		return true
	}
	// Role replacement is a reversible access change, never deletion of telemetry.
	if _, e = tx.ExecContext(r.Context(), "DELETE FROM user_roles WHERE user_id=?", id); e != nil {
		databaseError(w, e)
		return true
	}
	if _, e = tx.ExecContext(r.Context(), "INSERT INTO user_roles(user_id,role_id) SELECT ?,id FROM roles WHERE code=?", id, v.Role); e != nil {
		databaseError(w, e)
		return true
	}
	if _, e = tx.ExecContext(r.Context(), "INSERT INTO audit_log(actor_id,action,entity,entity_id,request_id,occurred_at) VALUES(?,'user_update','users',?,?,UTC_TIMESTAMP(3))", u.ID, id, token()); e != nil {
		databaseError(w, e)
		return true
	}
	if e = tx.Commit(); e != nil {
		databaseError(w, e)
		return true
	}
	s.Notify()
	status := 200
	if r.Method == "POST" {
		status = 201
	}
	respond(w, status, map[string]any{"data": map[string]any{"id": id}})
	return true
}
