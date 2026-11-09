package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"log"
	"net"
	"os"
	migrations "pastrack/database"
	store "pastrack/internal/database"
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
	if err := store.LocalTarget(host, name); err != nil {
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
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	cfg.Timeout = 5 * time.Second
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if _, err = db.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS `"+name+"` CHARACTER SET utf8mb4"); err != nil {
		return err
	}
	cfg.DBName = name
	target, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return err
	}
	defer target.Close()
	if err = store.Apply(ctx, target, migrations.Migrations); err != nil {
		return err
	}
	fmt.Println("Local migrations applied:", name)
	return nil
}
