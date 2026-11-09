package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/beego/beego/v2/server/web"
	"golang.org/x/oauth2/google"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const firebaseProject = "rumahcuci-56dec"

var pushHTTP *http.Client
var clientPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{16,64}$`)

func initPush() {
	path := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if path == "" {
		return
	}
	b, e := os.ReadFile(path)
	if e != nil {
		log.Print("FCM credential file unavailable")
		return
	}
	cfg, e := google.JWTConfigFromJSON(b, "https://www.googleapis.com/auth/firebase.messaging")
	if e != nil {
		log.Print("FCM credential format invalid")
		return
	}
	pushHTTP = cfg.Client(context.Background())
	pushHTTP.Timeout = 15 * time.Second
}

type NotificationAPI struct{ web.Controller }

func (c *NotificationAPI) Get() {
	if !authorize(&c.Controller, false) {
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 5*time.Second)
	defer cancel()
	u := currentUser(ctx)
	rows, e := query(ctx, `SELECT id,title,body,target,read_at,created_at FROM web_notifications WHERE user_id=? ORDER BY id DESC LIMIT 50`, u.ID)
	if e != nil {
		reply(&c.Controller, 500, map[string]string{"error": "Notifikasi tidak dapat dimuat"})
		return
	}
	var count int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM web_notifications WHERE user_id=? AND read_at IS NULL`, u.ID).Scan(&count)
	reply(&c.Controller, 200, map[string]any{"data": rows, "unread": count, "vapid_key": env("FCM_VAPID_PUBLIC_KEY", ""), "push_ready": pushHTTP != nil && os.Getenv("FCM_VAPID_PUBLIC_KEY") != "", "csrf_token": csrfToken(c.Ctx.Request)})
}
func (c *NotificationAPI) Post() {
	if !authorize(&c.Controller, true) {
		return
	}
	var in struct {
		Action   string `json:"action"`
		Token    string `json:"token"`
		ClientID string `json:"client_id"`
		ID       int64  `json:"id"`
	}
	c.Ctx.Request.Body = http.MaxBytesReader(c.Ctx.ResponseWriter, c.Ctx.Request.Body, 8192)
	d := json.NewDecoder(c.Ctx.Request.Body)
	d.DisallowUnknownFields()
	if d.Decode(&in) != nil || d.Decode(&struct{}{}) != io.EOF {
		reply(&c.Controller, 400, map[string]string{"error": "Data notifikasi tidak valid"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 8*time.Second)
	defer cancel()
	u := currentUser(ctx)
	var e error
	switch in.Action {
	case "subscribe":
		if !clientPattern.MatchString(in.ClientID) || len(in.Token) < 32 || len(in.Token) > 4096 || strings.ContainsAny(in.Token, "\r\n\x00") {
			reply(&c.Controller, 422, map[string]string{"error": "Token browser tidak valid"})
			return
		}
		cookie, _ := c.Ctx.Request.Cookie("pastrack_session")
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			e = err
			break
		}
		defer tx.Rollback()
		_, e = tx.ExecContext(ctx, `DELETE FROM web_push_tokens WHERE user_id=? AND client_id=?`, u.ID, in.ClientID)
		if e == nil {
			_, e = tx.ExecContext(ctx, `INSERT INTO web_push_tokens(token_hash,user_id,session_hash,client_id,token,created_at,updated_at) VALUES(?,?,?,?,?,UTC_TIMESTAMP(3),UTC_TIMESTAMP(3)) ON DUPLICATE KEY UPDATE user_id=VALUES(user_id),session_hash=VALUES(session_hash),client_id=VALUES(client_id),token=VALUES(token),updated_at=VALUES(updated_at)`, tokenHash(in.Token), u.ID, tokenHash(cookie.Value), in.ClientID, in.Token)
		}
		if e == nil {
			e = tx.Commit()
		}
	case "unsubscribe":
		if !clientPattern.MatchString(in.ClientID) {
			reply(&c.Controller, 422, map[string]string{"error": "Browser tidak valid"})
			return
		}
		_, e = db.ExecContext(ctx, `DELETE FROM web_push_tokens WHERE user_id=? AND client_id=?`, u.ID, in.ClientID)
	case "read":
		if in.ID > 0 {
			_, e = db.ExecContext(ctx, `UPDATE web_notifications SET read_at=COALESCE(read_at,UTC_TIMESTAMP(3)) WHERE user_id=? AND id=?`, u.ID, in.ID)
		} else {
			_, e = db.ExecContext(ctx, `UPDATE web_notifications SET read_at=UTC_TIMESTAMP(3) WHERE user_id=? AND read_at IS NULL`, u.ID)
		}
	case "test":
		var count int
		e = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM web_notifications WHERE user_id=? AND created_at>UTC_TIMESTAMP()-INTERVAL 1 MINUTE`, u.ID).Scan(&count)
		if e == nil && count >= 3 {
			reply(&c.Controller, 429, map[string]string{"error": "Tunggu satu menit sebelum mencoba kembali"})
			return
		}
		if e == nil {
			e = createNotification(ctx, u.ID, "Notifikasi Pastrack", "Notifikasi uji berhasil dibuat.", "/#notifications")
		}
	default:
		reply(&c.Controller, 400, map[string]string{"error": "Aksi tidak valid"})
		return
	}
	if e != nil {
		log.Print("notification operation failed")
		reply(&c.Controller, 500, map[string]string{"error": "Notifikasi belum dapat diproses"})
		return
	}
	if in.Action == "read" {
		publishRealtime("notifications")
	}
	reply(&c.Controller, 200, map[string]bool{"ok": true})
}
func createNotification(ctx context.Context, userID int64, title, body, target string) error {
	if len(title) == 0 || len(title) > 120 || len(body) > 1000 || !strings.HasPrefix(target, "/#") || len(target) > 200 {
		return errors.New("invalid notification")
	}
	tx, e := db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	r, e := tx.ExecContext(ctx, `INSERT INTO web_notifications(user_id,title,body,target,created_at) VALUES(?,?,?,?,UTC_TIMESTAMP(3))`, userID, title, body, target)
	if e != nil {
		return e
	}
	id, _ := r.LastInsertId()
	_, e = tx.ExecContext(ctx, `INSERT INTO web_push_jobs(notification_id,token_hash,next_attempt,updated_at) SELECT ?,t.token_hash,UTC_TIMESTAMP(3),UTC_TIMESTAMP(3) FROM web_push_tokens t JOIN web_sessions s ON s.token_hash=t.session_hash JOIN web_users u ON u.id=t.user_id WHERE t.user_id=? AND s.user_id=u.id AND s.expires_at>UTC_TIMESTAMP() AND s.auth_version=u.auth_version AND u.is_active=1 AND u.deleted_at IS NULL`, id, userID)
	if e != nil {
		return e
	}
	if e = tx.Commit(); e != nil {
		return e
	}
	publishRealtime("notifications")
	return nil
}
func pushSend(ctx context.Context, token, title, body, target string) (int, bool, error) {
	if pushHTTP == nil {
		return 0, false, errors.New("sender unavailable")
	}
	payload := map[string]any{"message": map[string]any{"token": token, "data": map[string]string{"title": title, "body": body, "url": target}, "webpush": map[string]any{"headers": map[string]string{"TTL": "3600"}}}}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://fcm.googleapis.com/v1/projects/"+firebaseProject+"/messages:send", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r, e := pushHTTP.Do(req)
	if e != nil {
		return 0, false, e
	}
	defer r.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(r.Body, 32768))
	unregistered := bytes.Contains(raw, []byte(`"UNREGISTERED"`))
	if r.StatusCode >= 300 {
		return r.StatusCode, unregistered, fmt.Errorf("FCM status %d", r.StatusCode)
	}
	return r.StatusCode, false, nil
}
func startPushWorker() {
	if pushHTTP == nil {
		return
	}
	go func() {
		for {
			processPushJobs()
			time.Sleep(5 * time.Second)
		}
	}()
}
func processPushJobs() {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	rows, e := query(ctx, `SELECT j.id,j.attempts,t.token,t.token_hash,n.title,n.body,n.target FROM web_push_jobs j JOIN web_push_tokens t ON t.token_hash=j.token_hash JOIN web_notifications n ON n.id=j.notification_id AND n.user_id=t.user_id JOIN web_sessions s ON s.token_hash=t.session_hash JOIN web_users u ON u.id=t.user_id WHERE j.state='pending' AND j.next_attempt<=UTC_TIMESTAMP() AND s.user_id=u.id AND s.expires_at>UTC_TIMESTAMP() AND s.auth_version=u.auth_version AND u.is_active=1 AND u.deleted_at IS NULL ORDER BY j.id LIMIT 3`)
	if e != nil {
		return
	}
	for _, r := range rows {
		status, invalid, err := pushSend(ctx, fmt.Sprint(r["token"]), fmt.Sprint(r["title"]), fmt.Sprint(r["body"]), fmt.Sprint(r["target"]))
		state := "sent"
		attempts, _ := r["attempts"].(int64)
		if err != nil {
			state = "pending"
			if invalid || attempts >= 4 || (status >= 400 && status < 500 && status != 429) {
				state = "failed"
			}
		}
		_, _ = db.ExecContext(ctx, `UPDATE web_push_jobs SET state=?,attempts=attempts+1,next_attempt=UTC_TIMESTAMP()+INTERVAL 60 SECOND,updated_at=UTC_TIMESTAMP(3) WHERE id=?`, state, r["id"])
		if invalid {
			_, _ = db.ExecContext(ctx, `DELETE FROM web_push_tokens WHERE token_hash=?`, r["token_hash"])
		}
	}
}

type FirebaseWorker struct{ web.Controller }

func (c *FirebaseWorker) Get() {
	b, e := assets.ReadFile("static/firebase-messaging-sw.js")
	if e != nil {
		c.Abort("500")
		return
	}
	c.Ctx.Output.Header("Content-Type", "application/javascript")
	c.Ctx.Output.Header("Service-Worker-Allowed", "/")
	c.Ctx.Output.Body(b)
}
