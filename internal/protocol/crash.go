package protocol

import (
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

type CrashReport struct {
	MessageType, ProtocolVersion, IMEI, DeviceName string
	CrashCounter                                   uint8
	DataType                                       *uint8
	TotalFrames, FrameNumber                       *int
	AccelerationHex                                string
	TriggerDegrees                                 *int
	EngineStatus                                   *bool
	Accuracy                                       *int
	Speed, Altitude, Longitude, Latitude           *float64
	Heading                                        *int
	GPSTime                                        *time.Time
	SendTime                                       time.Time
	CountNumber                                    uint16
}

func DecodeCrash(payload string) (CrashReport, error) {
	var r CrashReport
	if !strings.HasSuffix(payload, "$") {
		return r, fmt.Errorf("incomplete crash report")
	}
	f := strings.Split(strings.TrimSuffix(strings.TrimSpace(payload), "$"), ",")
	if len(f) < 4 {
		return r, fmt.Errorf("short crash report")
	}
	head := strings.Split(f[0], ":")
	if len(head) != 2 || (head[0] != "+RESP" && head[0] != "+BUFF") {
		return r, fmt.Errorf("unsupported crash envelope")
	}
	r.MessageType = head[1]
	r.ProtocolVersion = f[1]
	r.IMEI = f[2]
	r.DeviceName = f[3]
	if r.ProtocolVersion != "80201E0306" || len(r.IMEI) != 15 || strings.Trim(r.IMEI, "0123456789") != "" {
		return r, fmt.Errorf("unverified crash protocol identity")
	}
	switch r.MessageType {
	case "GTCRA", "GTECR":
		return decodeCrashAlarm(f, r)
	case "GTCRD":
		return decodeCrashData(f, r)
	case "GTACC":
		return decodeAcceleration(f, r)
	default:
		return r, fmt.Errorf("unsupported crash message")
	}
}

func decodeCrashAlarm(f []string, r CrashReport) (CrashReport, error) {
	if len(f) < 19 {
		return r, fmt.Errorf("short crash alarm")
	}
	counter, e := hexByte(f[4])
	if e != nil {
		return r, e
	}
	r.CrashCounter = counter
	accuracy, e := strconv.Atoi(f[5])
	if e != nil || accuracy < 0 || accuracy > 50 {
		return r, fmt.Errorf("invalid crash GNSS accuracy")
	}
	r.Accuracy = &accuracy
	positionPresent := f[9] != "" || f[10] != "" || f[11] != ""
	if positionPresent {
		values := []**float64{&r.Speed, &r.Altitude, &r.Longitude, &r.Latitude}
		for i, index := range []int{6, 8, 9, 10} {
			v, err := strconv.ParseFloat(f[index], 64)
			if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
				return r, fmt.Errorf("invalid crash position")
			}
			*values[i] = &v
		}
		heading, err := strconv.Atoi(f[7])
		if err != nil || heading < 0 || heading > 359 || *r.Speed < 0 || *r.Speed > 999.9 || *r.Longitude < -180 || *r.Longitude > 180 || *r.Latitude < -90 || *r.Latitude > 90 {
			return r, fmt.Errorf("invalid crash position bounds")
		}
		r.Heading = &heading
		at, err := parseUTC(f[11])
		if err != nil {
			return r, err
		}
		r.GPSTime = &at
	}
	mask, err := strconv.ParseUint(f[16], 16, 8)
	if err != nil || mask&^uint64(5) != 0 {
		return r, fmt.Errorf("unsupported crash append mask")
	}
	optional := int(mask&1) + int((mask>>2)&1)
	tailExtra := 0
	if r.MessageType == "GTECR" {
		tailExtra = 4
	}
	if len(f) != 19+optional+tailExtra {
		return r, fmt.Errorf("unexpected crash alarm layout")
	}
	base := 17 + optional
	if r.MessageType == "GTECR" {
		degrees, err := strconv.Atoi(f[base])
		if err != nil || degrees < 0 || degrees > 180 {
			return r, fmt.Errorf("invalid crash trigger degrees")
		}
		r.TriggerDegrees = &degrees
		if f[base+1] != "0" && f[base+1] != "1" {
			return r, fmt.Errorf("invalid crash engine status")
		}
		engine := f[base+1] == "1"
		r.EngineStatus = &engine
		base += 4
	}
	return decodeCrashTail(f, base, r)
}

func decodeCrashData(f []string, r CrashReport) (CrashReport, error) {
	if len(f) != 11 {
		return r, fmt.Errorf("unexpected crash data layout")
	}
	counter, e := hexByte(f[4])
	if e != nil {
		return r, e
	}
	dataType, e := hexByte(f[5])
	if e != nil || dataType > 0x7f {
		return r, fmt.Errorf("invalid crash data type")
	}
	total, e := boundedInt(f[6], 1, 15)
	if e != nil {
		return r, e
	}
	frame, e := boundedInt(f[7], 1, total)
	if e != nil {
		return r, e
	}
	if len(f[8]) == 0 || len(f[8]) > 1200 || len(f[8])%12 != 0 {
		return r, fmt.Errorf("invalid crash acceleration data length")
	}
	if _, e = hex.DecodeString(f[8]); e != nil {
		return r, fmt.Errorf("invalid crash acceleration hex")
	}
	r.CrashCounter = counter
	r.DataType = &dataType
	r.TotalFrames = &total
	r.FrameNumber = &frame
	r.AccelerationHex = strings.ToUpper(f[8])
	return decodeCrashTail(f, 9, r)
}
func decodeAcceleration(f []string, r CrashReport) (CrashReport, error) {
	if len(f) != 10 {
		return r, fmt.Errorf("unexpected acceleration layout")
	}
	if len(f[4]) != 900 {
		return r, fmt.Errorf("GTACC requires 75 acceleration samples")
	}
	if _, e := hex.DecodeString(f[4]); e != nil {
		return r, fmt.Errorf("invalid acceleration hex")
	}
	r.AccelerationHex = strings.ToUpper(f[4])
	return decodeCrashTail(f, 8, r)
}
func decodeCrashTail(f []string, index int, r CrashReport) (CrashReport, error) {
	at, e := parseUTC(f[index])
	if e != nil {
		return r, fmt.Errorf("invalid crash send time")
	}
	count, e := strconv.ParseUint(f[index+1], 16, 16)
	if e != nil {
		return r, fmt.Errorf("invalid crash count number")
	}
	r.SendTime = at
	r.CountNumber = uint16(count)
	return r, nil
}
func parseUTC(value string) (time.Time, error) { return time.Parse("20060102150405", value) }
func hexByte(value string) (uint8, error) {
	if len(value) != 2 {
		return 0, fmt.Errorf("invalid crash counter")
	}
	v, e := strconv.ParseUint(value, 16, 8)
	return uint8(v), e
}
func boundedInt(value string, min, max int) (int, error) {
	v, e := strconv.Atoi(value)
	if e != nil || v < min || v > max {
		return 0, fmt.Errorf("integer outside crash range")
	}
	return v, nil
}

func DecodeAccelerationSamples(data string) ([][3]int16, error) {
	if len(data)%12 != 0 {
		return nil, fmt.Errorf("invalid acceleration sample width")
	}
	out := make([][3]int16, 0, len(data)/12)
	for i := 0; i < len(data); i += 12 {
		var sample [3]int16
		for axis := 0; axis < 3; axis++ {
			value, e := strconv.ParseUint(data[i+axis*4:i+axis*4+4], 16, 16)
			if e != nil {
				return nil, e
			}
			sample[axis] = int16(uint16(value))
		}
		out = append(out, sample)
	}
	return out, nil
}
