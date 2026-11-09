package database

import (
	"fmt"
	"strings"
)

// RuntimeTarget separates explicitly configured production v2 from local test schemas.
// This does not run migrations or permit destructive database operations.
func RuntimeTarget(host, name, mode string) error {
	if mode != "production" {
		return LocalTarget(host, name)
	}
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		return fmt.Errorf("production database must be accessed over loopback")
	}
	if !strings.HasPrefix(name, "pastrack_v2_") || strings.HasSuffix(name, "_local") {
		return fmt.Errorf("production runtime requires an explicit pastrack_v2_* target")
	}
	for _, c := range name {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_') {
			return fmt.Errorf("invalid production target")
		}
	}
	return nil
}
