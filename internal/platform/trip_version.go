package platform

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"pastrack/internal/trip"
)

func tripVersion(c trip.Config) string {
	b, _ := json.Marshal(c)
	h := sha256.Sum256(b)
	return fmt.Sprintf("gps-v1-%x", h[:8])
}
