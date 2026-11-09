package controllers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/beego/beego/v2/server/web"
	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

type userContextKey struct{}
type User struct {
	ID              int64      `json:"id"`
	Username        string     `json:"username"`
	FullName        string     `json:"full_name"`
	Email           string     `json:"email"`
	Phone           string     `json:"phone"`
	JobTitle        string     `json:"job_title"`
	Department      string     `json:"department"`
	Role            string     `json:"role"`
	Active          bool       `json:"is_active"`
	Version         int64      `json:"row_version"`
	AuthVersion     int64      `json:"-"`
	LastLogin       *time.Time `json:"last_login_at"`
	PasswordChanged *time.Time `json:"password_changed_at"`
}

func currentUser(ctx context.Context) *User { u, _ := ctx.Value(userContextKey{}).(*User); return u }
func authenticated(r *http.Request) bool {
	u := currentUser(r.Context())
	return u != nil && u.ID > 0 && u.Active
}
func actorName(ctx context.Context) string {
	if u := currentUser(ctx); u != nil {
		return u.Username
	}
	return ""
}
func tokenHash(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func resolveUser(r *http.Request) *User {
	c, e := r.Cookie("pastrack_session")
	if e != nil || len(c.Value) != 64 {
		return nil
	}
	if _, e = hex.DecodeString(c.Value); e != nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	var u User
	var email, phone, title, department sql.NullString
	var last, changed sql.NullTime
	e = db.QueryRowContext(ctx, `SELECT u.id,u.username,u.full_name,u.email,u.phone,u.job_title,u.department,u.role,u.is_active,u.row_version,u.auth_version,u.last_login_at,u.password_changed_at FROM web_sessions s JOIN web_users u ON u.id=s.user_id WHERE s.token_hash=? AND s.expires_at>? AND s.auth_version=u.auth_version AND u.is_active=1 AND u.deleted_at IS NULL`, tokenHash(c.Value), time.Now().UTC()).Scan(&u.ID, &u.Username, &u.FullName, &email, &phone, &title, &department, &u.Role, &u.Active, &u.Version, &u.AuthVersion, &last, &changed)
	if e != nil {
		return nil
	}
	u.Email = email.String
	u.Phone = phone.String
	u.JobTitle = title.String
	u.Department = department.String
	if last.Valid {
		u.LastLogin = &last.Time
	}
	if changed.Valid {
		u.PasswordChanged = &changed.Time
	}
	return &u
}
func allowRole(c *web.Controller, roles ...string) bool {
	u := currentUser(c.Ctx.Request.Context())
	if u == nil {
		reply(c, 401, map[string]string{"error": "Sesi berakhir. Silakan masuk kembali."})
		return false
	}
	for _, r := range roles {
		if u.Role == r {
			return true
		}
	}
	reply(c, 403, map[string]string{"error": "Anda tidak memiliki akses untuk tindakan ini."})
	return false
}
func clientIP(r *http.Request) string {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		if forwarded := net.ParseIP(r.Header.Get("X-Real-IP")); forwarded != nil {
			return forwarded.String()
		}
	}
	return host
}
func validPassword(p string) error {
	if utf8.RuneCountInString(p) < 12 || len(p) > 72 {
		return errors.New("Password harus minimal 12 karakter dan maksimal 72 byte.")
	}
	if strings.ContainsRune(p, 0) {
		return errors.New("Password mengandung karakter tidak valid.")
	}
	return nil
}
func checkPassword(stored, algo, password string) bool {
	if len(password) > 256 {
		return false
	}
	if algo == "legacy_sha256" {
		digest := sha256.Sum256([]byte(password))
		return subtle.ConstantTimeCompare([]byte(stored), []byte(hex.EncodeToString(digest[:]))) == 1
	}
	if algo != "bcrypt" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(stored), []byte(password)) == nil
}
func newSession(ctx context.Context, tx *sql.Tx, uID, authVersion int64) (string, error) {
	var b [32]byte
	if _, e := rand.Read(b[:]); e != nil {
		return "", e
	}
	token := hex.EncodeToString(b[:])
	_, e := tx.ExecContext(ctx, `INSERT INTO web_sessions (token_hash,user_id,auth_version,expires_at,created_at) VALUES (?,?,?,?,?)`, tokenHash(token), uID, authVersion, time.Now().UTC().Add(12*time.Hour), time.Now().UTC())
	return token, e
}
func sessionCookie(w http.ResponseWriter, token string) {
	max := 43200
	if token == "" {
		max = -1
	}
	http.SetCookie(w, &http.Cookie{Name: "pastrack_session", Value: token, Path: "/", HttpOnly: true, Secure: env("COOKIE_SECURE", "false") == "true", SameSite: http.SameSiteStrictMode, MaxAge: max})
}
func (c *Login) Post() {
	expected := "http://" + c.Ctx.Request.Host
	if env("COOKIE_SECURE", "false") == "true" {
		expected = "https://" + c.Ctx.Request.Host
	}
	if origin := c.Ctx.Request.Header.Get("Origin"); origin != "" && origin != expected {
		reply(&c.Controller, 403, map[string]string{"error": "Origin login tidak valid."})
		return
	}

	c.Ctx.Request.Body = http.MaxBytesReader(c.Ctx.ResponseWriter, c.Ctx.Request.Body, 4096)
	name := strings.ToLower(strings.TrimSpace(c.GetString("username")))
	password := c.GetString("password")
	if !allowLogin("ip:"+clientIP(c.Ctx.Request)) || !allowLogin("user:"+name) {
		c.Ctx.Output.SetStatus(429)
		c.Ctx.Output.Body([]byte("Terlalu banyak percobaan. Tunggu 10 menit."))
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 8*time.Second)
	defer cancel()
	var id, authVersion int64
	var stored, algo string
	e := db.QueryRowContext(ctx, `SELECT id,password_hash,password_algo,auth_version FROM web_users WHERE username=? AND is_active=1 AND deleted_at IS NULL`, name).Scan(&id, &stored, &algo, &authVersion)
	if e != nil || !checkPassword(stored, algo, password) {
		if e == sql.ErrNoRows {
			bcrypt.CompareHashAndPassword([]byte("$2a$12$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"), []byte(password))
		}
		c.Redirect("/login?error=1", 303)
		return
	}
	tx, e := db.BeginTx(ctx, nil)
	if e != nil {
		reply(&c.Controller, 503, map[string]string{"error": "Login belum dapat diproses."})
		return
	}
	defer tx.Rollback()
	var latestHash string
	var latestVersion int64
	var active bool
	var deleted sql.NullTime
	e = tx.QueryRowContext(ctx, "SELECT password_hash,auth_version,is_active,deleted_at FROM web_users WHERE id=? FOR UPDATE", id).Scan(&latestHash, &latestVersion, &active, &deleted)
	if e != nil || latestHash != stored || latestVersion != authVersion || !active || deleted.Valid {
		c.Redirect("/login?error=1", 303)
		return
	}
	if algo == "legacy_sha256" {
		var hash []byte
		hash, e = bcrypt.GenerateFromPassword([]byte(password), 12)
		if e == nil {
			_, e = tx.ExecContext(ctx, "UPDATE web_users SET password_hash=?,password_algo='bcrypt',row_version=row_version+1,updated_at=? WHERE id=?", string(hash), time.Now().UTC(), id)
		}
	}
	if e == nil {
		_, e = tx.ExecContext(ctx, "UPDATE web_users SET last_login_at=? WHERE id=?", time.Now().UTC(), id)
	}
	// Best-effort cleanup is bounded by the expiry index; session storage contains hashes only.
	if e == nil {
		_, e = tx.ExecContext(ctx, "DELETE FROM web_sessions WHERE expires_at<? LIMIT 200", time.Now().UTC())
	}
	var token string
	if e == nil {
		token, e = newSession(ctx, tx, id, authVersion)
	}
	if e == nil {
		e = tx.Commit()
	}
	if e != nil {
		log.Printf("login session: %v", e)
		reply(&c.Controller, 503, map[string]string{"error": "Login belum dapat diproses."})
		return
	}
	sessionCookie(c.Ctx.ResponseWriter, token)
	c.Redirect("/", 303)
}
func (c *Logout) Post() {
	if cookie, e := c.Ctx.Request.Cookie("pastrack_session"); e == nil {
		ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 3*time.Second)
		defer cancel()
		if _, e = db.ExecContext(ctx, "DELETE FROM web_sessions WHERE token_hash=?", tokenHash(cookie.Value)); e != nil {
			reply(&c.Controller, 503, map[string]string{"error": "Logout belum berhasil. Coba kembali."})
			return
		}
	}
	sessionCookie(c.Ctx.ResponseWriter, "")
	c.Redirect("/login", 303)
}

type UserInput struct {
	Username        string `json:"username"`
	FullName        string `json:"full_name"`
	Email           string `json:"email"`
	Phone           string `json:"phone"`
	JobTitle        string `json:"job_title"`
	Department      string `json:"department"`
	Role            string `json:"role"`
	Active          bool   `json:"is_active"`
	Password        string `json:"password"`
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
	Version         int64  `json:"row_version"`
}

var userNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,63}$`)

func validateUser(in *UserInput, admin bool) error {
	in.Username = strings.ToLower(strings.TrimSpace(in.Username))
	in.FullName = strings.TrimSpace(in.FullName)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Phone = strings.TrimSpace(in.Phone)
	in.JobTitle = strings.TrimSpace(in.JobTitle)
	in.Department = strings.TrimSpace(in.Department)
	if in.FullName == "" {
		return errors.New("Nama lengkap wajib diisi.")
	}
	for _, p := range []struct {
		s string
		n int
	}{{in.FullName, 100}, {in.Email, 190}, {in.Phone, 32}, {in.JobTitle, 100}, {in.Department, 100}} {
		if utf8.RuneCountInString(p.s) > p.n || strings.ContainsAny(p.s, "\x00\r\n") {
			return errors.New("Data profil terlalu panjang atau mengandung karakter tidak valid.")
		}
	}
	if in.Email != "" {
		parsed, e := mail.ParseAddress(in.Email)
		if e != nil || parsed.Address != in.Email {
			return errors.New("Alamat email tidak valid.")
		}
	}
	if admin {
		if !userNamePattern.MatchString(in.Username) {
			return errors.New("Username 3-64 karakter: huruf kecil, angka, titik, garis bawah, atau tanda minus.")
		}
		if in.Role != "admin" && in.Role != "operator" && in.Role != "viewer" {
			return errors.New("Peran user tidak valid.")
		}
	}
	return nil
}
func decodeUser(c *web.Controller, in *UserInput) bool {
	c.Ctx.Request.Body = http.MaxBytesReader(c.Ctx.ResponseWriter, c.Ctx.Request.Body, 8192)
	d := json.NewDecoder(c.Ctx.Request.Body)
	d.DisallowUnknownFields()
	if d.Decode(in) != nil || d.Decode(&struct{}{}) != io.EOF {
		reply(c, 400, map[string]string{"error": "Data formulir tidak valid."})
		return false
	}
	return true
}

const publicUserFields = "id,username,full_name,email,phone,job_title,department,role,is_active,row_version,last_login_at,password_changed_at,deleted_at,created_at,updated_at"

func userSnapshot(ctx context.Context, tx *sql.Tx, id int64) (map[string]any, error) {
	var b []byte
	e := tx.QueryRowContext(ctx, `SELECT JSON_OBJECT('id',id,'username',username,'full_name',full_name,'email',email,'phone',phone,'job_title',job_title,'department',department,'role',role,'is_active',is_active,'row_version',row_version,'deleted_at',deleted_at) FROM web_users WHERE id=? FOR UPDATE`, id).Scan(&b)
	if e != nil {
		return nil, e
	}
	var row map[string]any
	e = json.Unmarshal(b, &row)
	return row, e
}
func userDBError(c *web.Controller, e error) {
	var me *mysql.MySQLError
	if errors.As(e, &me) && me.Number == 1062 {
		reply(c, 409, map[string]string{"error": "Username atau email sudah digunakan, termasuk akun diarsipkan."})
		return
	}
	log.Printf("user change: %v", e)
	reply(c, 503, map[string]string{"error": "Perubahan akun belum dapat disimpan. Muat ulang sebelum mencoba lagi."})
}
func userVersion(before map[string]any) int64 {
	n, _ := before["row_version"].(float64)
	return int64(n)
}
func requireVersion(c *web.Controller, before map[string]any, input int64) bool {
	if userVersion(before) != input {
		reply(c, 409, map[string]string{"error": "Data user telah berubah. Muat ulang sebelum menyimpan."})
		return false
	}
	return true
}

type Users struct{ web.Controller }

func (c *Users) Get() {
	if !allowRole(&c.Controller, "admin") {
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 8*time.Second)
	defer cancel()
	where := "deleted_at IS NULL"
	if c.GetString("archived") == "1" {
		where = "deleted_at IS NOT NULL"
	}
	args := []any{}
	if s := strings.TrimSpace(c.GetString("search")); s != "" {
		if len(s) > 100 {
			s = s[:100]
		}
		where += " AND (username LIKE ? OR full_name LIKE ? OR email LIKE ?)"
		args = append(args, "%"+s+"%", "%"+s+"%", "%"+s+"%")
	}
	page := integer(c.GetString("page"), 1, 1, 10000)
	args = append(args, (page-1)*50)
	rows, e := query(ctx, "SELECT "+publicUserFields+" FROM web_users WHERE "+where+" ORDER BY id DESC LIMIT 51 OFFSET ?", args...)
	if e != nil {
		userDBError(&c.Controller, e)
		return
	}
	reply(&c.Controller, 200, map[string]any{"data": rows, "csrf_token": csrfToken(c.Ctx.Request), "page": page})
}
func (c *Users) Post()          { c.mutate("create") }
func (c *Users) Put()           { c.mutate("update") }
func (c *Users) Delete()        { c.mutate("archive") }
func (c *Users) Restore()       { c.mutate("restore") }
func (c *Users) ResetPassword() { c.mutate("reset_password") }
func (c *Users) mutate(action string) {
	if !authorize(&c.Controller, true) || !allowRole(&c.Controller, "admin") {
		return
	}
	u := currentUser(c.Ctx.Request.Context())
	id := int64(integer(c.Ctx.Input.Param(":id"), 0, 0, 2147483647))
	if (action == "create") != (id == 0) {
		reply(&c.Controller, 400, map[string]string{"error": "ID user tidak valid."})
		return
	}
	var in UserInput
	if !decodeUser(&c.Controller, &in) {
		return
	}
	if action == "create" || action == "update" {
		if e := validateUser(&in, true); e != nil {
			reply(&c.Controller, 422, map[string]string{"error": e.Error()})
			return
		}
	}
	var encoded string
	if action == "create" || action == "reset_password" {
		if in.Password != in.ConfirmPassword {
			reply(&c.Controller, 422, map[string]string{"error": "Konfirmasi password tidak sama."})
			return
		}
		if e := validPassword(in.Password); e != nil {
			reply(&c.Controller, 422, map[string]string{"error": e.Error()})
			return
		}
		hash, e := bcrypt.GenerateFromPassword([]byte(in.Password), 12)
		if e != nil {
			userDBError(&c.Controller, e)
			return
		}
		encoded = string(hash)
	} else if in.Password != "" || in.NewPassword != "" {
		reply(&c.Controller, 422, map[string]string{"error": "Gunakan tindakan reset password secara terpisah."})
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 10*time.Second)
	defer cancel()
	tx, e := db.BeginTx(ctx, nil)
	if e != nil {
		userDBError(&c.Controller, e)
		return
	}
	defer tx.Rollback()
	// Serialize administrator changes and prevent concurrent last-admin removals.
	rows, e := tx.QueryContext(ctx, "SELECT id FROM web_users WHERE role='admin' AND is_active=1 AND deleted_at IS NULL ORDER BY id FOR UPDATE")
	if e != nil {
		userDBError(&c.Controller, e)
		return
	}
	admins := 0
	actorStillAdmin := false
	for rows.Next() {
		var n int64
		if e = rows.Scan(&n); e != nil {
			break
		}
		admins++
		if n == u.ID {
			actorStillAdmin = true
		}
	}
	rows.Close()
	if e != nil {
		userDBError(&c.Controller, e)
		return
	}
	if !actorStillAdmin {
		reply(&c.Controller, 403, map[string]string{"error": "Hak administrator telah berubah. Silakan login kembali."})
		return
	}
	var before map[string]any
	if action != "create" {
		before, e = userSnapshot(ctx, tx, id)
		if e == sql.ErrNoRows {
			reply(&c.Controller, 404, map[string]string{"error": "User tidak ditemukan."})
			return
		}
		if e != nil {
			userDBError(&c.Controller, e)
			return
		}
		if !requireVersion(&c.Controller, before, in.Version) {
			return
		}
		if (action == "restore") != (before["deleted_at"] != nil) {
			reply(&c.Controller, 409, map[string]string{"error": "Status arsip user telah berubah."})
			return
		}
	}
	removing := action == "archive" || (action == "update" && (!in.Active || in.Role != "admin"))
	if action != "create" && removing && before["role"] == "admin" && before["is_active"] == float64(1) && admins <= 1 {
		reply(&c.Controller, 409, map[string]string{"error": "Administrator aktif terakhir tidak dapat dinonaktifkan, diarsipkan, atau diturunkan perannya."})
		return
	}
	if id == u.ID && (action == "archive" || action == "reset_password" || (action == "update" && (!in.Active || in.Role != u.Role))) {
		reply(&c.Controller, 409, map[string]string{"error": "Anda tidak dapat menonaktifkan atau mengubah peran akun sendiri. Ganti password melalui Profil saya."})
		return
	}
	now := time.Now().UTC()
	switch action {
	case "create":
		var res sql.Result
		res, e = tx.ExecContext(ctx, `INSERT INTO web_users (username,full_name,email,phone,job_title,department,role,password_hash,password_algo,is_active,created_at,updated_at,password_changed_at) VALUES (?,?,?,?,?,?,?,?,'bcrypt',?,?,?,?)`, in.Username, in.FullName, nullableText(in.Email), nullableText(in.Phone), nullableText(in.JobTitle), nullableText(in.Department), in.Role, encoded, in.Active, now, now, now)
		if e == nil {
			id, e = res.LastInsertId()
		}
	case "update":
		changedSecurity := before["role"] != in.Role || before["is_active"] != func() float64 {
			if in.Active {
				return 1
			}
			return 0
		}() || before["username"] != in.Username
		increment := 0
		if changedSecurity {
			increment = 1
		}
		_, e = tx.ExecContext(ctx, `UPDATE web_users SET username=?,full_name=?,email=?,phone=?,job_title=?,department=?,role=?,is_active=?,auth_version=auth_version+?,row_version=row_version+1,updated_at=? WHERE id=?`, in.Username, in.FullName, nullableText(in.Email), nullableText(in.Phone), nullableText(in.JobTitle), nullableText(in.Department), in.Role, in.Active, increment, now, id)
	case "archive":
		_, e = tx.ExecContext(ctx, "UPDATE web_users SET deleted_at=?,is_active=0,auth_version=auth_version+1,row_version=row_version+1,updated_at=? WHERE id=?", now, now, id)
	case "restore":
		_, e = tx.ExecContext(ctx, "UPDATE web_users SET deleted_at=NULL,is_active=0,auth_version=auth_version+1,row_version=row_version+1,updated_at=? WHERE id=?", now, id)
	case "reset_password":
		_, e = tx.ExecContext(ctx, "UPDATE web_users SET password_hash=?,password_algo='bcrypt',password_changed_at=?,auth_version=auth_version+1,row_version=row_version+1,updated_at=? WHERE id=?", encoded, now, now, id)
	}
	var after map[string]any
	req := requestID()
	if e == nil {
		after, e = userSnapshot(ctx, tx, id)
	}
	if e == nil {
		e = audit(ctx, tx, action, "users", id, before, after, req)
	}
	if e == nil {
		e = tx.Commit()
	}
	if e != nil {
		userDBError(&c.Controller, e)
		return
	}
	publishRealtime("users")
	reply(&c.Controller, 200, map[string]any{"data": after, "request_id": req, "message": "Perubahan user berhasil disimpan."})
}
func nullableText(s string) any {
	if s == "" {
		return nil
	}
	return s
}

type Me struct{ web.Controller }

func (c *Me) Get() {
	if !authorize(&c.Controller, false) {
		return
	}
	u := currentUser(c.Ctx.Request.Context())
	reply(&c.Controller, 200, map[string]any{"data": u, "csrf_token": csrfToken(c.Ctx.Request), "permissions": map[string]bool{"manage_users": u.Role == "admin", "manage_master": u.Role == "admin" || u.Role == "operator", "view_audit": u.Role == "admin"}})
}
func (c *Me) Put() {
	if !authorize(&c.Controller, true) {
		return
	}
	var in UserInput
	if !decodeUser(&c.Controller, &in) {
		return
	}
	if in.Role != "" || in.Username != "" || in.Password != "" || in.NewPassword != "" {
		reply(&c.Controller, 422, map[string]string{"error": "Username dan peran hanya dapat diubah administrator; password melalui formulir terpisah."})
		return
	}
	if e := validateUser(&in, false); e != nil {
		reply(&c.Controller, 422, map[string]string{"error": e.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 8*time.Second)
	defer cancel()
	tx, e := db.BeginTx(ctx, nil)
	if e != nil {
		userDBError(&c.Controller, e)
		return
	}
	defer tx.Rollback()
	u := currentUser(ctx)
	before, e := userSnapshot(ctx, tx, u.ID)
	if e != nil {
		userDBError(&c.Controller, e)
		return
	}
	if !requireVersion(&c.Controller, before, in.Version) {
		return
	}
	if before["deleted_at"] != nil || before["is_active"] != float64(1) {
		reply(&c.Controller, 401, map[string]string{"error": "Akun telah dinonaktifkan. Silakan login kembali."})
		return
	}
	_, e = tx.ExecContext(ctx, "UPDATE web_users SET full_name=?,email=?,phone=?,job_title=?,department=?,row_version=row_version+1,updated_at=? WHERE id=?", in.FullName, nullableText(in.Email), nullableText(in.Phone), nullableText(in.JobTitle), nullableText(in.Department), time.Now().UTC(), u.ID)
	var after map[string]any
	req := requestID()
	if e == nil {
		after, e = userSnapshot(ctx, tx, u.ID)
	}
	if e == nil {
		e = audit(ctx, tx, "profile_update", "users", u.ID, before, after, req)
	}
	if e == nil {
		e = tx.Commit()
	}
	if e != nil {
		userDBError(&c.Controller, e)
		return
	}
	reply(&c.Controller, 200, map[string]any{"data": after, "message": "Profil berhasil diperbarui.", "request_id": req})
}
func (c *Me) Password() {
	if !authorize(&c.Controller, true) {
		return
	}
	var in UserInput
	if !decodeUser(&c.Controller, &in) {
		return
	}
	if in.NewPassword != in.ConfirmPassword {
		reply(&c.Controller, 422, map[string]string{"error": "Konfirmasi password baru tidak sama."})
		return
	}
	if e := validPassword(in.NewPassword); e != nil {
		reply(&c.Controller, 422, map[string]string{"error": e.Error()})
		return
	}
	if in.NewPassword == in.CurrentPassword {
		reply(&c.Controller, 422, map[string]string{"error": "Password baru harus berbeda dari password saat ini."})
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 10*time.Second)
	defer cancel()
	u := currentUser(ctx)
	if !allowLogin("password:" + strconv.FormatInt(u.ID, 10)) {
		reply(&c.Controller, 429, map[string]string{"error": "Terlalu banyak percobaan. Tunggu 10 menit."})
		return
	}
	var stored, algo string
	e := db.QueryRowContext(ctx, "SELECT password_hash,password_algo FROM web_users WHERE id=?", u.ID).Scan(&stored, &algo)
	if e != nil {
		userDBError(&c.Controller, e)
		return
	}
	if !checkPassword(stored, algo, in.CurrentPassword) {
		reply(&c.Controller, 422, map[string]string{"error": "Password saat ini tidak sesuai."})
		return
	}
	encoded, e := bcrypt.GenerateFromPassword([]byte(in.NewPassword), 12)
	if e != nil {
		userDBError(&c.Controller, e)
		return
	}
	tx, e := db.BeginTx(ctx, nil)
	if e != nil {
		userDBError(&c.Controller, e)
		return
	}
	defer tx.Rollback()
	before, e := userSnapshot(ctx, tx, u.ID)
	if e != nil {
		userDBError(&c.Controller, e)
		return
	}
	var lockedHash string
	var lockedVersion int64
	e = tx.QueryRowContext(ctx, "SELECT password_hash,auth_version FROM web_users WHERE id=?", u.ID).Scan(&lockedHash, &lockedVersion)
	if e != nil {
		userDBError(&c.Controller, e)
		return
	}
	if lockedHash != stored || lockedVersion != u.AuthVersion {
		reply(&c.Controller, 409, map[string]string{"error": "Keamanan akun telah berubah. Silakan login kembali."})
		return
	}
	_, e = tx.ExecContext(ctx, "UPDATE web_users SET password_hash=?,password_algo='bcrypt',password_changed_at=?,auth_version=auth_version+1,row_version=row_version+1,updated_at=? WHERE id=?", string(encoded), time.Now().UTC(), time.Now().UTC(), u.ID)
	var after map[string]any
	req := requestID()
	if e == nil {
		after, e = userSnapshot(ctx, tx, u.ID)
	}
	if e == nil {
		e = audit(ctx, tx, "password_change", "users", u.ID, before, after, req)
	}
	if e == nil {
		e = tx.Commit()
	}
	if e != nil {
		userDBError(&c.Controller, e)
		return
	}
	sessionCookie(c.Ctx.ResponseWriter, "")
	reply(&c.Controller, 200, map[string]any{"message": "Password berhasil diubah. Silakan login kembali pada semua perangkat.", "reauthenticate": true})
}
