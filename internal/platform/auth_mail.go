package platform

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type smtpConfig struct {
	Enabled   bool   `json:"enabled"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	TLSMode   string `json:"tlsMode"`
	Username  string `json:"username"`
	Password  string `json:"password,omitempty"`
	FromEmail string `json:"fromEmail"`
	FromName  string `json:"fromName"`
}

func defaultSMTP() smtpConfig {
	return smtpConfig{Port: 587, TLSMode: "starttls", FromName: "PASTRACK"}
}
func validateSMTP(v smtpConfig) error {
	if !v.Enabled {
		return nil
	}
	if v.Host == "" || len(v.Host) > 253 || strings.ContainsAny(v.Host, "/\\\r\n\t :") || v.Port < 1 || v.Port > 65535 {
		return fmt.Errorf("invalid_smtp_host_or_port")
	}
	if v.TLSMode != "starttls" && v.TLSMode != "tls" && v.TLSMode != "none" {
		return fmt.Errorf("invalid_smtp_tls_mode")
	}
	address, err := mail.ParseAddress(v.FromEmail)
	if err != nil || address.Address != v.FromEmail || len(v.FromName) > 100 || strings.ContainsAny(v.FromName+v.Username+v.Password, "\r\n") {
		return fmt.Errorf("invalid_smtp_identity")
	}
	if v.TLSMode == "none" {
		ip := net.ParseIP(v.Host)
		if (v.Host != "localhost" && (ip == nil || !ip.IsLoopback())) || v.Username != "" || v.Password != "" {
			return fmt.Errorf("smtp_tls_required")
		}
	}
	if len(v.Password) > 4096 || len(v.Username) > 190 {
		return fmt.Errorf("smtp_credentials_too_long")
	}
	return nil
}
func (s *Server) ConfigureAuthFromEnv() error {
	rawURL := os.Getenv("AUTH_APP_URL")
	if rawURL != "" {
		v, err := url.Parse(rawURL)
		if err != nil || v.Host == "" || v.User != nil || v.RawQuery != "" || v.Fragment != "" || (v.Scheme != "https" && v.Scheme != "http") {
			return fmt.Errorf("AUTH_APP_URL must be an absolute application URL")
		}
		if v.Scheme == "http" {
			ip := net.ParseIP(v.Hostname())
			if v.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback()) {
				return fmt.Errorf("AUTH_APP_URL requires HTTPS outside loopback")
			}
		}
		s.PublicAppURL = strings.TrimRight(rawURL, "/")
	}
	if raw := os.Getenv("AUTH_MAIL_KEY"); raw != "" {
		key, err := base64.StdEncoding.DecodeString(raw)
		if err != nil || len(key) != 32 {
			return fmt.Errorf("AUTH_MAIL_KEY must be base64 of 32 random bytes")
		}
		s.ResetMailKey = key
	}
	if host := os.Getenv("SMTP_HOST"); host != "" {
		cfg := defaultSMTP()
		cfg.Enabled = true
		cfg.Host = host
		cfg.Username = os.Getenv("SMTP_USERNAME")
		cfg.Password = os.Getenv("SMTP_PASSWORD")
		cfg.FromEmail = os.Getenv("SMTP_FROM")
		if v := os.Getenv("SMTP_PORT"); v != "" {
			cfg.Port, _ = strconv.Atoi(v)
		}
		if v := os.Getenv("SMTP_TLS_MODE"); v != "" {
			cfg.TLSMode = v
		}
		if err := validateSMTP(cfg); err != nil {
			return err
		}
		if len(s.ResetMailKey) != 32 || s.PublicAppURL == "" {
			return fmt.Errorf("SMTP recovery requires AUTH_MAIL_KEY and AUTH_APP_URL")
		}
		s.SendResetMail = makeResetSender(cfg)
	}
	return nil
}
func (s *Server) smtpConfig(ctx context.Context) (smtpConfig, error) {
	var encrypted []byte
	err := s.DB.QueryRowContext(ctx, "SELECT encrypted_config FROM auth_smtp_settings WHERE id=1").Scan(&encrypted)
	if err != nil {
		return defaultSMTP(), err
	}
	tag, payload, err := s.decryptReset(encrypted)
	if err != nil || tag != "smtp" {
		return smtpConfig{}, fmt.Errorf("smtp_config_unavailable")
	}
	var cfg smtpConfig
	if json.Unmarshal([]byte(payload), &cfg) != nil {
		return smtpConfig{}, fmt.Errorf("smtp_config_unavailable")
	}
	return cfg, nil
}
func (s *Server) resetSender(ctx context.Context) (func(context.Context, string, string) error, error) {
	cfg, err := s.smtpConfig(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return s.SendResetMail, nil
	}
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, nil
	}
	if err = validateSMTP(cfg); err != nil {
		return nil, err
	}
	return makeResetSender(cfg), nil
}
func smtpClient(ctx context.Context, cfg smtpConfig) (*smtp.Client, error) {
	dialer := net.Dialer{Timeout: 8 * time.Second}
	connection, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)))
	if err != nil {
		return nil, fmt.Errorf("smtp_connection_failed")
	}
	deadline := time.Now().Add(12 * time.Second)
	if at, ok := ctx.Deadline(); ok && at.Before(deadline) {
		deadline = at
	}
	connection.SetDeadline(deadline)
	config := &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}
	if cfg.TLSMode == "tls" {
		secured := tls.Client(connection, config)
		if err = secured.HandshakeContext(ctx); err != nil {
			connection.Close()
			return nil, fmt.Errorf("smtp_tls_failed")
		}
		connection = secured
	}
	client, err := smtp.NewClient(connection, cfg.Host)
	if err != nil {
		connection.Close()
		return nil, fmt.Errorf("smtp_handshake_failed")
	}
	reject := func(message string) (*smtp.Client, error) { client.Close(); return nil, fmt.Errorf("%s", message) }
	if cfg.TLSMode == "starttls" {
		if err = client.StartTLS(config); err != nil {
			return reject("smtp_starttls_failed")
		}
	}
	if cfg.Username != "" {
		if err = client.Auth(smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)); err != nil {
			return reject("smtp_authentication_failed")
		}
	}
	return client, nil
}
func makeResetSender(cfg smtpConfig) func(context.Context, string, string) error {
	return func(ctx context.Context, recipient, link string) error {
		address, err := mail.ParseAddress(recipient)
		if err != nil || address.Address != recipient {
			return fmt.Errorf("invalid_recipient")
		}
		client, err := smtpClient(ctx, cfg)
		if err != nil {
			return err
		}
		defer client.Close()
		if err = client.Mail(cfg.FromEmail); err != nil {
			return fmt.Errorf("smtp_sender_rejected")
		}
		if err = client.Rcpt(recipient); err != nil {
			return fmt.Errorf("smtp_recipient_rejected")
		}
		writer, err := client.Data()
		if err != nil {
			return fmt.Errorf("smtp_data_unavailable")
		}
		message := "From: " + (&mail.Address{Name: cfg.FromName, Address: cfg.FromEmail}).String() + "\r\nTo: " + address.String() + "\r\nSubject: Reset your PASTRACK password\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nA password reset was requested for your PASTRACK account.\r\n\r\nOpen this one-time link within 15 minutes:\r\n" + link + "\r\n\r\nIf you did not request this, ignore this email. Your password remains unchanged.\r\n"
		if _, err = writer.Write([]byte(message)); err != nil {
			return fmt.Errorf("smtp_message_failed")
		}
		if err = writer.Close(); err != nil {
			return fmt.Errorf("smtp_acceptance_failed")
		}
		client.Quit()
		return nil
	}
}
func (s *Server) authSMTP(w http.ResponseWriter, r *http.Request) {
	user := s.identity(r)
	if user == nil {
		fail(w, 401, "authentication_required")
		return
	}
	if user.Role != "admin" {
		fail(w, 403, "admin_required")
		return
	}
	if r.Method == "GET" && r.URL.Path == "/api/v1/auth/smtp" {
		cfg, err := s.smtpConfig(r.Context())
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			fail(w, 503, "smtp_config_unavailable")
			return
		}
		configured := cfg.Password != ""
		cfg.Password = ""
		respond(w, 200, map[string]any{"data": cfg, "passwordConfigured": configured, "recoveryReady": cfg.Enabled && len(s.ResetMailKey) == 32 && s.PublicAppURL != ""})
		return
	}
	if r.Method != "PUT" && !(r.Method == "POST" && r.URL.Path == "/api/v1/auth/smtp/test") {
		fail(w, 405, "method_not_allowed")
		return
	}
	copyRequest := r.Clone(r.Context())
	copyRequest.Method = "POST"
	if !authMutation(w, copyRequest) {
		return
	}
	if len(s.ResetMailKey) != 32 || s.PublicAppURL == "" {
		fail(w, 503, "smtp_config_unavailable")
		return
	}
	var input struct {
		smtpConfig
		ClearPassword bool `json:"clearPassword"`
	}
	if body(w, r, &input) != nil {
		fail(w, 400, "invalid_smtp_settings")
		return
	}
	old, err := s.smtpConfig(r.Context())
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		fail(w, 503, "smtp_config_unavailable")
		return
	}
	if input.Password == "" && !input.ClearPassword {
		input.Password = old.Password
	}
	if err = validateSMTP(input.smtpConfig); err != nil {
		fail(w, 400, err.Error())
		return
	}
	if r.URL.Path == "/api/v1/auth/smtp/test" {
		if !input.Enabled {
			fail(w, 400, "enable_smtp_first")
			return
		}
		if !s.authLimit("smtp-test:"+strconv.FormatInt(user.ID, 10), 5, 10*time.Minute) {
			fail(w, 429, "too_many_attempts")
			return
		}
		client, err := smtpClient(r.Context(), input.smtpConfig)
		if err != nil {
			fail(w, 502, err.Error())
			return
		}
		client.Quit()
		respond(w, 200, map[string]any{"data": true})
		return
	}
	data, _ := json.Marshal(input.smtpConfig)
	encrypted, err := s.encryptReset("smtp", string(data))
	if err != nil {
		fail(w, 503, "smtp_config_unavailable")
		return
	}
	if _, err = s.DB.ExecContext(r.Context(), "INSERT INTO auth_smtp_settings(id,encrypted_config,updated_by) VALUES(1,?,?) ON DUPLICATE KEY UPDATE encrypted_config=VALUES(encrypted_config),updated_by=VALUES(updated_by)", encrypted, user.ID); err != nil {
		fail(w, 503, "database_unavailable")
		return
	}
	respond(w, 200, map[string]any{"data": true})
}
