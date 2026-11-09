package protocol

import (
	"fmt"
	"strings"
	"time"
)

// GTIGL/GTVGL Report Type follows GV30CEU v0306 pp192-193: 0 on, 1 off.
type IgnitionReport struct {
	State, Source string
	At            time.Time
	GPSTime       time.Time
}

func DecodeIgnition(payload string) (IgnitionReport, error) {
	var r IgnitionReport
	if !strings.HasSuffix(payload, "$") {
		return r, fmt.Errorf("incomplete ignition report")
	}
	f := strings.Split(strings.TrimSuffix(payload, "$"), ",")
	if len(f) < 16 {
		return r, fmt.Errorf("short ignition report")
	}
	head := strings.Split(f[0], ":")
	if len(head) != 2 || (head[0] != "+RESP" && head[0] != "+BUFF") {
		return r, fmt.Errorf("unsupported ignition report")
	}
	gpsIndex := 13
	switch head[1] {
	case "GTIGL", "GTVGL":
		if len(f[5]) != 2 || !strings.Contains("0123456789ABCDEF", strings.ToUpper(f[5][:1])) {
			return r, fmt.Errorf("invalid ignition report type")
		}
		switch f[5][1] {
		case '0':
			r.State = "ON"
		case '1':
			r.State = "OFF"
		default:
			return r, fmt.Errorf("unknown ignition state")
		}
		if head[1] == "GTIGL" && f[5][0] != '0' {
			return r, fmt.Errorf("unknown GTIGL report id")
		}
	case "GTIGN":
		r.State = "ON"
		gpsIndex = 11
	case "GTIGF":
		r.State = "OFF"
		gpsIndex = 11
	case "GTVGN", "GTVGF":
		if f[4] != "00" || !strings.Contains("2457", f[5]) || len(f[5]) != 1 {
			return r, fmt.Errorf("unknown virtual ignition source")
		}
		r.State = "ON"
		if head[1] == "GTVGF" {
			r.State = "OFF"
		}
		gpsIndex = 13
	default:
		return r, fmt.Errorf("unsupported ignition report")
	}
	at, e := time.Parse("20060102150405", f[gpsIndex])
	if e != nil {
		return r, fmt.Errorf("invalid ignition GPS time")
	}
	r.GPSTime = at
	sent, e := time.Parse("20060102150405", f[len(f)-2])
	if e != nil {
		return r, fmt.Errorf("invalid ignition send time")
	}
	r.At = sent
	r.Source = head[1]
	return r, nil
}
