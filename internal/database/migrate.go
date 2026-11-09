// Package database implements an explicit, resumable development-only migration runner.
package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

// LocalTarget rejects live hosts and schemas even when environment variables are wrong.
func LocalTarget(host, name string) error {
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		return fmt.Errorf("migration requires loopback database host")
	}
	if !strings.HasPrefix(name, "pastrack_") || !strings.HasSuffix(name, "_local") {
		return fmt.Errorf("migration requires pastrack_*_local database name")
	}
	for _, c := range name {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_') {
			return fmt.Errorf("invalid database name")
		}
	}
	return nil
}

// Apply records every successful DDL statement because MySQL DDL implicitly commits.
// A changed checksum aborts rather than silently replaying a modified migration.
func Apply(ctx context.Context, db *sql.DB, source fs.FS) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	var locked int
	if err = conn.QueryRowContext(ctx, "SELECT GET_LOCK('pastrack_local_migration',0)").Scan(&locked); err != nil {
		return err
	}
	if locked != 1 {
		return fmt.Errorf("migration already running")
	}
	defer conn.ExecContext(context.Background(), "SELECT RELEASE_LOCK('pastrack_local_migration')")
	_, err = conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migration_steps (version VARCHAR(128) NOT NULL, step INT NOT NULL, checksum CHAR(64) NOT NULL, applied_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3), PRIMARY KEY(version,step))`)
	if err != nil {
		return err
	}
	files, err := fs.Glob(source, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, file := range files {
		content, err := fs.ReadFile(source, file)
		if err != nil {
			return err
		}
		// Migrations deliberately contain no stored procedures or semicolons in strings.
		var clean []string
		for _, line := range strings.Split(string(content), "\n") {
			if !strings.HasPrefix(strings.TrimSpace(line), "--") {
				clean = append(clean, line)
			}
		}
		for i, statement := range strings.Split(strings.Join(clean, "\n"), ";") {
			statement = strings.TrimSpace(statement)
			if statement == "" {
				continue
			}
			sum := sha256.Sum256([]byte(statement))
			checksum := hex.EncodeToString(sum[:])
			var previous string
			err = conn.QueryRowContext(ctx, "SELECT checksum FROM schema_migration_steps WHERE version=? AND step=?", file, i).Scan(&previous)
			if err == nil {
				if previous != checksum {
					return fmt.Errorf("migration checksum changed: %s step %d", file, i)
				}
				continue
			}
			if err != sql.ErrNoRows {
				return err
			}
			if _, err = conn.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("migration %s step %d: %w", file, i, err)
			}
			if _, err = conn.ExecContext(ctx, "INSERT INTO schema_migration_steps(version,step,checksum) VALUES(?,?,?)", file, i, checksum); err != nil {
				return err
			}
		}
	}
	return nil
}
