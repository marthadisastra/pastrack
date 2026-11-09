package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"pastrack/internal/alertengine"
	"pastrack/internal/alertrules"
	store "pastrack/internal/database"
	"pastrack/internal/platform"
	"strings"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
func run() error {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	name := os.Getenv("DB_NAME")
	if err := store.RuntimeTarget(host, name, os.Getenv("PASTRACK_RUNTIME")); err != nil {
		return err
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "33307"
	}
	cfg := mysql.NewConfig()
	cfg.User = os.Getenv("DB_USER")
	cfg.Passwd = os.Getenv("DB_PASSWORD")
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(host, port)
	cfg.DBName = name
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	cfg.Params = map[string]string{"time_zone": "'+00:00'"}
	cfg.Timeout = 5 * time.Second
	cfg.ReadTimeout = 10 * time.Second
	cfg.WriteTimeout = 10 * time.Second
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxOpenConns(12)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(5 * time.Minute)
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		return err
	}
	if file := os.Getenv("LOCAL_BOOTSTRAP_PASSWORD_FILE"); file != "" {
		if os.Getenv("PASTRACK_RUNTIME") == "production" {
			return fmt.Errorf("production runtime cannot bootstrap local accounts")
		}
		info, e := os.Stat(file)
		if e != nil {
			return e
		}
		if info.Mode().Perm()&0077 != 0 {
			return fmt.Errorf("bootstrap password file must be private")
		}
		password, e := os.ReadFile(file)
		if e != nil {
			return e
		}
		if err = platform.Bootstrap(ctx, db, os.Getenv("LOCAL_BOOTSTRAP_USER"), strings.TrimSpace(string(password))); err != nil {
			return err
		}
	}
	app := platform.New(db)
	if err = app.ConfigureAuthFromEnv(); err != nil {
		return err
	}
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := app.DeliverResetEmails(ctx); err != nil && ctx.Err() == nil {
					log.Print("password reset delivery queue temporarily unavailable")
				}
			}
		}
	}()
	alertWake := make(chan struct{}, 1)
	wakeAlerts := func() {
		select {
		case alertWake <- struct{}{}:
		default:
		}
	}
	app.OnNotify = wakeAlerts
	wakeAlerts()
	go runAlertWorker(ctx, db, app.Notify, alertWake)
	addr := os.Getenv("WEB_ADDR")
	if addr == "" {
		addr = "127.0.0.1:18090"
	}
	webHost, _, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	if webHost != "127.0.0.1" && webHost != "localhost" && webHost != "::1" {
		return fmt.Errorf("local server requires loopback bind")
	}
	// Reuse the existing private Unix datagram invalidation transport.
	socket := os.Getenv("REALTIME_SOCKET_PATH")
	if socket != "" && socket != "off" {
		a := &net.UnixAddr{Name: socket, Net: "unixgram"}
		conn, e := net.ListenUnixgram("unixgram", a)
		if e != nil {
			return e
		}
		defer conn.Close()
		defer os.Remove(socket)
		if e = os.Chmod(socket, 0600); e != nil {
			return e
		}
		go func() {
			b := make([]byte, 4096)
			for {
				_, _, e := conn.ReadFromUnix(b)
				if e != nil {
					return
				}
				app.Notify()
			}
		}()
	}
	server := &http.Server{Addr: addr, Handler: app, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}
	go func() {
		<-ctx.Done()
		stop, c := context.WithTimeout(context.Background(), 5*time.Second)
		defer c()
		server.Shutdown(stop)
	}()
	log.Printf("PasTrack local v2: http://%s", addr)
	err = server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func runAlertWorker(ctx context.Context, db *sql.DB, notify func(), wake <-chan struct{}) {
	workerID := fmt.Sprintf("web-%d", os.Getpid())
	runNow := false
	for {
		if !runNow {
			select {
			case <-ctx.Done():
				return
			case <-wake:
			}
		}
		runNow = false
		registry, err := alertrules.LoadEnabled(ctx, db, nil)
		retryAfterError := false
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("alert worker registry: %v", err)
			retryAfterError = true
		} else {
			worker := alertengine.Worker{DB: db, Registry: registry, ID: workerID}
			worker.Queue = alertengine.Queue{DB: db}
			worker.Episodes = alertengine.EpisodeManager{DB: db}
			changed := false
			for i := 0; i < 100; i++ {
				processed, runErr := worker.RunOnce(ctx)
				if runErr != nil {
					if ctx.Err() == nil {
						log.Printf("alert worker: %v", runErr)
					}
					retryAfterError = true
					break
				}
				if !processed {
					break
				}
				changed = true
			}
			delivered, deliveryErr := alertengine.DeliverNotifications(ctx, db, 200)
			if deliveryErr != nil && ctx.Err() == nil {
				log.Printf("alert notification delivery: %v", deliveryErr)
				retryAfterError = true
			}
			if changed || delivered > 0 {
				notify()
			}
		}
		queue := alertengine.Queue{DB: db}
		deadline, deadlineErr := queue.NextWake(ctx)
		if deadlineErr != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("alert worker deadline: %v", deadlineErr)
			deadlineValue := time.Now().Add(time.Second)
			deadline = &deadlineValue
		}
		if retryAfterError {
			deadlineValue := time.Now().Add(time.Second)
			deadline = &deadlineValue
		}
		if deadline != nil {
			delay := time.Until(*deadline)
			if delay < 0 {
				delay = 0
			}
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-wake:
				timer.Stop()
			case <-timer.C:
			}
			runNow = true
			continue
		}
	}
}
