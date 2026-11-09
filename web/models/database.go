package models

import (
	"context"
	"database/sql"
	"github.com/go-sql-driver/mysql"
	"net"
	"os"
	"time"
)

var DB *sql.DB

func Env(k, d string) string {
	if s := os.Getenv(k); s != "" {
		return s
	}
	return d
}
func Open() (*sql.DB, error) {
	cfg := mysql.NewConfig()
	cfg.User = Env("DB_USER", "gps_listener")
	cfg.Passwd = os.Getenv("DB_PASSWORD")
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(Env("DB_HOST", "127.0.0.1"), Env("DB_PORT", "3306"))
	cfg.DBName = Env("DB_NAME", "pastrack")
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	cfg.Params = map[string]string{"time_zone": "'+00:00'"}
	cfg.Timeout = 5 * time.Second
	cfg.ReadTimeout = 8 * time.Second
	cfg.WriteTimeout = 8 * time.Second
	var e error
	DB, e = sql.Open("mysql", cfg.FormatDSN())
	if e == nil {
		DB.SetMaxOpenConns(8)
		DB.SetMaxIdleConns(4)
		DB.SetConnMaxLifetime(5 * time.Minute)
	}
	return DB, e
}

type Queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func Query(ctx context.Context, q string, args ...any) ([]map[string]any, error) {
	return QueryOn(ctx, DB, q, args...)
}
func QueryOn(ctx context.Context, conn Queryer, q string, args ...any) ([]map[string]any, error) {
	rows, e := conn.QueryContext(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	cols, e := rows.Columns()
	if e != nil {
		return nil, e
	}
	out := make([]map[string]any, 0)
	for rows.Next() {
		values := make([]any, len(cols))
		ptr := make([]any, len(cols))
		for i := range ptr {
			ptr[i] = &values[i]
		}
		if e = rows.Scan(ptr...); e != nil {
			return nil, e
		}
		r := make(map[string]any, len(cols))
		for i, k := range cols {
			switch v := values[i].(type) {
			case []byte:
				r[k] = string(v)
			case time.Time:
				r[k] = v.UTC().Format(time.RFC3339Nano)
			default:
				r[k] = v
			}
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
