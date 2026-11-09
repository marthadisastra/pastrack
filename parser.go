package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

type Message struct {
	Type     string    `json:"message_type"`
	Version  string    `json:"protocol_version"`
	IMEI     string    `json:"imei"`
	Model    string    `json:"model"`
	Fields   []string  `json:"fields"`
	Status   string    `json:"status_code_raw,omitempty"`
	Accuracy string    `json:"accuracy_raw,omitempty"`
	Position *Position `json:"position,omitempty"`
}
type Position struct {
	AccuracyRaw string     `json:"gnss_accuracy_raw"`
	FixStatus   string     `json:"fix_status"`
	HDOP        *float64   `json:"hdop"`
	Speed       float64    `json:"speed_kmh"`
	Heading     int        `json:"heading_deg"`
	Altitude    float64    `json:"altitude_m"`
	Longitude   float64    `json:"longitude"`
	Latitude    float64    `json:"latitude"`
	GPSTime     *time.Time `json:"gps_time"`
	SendTime    *time.Time `json:"device_send_time"`
	Satellites  *int       `json:"satellites"`
}

func deviceTime(s string, loc *time.Location) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	if len(s) != 14 {
		return nil, fmt.Errorf("invalid device timestamp")
	}
	t, e := time.ParseInLocation("20060102150405", s, loc)
	if e != nil {
		return nil, e
	}
	t = t.UTC()
	return &t, nil
}

// Only the GTSTT layout supplied in pastrack.py and observed on GV30CEU is decoded.
// All other layouts, including fuel reports, retain their complete raw fields.
func parseMessage(raw string, loc *time.Location) (m Message, err error) {
	s := strings.TrimSpace(raw)
	if !strings.HasSuffix(s, "$") {
		return m, fmt.Errorf("missing frame terminator")
	}
	f := strings.Split(strings.TrimSuffix(s, "$"), ",")
	// ACK/SACK envelopes are retained without treating their fields as RESP layout.
	for _, prefix := range []string{"+ACK:", "+SACK:"} {
		if strings.HasPrefix(f[0], prefix) {
			m.Type = strings.TrimSuffix(strings.TrimPrefix(prefix, "+"), ":")
			m.Fields = f
			if len(f) >= 4 && len(f[2]) == 15 && strings.Trim(f[2], "0123456789") == "" {
				m.Version = f[1]
				m.IMEI = f[2]
				m.Model = f[3]
			}
			return m, nil
		}
	}
	if len(f) < 4 {
		return m, fmt.Errorf("incomplete header")
	}
	switch {
	case strings.HasPrefix(f[0], "+RESP:"):
		m.Type = strings.TrimPrefix(f[0], "+RESP:")
	case strings.HasPrefix(f[0], "+BUFF:"):
		m.Type = strings.TrimPrefix(f[0], "+BUFF:")
	default:
		return m, fmt.Errorf("unsupported header")
	}
	m.Version, m.IMEI, m.Model, m.Fields = f[1], f[2], f[3], f[4:]
	if len(m.IMEI) != 15 || strings.Trim(m.IMEI, "0123456789") != "" {
		return m, fmt.Errorf("invalid IMEI")
	}
	if m.Type != "GTSTT" {
		return m, nil
	}
	if (len(f) < 19 || len(f) > 21) || (f[1] != "80201E0100" && f[1] != "80201E0306") || (f[3] != "" && f[3] != "GV30CEU") {
		return m, fmt.Errorf("unverified GTSTT layout/version/model")
	}
	m.Status, m.Accuracy = f[4], f[5]
	// The observed layouts use bit 0 for Satellites in Use. Other masks are
	// preserved as unsupported until their optional fields are verified.
	mask, maskErr := strconv.ParseUint(f[16], 16, 8)
	if maskErr != nil || mask&^uint64(5) != 0 || (f[1] != "80201E0306" && mask > 1) || len(f) != 19+int(mask&1)+int((mask>>2)&1) {
		return m, fmt.Errorf("unverified GTSTT append mask/layout")
	}
	sendIndex := len(f) - 2
	var satellites *int
	if mask&1 != 0 && f[17] != "" {
		n, e := strconv.Atoi(f[17])
		if e != nil || n < 0 || n > 72 {
			return m, fmt.Errorf("invalid satellites")
		}
		satellites = &n
	}
	// GNSS UTC Time and Send Time are UTC according to the protocol.
	loc = time.UTC
	if f[6] == "" && f[7] == "" && f[8] == "" && f[9] == "" && f[10] == "" && f[11] == "" {
		_, err = deviceTime(f[sendIndex], loc)
		return m, err
	}
	p := &Position{Satellites: satellites, FixStatus: "unknown", AccuracyRaw: m.Accuracy}
	if f[1] == "80201E0306" && f[5] != "" {
		accuracy, e := strconv.Atoi(f[5])
		if e != nil || accuracy < 0 || accuracy > 50 {
			return m, fmt.Errorf("invalid GNSS accuracy")
		}
		p.FixStatus = "no_fix"
		if accuracy > 0 {
			p.FixStatus = "valid"
			hdop := float64(accuracy)
			p.HDOP = &hdop
		}
	}
	nums := []*float64{&p.Speed, &p.Altitude, &p.Longitude, &p.Latitude}
	for i, index := range []int{6, 8, 9, 10} {
		v, e := strconv.ParseFloat(f[index], 64)
		if e != nil || math.IsNaN(v) || math.IsInf(v, 0) {
			return m, fmt.Errorf("invalid numeric field %d", index)
		}
		*nums[i] = v
	}
	p.Heading, err = strconv.Atoi(f[7])
	if err != nil {
		return m, fmt.Errorf("invalid heading")
	}
	if p.Speed < 0 || p.Heading < 0 || p.Heading > 359 || p.Latitude < -90 || p.Latitude > 90 || p.Longitude < -180 || p.Longitude > 180 {
		return m, fmt.Errorf("position outside valid range")
	}
	p.GPSTime, err = deviceTime(f[11], loc)
	if err != nil {
		return m, err
	}
	p.SendTime, err = deviceTime(f[sendIndex], loc)
	if err != nil {
		return m, err
	}
	m.Position = p
	return m, nil
}
