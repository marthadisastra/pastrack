package alertengine

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
)

func itoa(value int64) string { return strconv.FormatInt(value, 10) }

func requestID() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "system-unavailable"
	}
	return hex.EncodeToString(value)
}
