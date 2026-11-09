package platform

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

func (s *Server) authLimit(bucket string, limit int, window time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for key, times := range s.attempts {
		if len(times) == 0 || now.Sub(times[len(times)-1]) > time.Hour {
			delete(s.attempts, key)
		}
	}
	prior := s.attempts[bucket]
	recent := make([]time.Time, 0, len(prior)+1)
	for _, at := range prior {
		if now.Sub(at) < window {
			recent = append(recent, at)
		}
	}
	if len(recent) >= limit || len(s.attempts) >= 4096 {
		return false
	}
	s.attempts[bucket] = append(recent, now)
	return true
}
func authIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	if peer := net.ParseIP(host); peer != nil && peer.IsLoopback() {
		if forwarded := net.ParseIP(r.Header.Get("X-Real-IP")); forwarded != nil {
			return forwarded.String()
		}
	}
	return host
}
func authMutation(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		fail(w, 405, "method_not_allowed")
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" && origin != "http://"+r.Host && origin != "https://"+r.Host {
		fail(w, 403, "origin_rejected")
		return false
	}
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		fail(w, 403, "origin_rejected")
		return false
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		fail(w, 415, "json_required")
		return false
	}
	return true
}
func secureCookie(r *http.Request) bool { return r.TLS != nil || os.Getenv("COOKIE_SECURE") == "true" }
func clearSession(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "pastrack_v2_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: secureCookie(r), SameSite: http.SameSiteStrictMode})
}
func (s *Server) authRoute(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Referrer-Policy", "no-referrer")
	switch r.URL.Path {
	case "/api/v1/auth/smtp", "/api/v1/auth/smtp/test":
		s.authSMTP(w, r)
	case "/api/v1/auth/login":
		s.authLogin(w, r)
	case "/api/v1/auth/forgot-password":
		s.authForgot(w, r)
	case "/api/v1/auth/reset-password":
		s.authReset(w, r)
	case "/api/v1/auth/me":
		if r.Method != "GET" {
			fail(w, 405, "method_not_allowed")
			return
		}
		user := s.identity(r)
		if user == nil {
			clearSession(w, r)
			fail(w, 401, "authentication_required")
			return
		}
		respond(w, 200, map[string]any{"data": user})
	case "/api/v1/auth/logout":
		if !authMutation(w, r) {
			return
		}
		if cookie, err := r.Cookie("pastrack_v2_session"); err == nil {
			if _, err = s.DB.ExecContext(r.Context(), "DELETE FROM sessions WHERE token_hash=?", hash(cookie.Value)); err != nil {
				fail(w, 503, "database_unavailable")
				return
			}
		}
		clearSession(w, r)
		s.Notify()
		respond(w, 200, map[string]any{"data": true})
	default:
		fail(w, 404, "not_found")
	}
}
func (s *Server) authLogin(w http.ResponseWriter, r *http.Request) {
	if !authMutation(w, r) {
		return
	}
	if !s.authLimit("login:"+authIP(r), 15, 10*time.Minute) {
		w.Header().Set("Retry-After", "600")
		fail(w, 429, "too_many_attempts")
		return
	}
	var input struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Remember bool   `json:"remember"`
	}
	if body(w, r, &input) != nil {
		fail(w, 400, "invalid_login")
		return
	}
	login := strings.TrimSpace(input.Email)
	if login == "" {
		login = strings.TrimSpace(input.Username)
	}
	if len(login) == 0 || len(login) > 190 || len(input.Password) == 0 || len(input.Password) > 72 {
		fail(w, 400, "invalid_login")
		return
	}
	var id, version int64
	var digest string
	// Ambiguous email/username matches never select an arbitrary account.
	rows, err := s.DB.QueryContext(r.Context(), "SELECT id,password_hash,auth_version FROM users WHERE (username=? OR email=?) AND is_active=1 AND deleted_at IS NULL LIMIT 2", login, login)
	count := 0
	if err == nil {
		for rows.Next() {
			count++
			if e := rows.Scan(&id, &digest, &version); e != nil {
				err = e
				break
			}
		}
		if e := rows.Err(); e != nil {
			err = e
		}
		rows.Close()
	}
	if err != nil {
		fail(w, 503, "database_unavailable")
		return
	}
	if count != 1 {
		digest = "$2a$12$8nCf7PE8REyqT9q9gCsj6uEOYl5Jt6A3rXbD.1xUEJ8AlhP8pz4Fa"
	}
	match := bcrypt.CompareHashAndPassword([]byte(digest), []byte(input.Password))
	if count != 1 || match != nil {
		fail(w, 401, "invalid_credentials")
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		fail(w, 503, "database_unavailable")
		return
	}
	defer tx.Rollback()
	// Serialize login against concurrent password reset; never issue an old-version session.
	var currentVersion int64
	if err = tx.QueryRowContext(r.Context(), "SELECT auth_version FROM users WHERE id=? AND is_active=1 AND deleted_at IS NULL FOR UPDATE", id).Scan(&currentVersion); err != nil || currentVersion != version {
		fail(w, 401, "invalid_credentials")
		return
	}
	raw := token()
	duration := 12 * time.Hour
	if input.Remember {
		duration = 30 * 24 * time.Hour
	}
	if cookie, e := r.Cookie("pastrack_v2_session"); e == nil {
		if _, err = tx.ExecContext(r.Context(), "DELETE FROM sessions WHERE token_hash=?", hash(cookie.Value)); err != nil {
			fail(w, 503, "database_unavailable")
			return
		}
	}
	if _, err = tx.ExecContext(r.Context(), "INSERT INTO sessions(user_id,token_hash,auth_version,expires_at) VALUES(?,?,?,?)", id, hash(raw), version, time.Now().UTC().Add(duration)); err != nil {
		fail(w, 503, "database_unavailable")
		return
	}
	if _, err = tx.ExecContext(r.Context(), "UPDATE users SET last_login_at=UTC_TIMESTAMP(3) WHERE id=?", id); err != nil {
		fail(w, 503, "database_unavailable")
		return
	}
	if err = tx.Commit(); err != nil {
		fail(w, 503, "database_unavailable")
		return
	}
	cookie := &http.Cookie{Name: "pastrack_v2_session", Value: raw, Path: "/", HttpOnly: true, Secure: secureCookie(r), SameSite: http.SameSiteStrictMode}
	if input.Remember {
		cookie.MaxAge = int(duration.Seconds())
		cookie.Expires = time.Now().Add(duration)
	}
	http.SetCookie(w, cookie)
	// Keep legacy login response compatible; the new frontend obtains the user via /auth/me.
	respond(w, 200, map[string]any{"data": true})
}
func validNewPassword(password string) bool {
	if len(password) < 12 || len(password) > 72 {
		return false
	}
	var lower, upper, number, symbol bool
	for _, c := range password {
		lower = lower || unicode.IsLower(c)
		upper = upper || unicode.IsUpper(c)
		number = number || unicode.IsDigit(c)
		symbol = symbol || (!unicode.IsLetter(c) && !unicode.IsDigit(c) && !unicode.IsSpace(c))
	}
	return lower && upper && number && symbol
}
func (s *Server) authForgot(w http.ResponseWriter, r *http.Request) {
	if !authMutation(w, r) {
		return
	}
	if !s.authLimit("forgot-ip:"+authIP(r), 10, 15*time.Minute) {
		w.Header().Set("Retry-After", "900")
		fail(w, 429, "too_many_attempts")
		return
	}
	var input struct {
		Email string `json:"email"`
	}
	if body(w, r, &input) != nil || len(input.Email) > 190 {
		fail(w, 400, "invalid_email")
		return
	}
	email := strings.TrimSpace(input.Email)
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		fail(w, 400, "invalid_email")
		return
	}
	sender, mailErr := s.resetSender(r.Context())
	if mailErr != nil || sender == nil || len(s.ResetMailKey) != 32 || s.PublicAppURL == "" {
		fail(w, 503, "password_recovery_unavailable")
		return
	}
	generic := func() {
		respond(w, 202, map[string]any{"data": map[string]string{"message": "If an active account exists for this email, a reset link will be sent."}})
	}
	if !s.authLimit("forgot-account:"+hash(strings.ToLower(email)), 3, 15*time.Minute) {
		generic()
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		fail(w, 503, "database_unavailable")
		return
	}
	defer tx.Rollback()
	var id, version int64
	var recipient string
	// Duplicate recovery addresses require administrator resolution, never reset several identities.
	rows, err := tx.QueryContext(r.Context(), "SELECT id,auth_version,email FROM users WHERE email=? AND is_active=1 AND deleted_at IS NULL LIMIT 2", email)
	if err != nil {
		fail(w, 503, "database_unavailable")
		return
	}
	count := 0
	for rows.Next() {
		count++
		err = rows.Scan(&id, &version, &recipient)
		if err != nil {
			break
		}
	}
	if e := rows.Err(); e != nil {
		err = e
	}
	rows.Close()
	if err != nil {
		fail(w, 503, "database_unavailable")
		return
	}
	if count != 1 {
		generic()
		return
	}
	raw := token()
	link := strings.TrimRight(s.PublicAppURL, "/") + "/#/reset-password?token=" + url.QueryEscape(raw)
	payload, err := s.encryptReset(recipient, link)
	if err != nil {
		fail(w, 503, "password_recovery_unavailable")
		return
	}
	// Do not invalidate previously delivered links on request: a requester must not lock out the owner.
	_, err = tx.ExecContext(r.Context(), "INSERT INTO auth_password_resets(user_id,token_hash,auth_version,expires_at,delivery_payload) VALUES(?,?,?,UTC_TIMESTAMP(3)+INTERVAL 15 MINUTE,?)", id, hash(raw), version, payload)
	if err != nil {
		fail(w, 503, "database_unavailable")
		return
	}
	if err = tx.Commit(); err != nil {
		fail(w, 503, "database_unavailable")
		return
	}
	generic()
}
func (s *Server) authReset(w http.ResponseWriter, r *http.Request) {
	if !authMutation(w, r) {
		return
	}
	if !s.authLimit("reset:"+authIP(r), 20, 15*time.Minute) {
		fail(w, 429, "too_many_attempts")
		return
	}
	var input struct {
		Token             string `json:"token"`
		Password          string `json:"password"`
		TerminateSessions bool   `json:"terminateSessions"`
	}
	if body(w, r, &input) != nil || len(input.Token) != 64 {
		fail(w, 400, "invalid_or_expired_reset_token")
		return
	}
	if _, err := hex.DecodeString(input.Token); err != nil {
		fail(w, 400, "invalid_or_expired_reset_token")
		return
	}
	if !validNewPassword(input.Password) {
		fail(w, 400, "password_policy_not_met")
		return
	}
	// Compute the expensive hash before acquiring row locks.
	digest, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
	if err != nil {
		fail(w, 400, "password_policy_not_met")
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		fail(w, 503, "database_unavailable")
		return
	}
	defer tx.Rollback()
	var userID, version int64
	// Initial lookup does not lock reset rows: all reset transactions lock user -> reset in the same order.
	err = tx.QueryRowContext(r.Context(), "SELECT user_id,auth_version FROM auth_password_resets WHERE token_hash=? AND used_at IS NULL AND expires_at>UTC_TIMESTAMP(3)", hash(input.Token)).Scan(&userID, &version)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fail(w, 400, "invalid_or_expired_reset_token")
		} else {
			fail(w, 503, "database_unavailable")
		}
		return
	}
	var current int64
	if err = tx.QueryRowContext(r.Context(), "SELECT auth_version FROM users WHERE id=? AND is_active=1 AND deleted_at IS NULL FOR UPDATE", userID).Scan(&current); err != nil || current != version {
		fail(w, 400, "invalid_or_expired_reset_token")
		return
	}
	result, err := tx.ExecContext(r.Context(), "UPDATE auth_password_resets SET used_at=UTC_TIMESTAMP(3),delivery_payload=NULL WHERE token_hash=? AND used_at IS NULL AND expires_at>UTC_TIMESTAMP(3)", hash(input.Token))
	if err != nil {
		fail(w, 503, "database_unavailable")
		return
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		fail(w, 400, "invalid_or_expired_reset_token")
		return
	}
	if _, err = tx.ExecContext(r.Context(), "UPDATE users SET password_hash=?,password_algo='bcrypt',auth_version=auth_version+1,password_changed_at=UTC_TIMESTAMP(3),row_version=row_version+1 WHERE id=?", string(digest), userID); err != nil {
		fail(w, 503, "database_unavailable")
		return
	}
	if _, err = tx.ExecContext(r.Context(), "DELETE FROM sessions WHERE user_id=?", userID); err != nil {
		fail(w, 503, "database_unavailable")
		return
	}
	if _, err = tx.ExecContext(r.Context(), "UPDATE auth_password_resets SET used_at=COALESCE(used_at,UTC_TIMESTAMP(3)),delivery_payload=NULL WHERE user_id=?", userID); err != nil {
		fail(w, 503, "database_unavailable")
		return
	}
	if err = tx.Commit(); err != nil {
		fail(w, 503, "database_unavailable")
		return
	}
	s.Notify()
	clearSession(w, r)
	respond(w, 200, map[string]any{"data": true})
}
func (s *Server) encryptReset(email, link string) ([]byte, error) {
	block, err := aes.NewCipher(s.ResetMailKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	payload, _ := json.Marshal([]string{email, link})
	return gcm.Seal(nonce, nonce, payload, nil), nil
}
func (s *Server) decryptReset(payload []byte) (string, string, error) {
	block, err := aes.NewCipher(s.ResetMailKey)
	if err != nil {
		return "", "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(payload) < gcm.NonceSize() {
		return "", "", fmt.Errorf("invalid delivery payload")
	}
	plain, err := gcm.Open(nil, payload[:gcm.NonceSize()], payload[gcm.NonceSize():], nil)
	if err != nil {
		return "", "", err
	}
	var fields []string
	if json.Unmarshal(plain, &fields) != nil || len(fields) != 2 {
		return "", "", fmt.Errorf("invalid delivery fields")
	}
	return fields[0], fields[1], nil
}

// DeliverResetEmails claims a short lease; raw links are encrypted at rest and removed after delivery.
func (s *Server) DeliverResetEmails(ctx context.Context) error {
	sender, senderErr := s.resetSender(ctx)
	if senderErr != nil {
		return senderErr
	}
	if sender == nil {
		return nil
	}
	if _, err := s.DB.ExecContext(ctx, "UPDATE auth_password_resets SET delivery_payload=NULL,delivery_state='expired' WHERE expires_at<=UTC_TIMESTAMP(3) AND delivery_payload IS NOT NULL"); err != nil {
		return err
	}
	for i := 0; i < 10; i++ {
		tx, err := s.DB.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		var id int64
		var payload []byte
		err = tx.QueryRowContext(ctx, "SELECT id,delivery_payload FROM auth_password_resets WHERE used_at IS NULL AND expires_at>UTC_TIMESTAMP(3) AND delivery_payload IS NOT NULL AND delivery_state IN ('queued','sending') AND delivery_attempts<5 AND next_attempt_at<=UTC_TIMESTAMP(3) AND (locked_until IS NULL OR locked_until<UTC_TIMESTAMP(3)) ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED").Scan(&id, &payload)
		if errors.Is(err, sql.ErrNoRows) {
			tx.Rollback()
			return nil
		}
		if err != nil {
			tx.Rollback()
			return err
		}
		_, err = tx.ExecContext(ctx, "UPDATE auth_password_resets SET delivery_state='sending',delivery_attempts=delivery_attempts+1,locked_until=UTC_TIMESTAMP(3)+INTERVAL 1 MINUTE WHERE id=?", id)
		if err != nil {
			tx.Rollback()
			return err
		}
		if err = tx.Commit(); err != nil {
			return err
		}
		recipient, link, err := s.decryptReset(payload)
		if err == nil {
			err = sender(ctx, recipient, link)
		}
		if err == nil {
			_, err = s.DB.ExecContext(ctx, "UPDATE auth_password_resets SET delivery_state='sent',delivery_payload=NULL,locked_until=NULL WHERE id=?", id)
		} else {
			_, err = s.DB.ExecContext(ctx, "UPDATE auth_password_resets SET delivery_state=IF(delivery_attempts>=5,'failed','queued'),delivery_payload=IF(delivery_attempts>=5,NULL,delivery_payload),locked_until=NULL,next_attempt_at=UTC_TIMESTAMP(3)+INTERVAL 1 MINUTE WHERE id=?", id)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
func (s *Server) serveFrontend(w http.ResponseWriter, r *http.Request) {
	directory := os.Getenv("APP_FRONTEND_DIR")
	if directory == "" {
		http.NotFound(w, r)
		return
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		fail(w, 405, "method_not_allowed")
		return
	}
	if r.URL.Path == "/app" {
		http.Redirect(w, r, "/app/", http.StatusTemporaryRedirect)
		return
	}
	// Frontend has no inline JS. OpenStreetMap supplies the attributed raster basemap.
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data: blob: https://tile.openstreetmap.org https://server.arcgisonline.com; connect-src 'self'; media-src 'self' blob:; object-src 'none'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
	w.Header().Set("Referrer-Policy", "no-referrer")
	http.StripPrefix("/app/", http.FileServer(http.Dir(directory))).ServeHTTP(w, r)
}
