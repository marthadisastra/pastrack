// Package remoteoutput implements only GV30CEU v0306 steady digital output 1.
package remoteoutput

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

const DefaultPassword = "gv30ceu"

var passwordPattern = regexp.MustCompile(`^[A-Za-z0-9]{4,20}$`)
var serialPattern = regexp.MustCompile(`^[0-9A-F]{4}$`)

// Frame enables DOS reporting, disables the speed restriction, and allows output while ignition is on.
// A relay's electrical polarity and its effect on an engine are installation-specific.
func Frame(password string, active bool, serial string) (string, error) {
	if password == "" {
		password = DefaultPassword
	}
	if !passwordPattern.MatchString(password) || !serialPattern.MatchString(serial) {
		return "", fmt.Errorf("invalid command credentials or serial")
	}
	state := 0
	if active {
		state = 1
	}
	return fmt.Sprintf("AT+GTDOS=%s,0,1,1,%d,0,0,,,1,0,5,,1,,%s$", password, state, serial), nil
}

type Reply struct {
	Kind, IMEI, Serial string
	Active             *bool
	At                 time.Time
}

func Decode(payload string) (Reply, error) {
	var r Reply
	if !strings.HasSuffix(payload, "$") {
		return r, fmt.Errorf("incomplete output frame")
	}
	f := strings.Split(strings.TrimSuffix(payload, "$"), ",")
	if len(f) < 7 || len(f[2]) != 15 {
		return r, fmt.Errorf("short output frame")
	}
	r.IMEI = f[2]
	switch f[0] {
	case "+ACK:GTDOS":
		if len(f) != 7 {
			return r, fmt.Errorf("unexpected DOS ACK layout")
		}
		r.Kind = "ack"
		r.Serial = f[4]
		if !serialPattern.MatchString(r.Serial) {
			return r, fmt.Errorf("invalid serial")
		}
	case "+RESP:GTDOS", "+BUFF:GTDOS":
		if len(f) < 16 || f[4] != "1" || (f[5] != "0" && f[5] != "1") {
			return r, fmt.Errorf("unsupported output report")
		}
		r.Kind = "output"
		active := f[5] == "1"
		r.Active = &active
	default:
		return r, fmt.Errorf("not an output report")
	}
	at, e := time.Parse("20060102150405", f[len(f)-2])
	if e != nil {
		return r, e
	}
	r.At = at
	return r, nil
}
