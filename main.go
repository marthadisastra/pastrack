package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"pastrack/internal/remoteoutput"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-sql-driver/mysql"
	localdb "pastrack/internal/database"
)

const maxFrame = 60000 // MySQL TEXT stores at most 65535 bytes.
func env(k, d string) string {
	if v, ok := os.LookupEnv(k); ok {
		return v
	}
	return d
}
func main() {
	if e := run(); e != nil {
		slog.Error("listener stopped", "error", e)
		os.Exit(1)
	}
}

var outputService *remoteoutput.Service

func run() error {
	loc, e := time.LoadLocation(env("GPS_TIMEZONE", "UTC"))
	if e != nil {
		return e
	}
	cfg := mysql.NewConfig()
	cfg.User = env("DB_USER", "gps_listener")
	cfg.Passwd = os.Getenv("DB_PASSWORD")
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(env("DB_HOST", "127.0.0.1"), env("DB_PORT", "3306"))
	cfg.DBName = env("DB_NAME", "gps_tracker_rnd")
	if os.Getenv("PASTRACK_SCHEMA") == "v2" {
		if e = localdb.RuntimeTarget(env("DB_HOST", "127.0.0.1"), cfg.DBName, os.Getenv("PASTRACK_RUNTIME")); e != nil {
			return e
		}
		host := env("GPS_HOST", "127.0.0.1")
		if os.Getenv("PASTRACK_RUNTIME") != "production" && host != "127.0.0.1" && host != "::1" && host != "localhost" {
			return fmt.Errorf("v2 receiver requires loopback binding during local rebuild")
		}
	}
	cfg.Params = map[string]string{"time_zone": "'+00:00'"}
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	cfg.Timeout = 10 * time.Second
	cfg.ReadTimeout = 15 * time.Second
	cfg.WriteTimeout = 15 * time.Second
	db, e := sql.Open("mysql", cfg.FormatDSN())
	if e != nil {
		return e
	}
	defer db.Close()
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(5 * time.Minute)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if commandSocket := os.Getenv("REMOTE_OUTPUT_SOCKET"); commandSocket != "" && commandSocket != "off" {
		service, err := remoteoutput.New(db, os.Getenv("REMOTE_OUTPUT_PROFILES"))
		if err != nil {
			return err
		}
		service.Notify = signalRealtime
		service.CommandLog = func(imei, serial, frame, state string) {
			if env("RAW_TEXT_LOG", "off") != "on" {
				return
			}
			path := filepath.Join(filepath.Dir(env("SPOOL_DIR", "/opt/pastrack/spool")), "raw-messages.txt")
			if err := appendRawLog(path, "TX GTDOS device="+imei+" serial="+serial+" state="+state+" "+frame); err != nil {
				slog.Error("command text log write failed", "error", err)
			}
		}
		commandListener, err := net.Listen("unix", commandSocket)
		if err != nil {
			return err
		}
		defer commandListener.Close()
		defer os.Remove(commandSocket)
		if err = os.Chmod(commandSocket, 0600); err != nil {
			return err
		}
		outputService = service
		commandServer := &http.Server{Handler: service, ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second}
		defer commandServer.Close()
		go commandServer.Serve(commandListener)
	}

	spool := env("SPOOL_DIR", "/opt/pastrack/spool")
	if e = os.MkdirAll(spool, 0700); e != nil {
		return e
	}
	listener, e := net.Listen("tcp", net.JoinHostPort(env("GPS_HOST", "127.0.0.1"), env("GPS_PORT", "5001")))
	if e != nil {
		return e
	}
	defer listener.Close()
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		c, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		files, err := filepath.Glob(filepath.Join(spool, "*.json"))
		if err != nil {
			http.Error(w, "spool unavailable", 503)
			return
		}
		if db.PingContext(c) != nil {
			http.Error(w, "database unavailable; frames spooled", 503)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "ok", "queued": len(files), "fuel_decoder": "raw_cfs300b_supported_units_unverified", "schema": env("PASTRACK_SCHEMA", "legacy")})
	})
	healthListener, e := net.Listen("tcp", env("HEALTH_ADDR", "127.0.0.1:18087"))
	if e != nil {
		return e
	}
	health := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	defer health.Close()
	go health.Serve(healthListener)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); drain(ctx, db, spool, loc) }()
	go func() {
		defer wg.Done()
		publishChanges(ctx, db, env("REALTIME_SOCKET_PATH", "/tmp/pastrack/live.sock"))
	}()
	go func() { <-ctx.Done(); listener.Close(); health.Close() }()
	slots := make(chan struct{}, 256)
	slog.Info("Pastrack started", "tcp", listener.Addr(), "health", healthListener.Addr())
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			stop()
			wg.Wait()
			return err
		}
		select {
		case slots <- struct{}{}:
			wg.Add(1)
			go func() { defer wg.Done(); defer func() { <-slots }(); handle(ctx, conn, spool) }()
		default:
			conn.Close()
		}
	}
	wg.Wait()
	return nil
}
func readFrame(r *bufio.Reader) (string, error) {
	var b strings.Builder
	for {
		part, e := r.ReadSlice('$')
		if b.Len()+len(part) > maxFrame {
			return "", fmt.Errorf("frame exceeds %d bytes", maxFrame)
		}
		b.Write(part)
		if e == nil {
			return b.String(), nil
		}
		if !errors.Is(e, bufio.ErrBufferFull) {
			return "", e
		}
	}
}
func handle(ctx context.Context, c net.Conn, spool string) {
	defer c.Close()
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			c.Close()
		case <-done:
		}
	}()
	slog.Info("device connected", "peer", c.RemoteAddr().String())
	boundIMEI := ""
	var unbind func()
	defer func() {
		if unbind != nil {
			unbind()
		}
	}()
	reader := bufio.NewReaderSize(c, 8192)
	for {
		c.SetReadDeadline(time.Now().Add(300 * time.Second))
		frame, e := readFrame(reader)
		if e != nil {
			if e != io.EOF && ctx.Err() == nil {
				slog.Warn("connection closed", "peer", c.RemoteAddr().String(), "error", e)
			}
			return
		}
		if strings.TrimSpace(frame) == "$" {
			continue
		}
		// Payload is JSON escaped without dropping non-ASCII bytes.
		for _, b := range []byte(frame) {
			if b > 127 {
				slog.Warn("non ASCII frame retained")
				break
			}
		}
		if outputService != nil {
			fields := strings.Split(frame, ",")
			if len(fields) > 2 {
				imei := fields[2]
				if boundIMEI == "" {
					boundIMEI = imei
					unbind = outputService.Bind(imei, c)
				} else if imei != boundIMEI {
					return
				}
			}
		}
		if e = enqueue(spool, Record{Payload: frame, Received: time.Now().UTC(), Peer: c.RemoteAddr().String()}); e != nil {
			slog.Error("spool write failed", "error", e)
			return
		}
		if env("RAW_TEXT_LOG", "off") == "on" {
			if e = appendRawLog(filepath.Join(filepath.Dir(spool), "raw-messages.txt"), frame); e != nil {
				slog.Error("raw text log write failed", "error", e)
			}
		}
	}
}
func appendRawLog(path, frame string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(f, "%s %q\n", time.Now().UTC().Format(time.RFC3339Nano), frame)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}
func enqueue(dir string, r Record) error {
	var id [16]byte
	if _, e := rand.Read(id[:]); e != nil {
		return e
	}
	r.ReceiptID = hex.EncodeToString(id[:])
	r.PayloadBytes = []byte(r.Payload)
	b, e := json.Marshal(r)
	if e != nil {
		return e
	}
	name := fmt.Sprintf("%020d-%s", r.Received.UnixNano(), r.ReceiptID)
	tmp := filepath.Join(dir, name+".tmp")
	f, e := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	defer os.Remove(tmp)
	_, e = f.Write(b)
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	if e = os.Rename(tmp, filepath.Join(dir, name+".json")); e != nil {
		return e
	}
	d, e := os.Open(dir)
	if e != nil {
		return e
	}
	defer d.Close()
	if e = d.Sync(); e != nil {
		return e
	}
	select {
	case spoolReady <- struct{}{}:
	default:
	}
	return nil
}
func drain(ctx context.Context, db *sql.DB, dir string, loc *time.Location) {
	for {
		failed := false
		files, e := filepath.Glob(filepath.Join(dir, "*.json"))
		if e != nil {
			failed = true
			slog.Error("spool read failed", "error", e)
		}
		sort.Strings(files)
		for _, name := range files {
			if ctx.Err() != nil {
				return
			}
			b, e := os.ReadFile(name)
			var r Record
			if e == nil {
				e = json.Unmarshal(b, &r)
				if r.PayloadBytes != nil {
					r.Payload = string(r.PayloadBytes)
				}
				if r.ReceiptID == "" {
					r.ReceiptID = filepath.Base(name)
				}
			}
			if e != nil {
				failed = true
				slog.Error("invalid spool record retained in quarantine", "file", filepath.Base(name))
				if err := os.Rename(name, name+".invalid"); err != nil {
					break
				}
				continue
			}
			work, cancel := context.WithTimeout(ctx, 20*time.Second)
			e = save(work, db, r, loc)
			cancel()
			if e != nil {
				var permanent normalizationError
				if errors.As(e, &permanent) {
					if renameErr := os.Rename(name, name+".blocked"); renameErr != nil {
						failed = true
						break
					}
					slog.Error("normalization blocked; receipt and spool retained for repair", "code", permanent.code)
					continue
				}
				failed = true
				slog.Error("database write failed; frame retained", "error_kind", fmt.Sprintf("%T", e))
				break
			}
			if outputService != nil {
				work, cancel := context.WithTimeout(ctx, 5*time.Second)
				err := outputService.Record(work, r.Payload)
				cancel()
				if err != nil {
					failed = true
					slog.Error("output acknowledgement persistence failed")
					break
				}
			}
			if e = os.Remove(name); e != nil {
				failed = true
				slog.Error("spool remove failed", "error", e)
				break
			}
			signalFrameRealtime(r.Payload, loc)
			slog.Info("frame saved", "peer", r.Peer)
		}
		var retry *time.Timer
		var retryC <-chan time.Time
		if failed {
			retry = time.NewTimer(time.Second)
			retryC = retry.C
		}
		select {
		case <-ctx.Done():
			if retry != nil {
				retry.Stop()
			}
			return
		case <-spoolReady:
			if retry != nil {
				retry.Stop()
			}
		case <-retryC:
		}
	}
}
