package protocol

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

type ERIPosition struct {
	Speed, Altitude, Longitude, Latitude float64
	Heading                              int
	GPSTime                              *time.Time
	Satellites                           *int
	Accuracy                             string
	FixStatus                            string
	HDOP                                 *float64
}
type ERIReport struct {
	Positions       []ERIPosition
	Accessories     []Accessory
	SendTime        time.Time
	OpaqueCANFields []string
	RAT             *int
	Band            *int
}
type Accessory struct {
	Index, Type, Model         int
	Raw, Name, MAC, AppendMask string
	Connected                  *bool
	BatteryMV                  *int
	TemperatureC               *float64
	RSSI                       *int
	BatteryPercent             *int
	SensorTime                 *time.Time
	CFS                        *CFS300B
}
type cursor struct {
	fields []string
	index  int
}

func (c *cursor) take() (string, error) {
	if c.index >= len(c.fields) {
		return "", fmt.Errorf("truncated ERI field %d", c.index)
	}
	s := c.fields[c.index]
	c.index++
	return s, nil
}
func (c *cursor) number(min, max int) (int, error) {
	s, e := c.take()
	if e != nil {
		return 0, e
	}
	n, e := strconv.Atoi(s)
	if e != nil || n < min || n > max {
		return 0, fmt.Errorf("invalid ERI number at field %d", c.index-1)
	}
	return n, nil
}
func (c *cursor) skip(n int) error {
	for i := 0; i < n; i++ {
		if _, e := c.take(); e != nil {
			return e
		}
	}
	return nil
}

// DecodeERIAccessories follows the ASCII ERI tables in v0306 pp198-205.
// It rejects optional sections it cannot safely skip. No marker-based guessing.
// Position decoding is separate: this function does not claim a GPS fix.
func DecodeERIAccessories(fields []string) ([]Accessory, error) {
	report, err := DecodeERI(fields)
	if err != nil {
		return nil, err
	}
	return report.Accessories, nil
}
func DecodeERI(fields []string) (*ERIReport, error) {
	if len(fields) < 10 || fields[1] != "80201E0306" {
		return nil, fmt.Errorf("unverified ERI version")
	}
	mask, e := strconv.ParseUint(fields[4], 16, 32)
	if e != nil || (mask != 0x100 && mask != 0x8104) {
		return nil, fmt.Errorf("unsupported ERI sections mask")
	}
	report, c, e := decodeERIPositionGroup(fields)
	if e != nil {
		return nil, e
	}
	// Mileage, hour meter, analog voltage, reserved x2, backup%, status, reserved.
	if e = c.skip(8); e != nil {
		return nil, e
	}
	// Actual GV30CEU receipts use 0x8104 with the exact opaque CAN token pair
	// "0,0". Retain those tokens without assigning undocumented meanings. Any
	// different CAN body stays unsupported rather than shifting BLE field offsets.
	if mask&4 != 0 {
		for i := 0; i < 2; i++ {
			value, err := c.take()
			if err != nil || value != "0" {
				return nil, fmt.Errorf("unverified CAN section layout")
			}
			report.OpaqueCANFields = append(report.OpaqueCANFields, value)
		}
	}
	accessories, e := c.number(0, 10)
	if e != nil {
		return nil, e
	}
	out := []Accessory{}
	for i := 0; i < accessories; i++ {
		a := Accessory{}
		a.Index, e = c.number(0, 9)
		if e != nil {
			return nil, e
		}
		a.Type, e = c.number(0, 27)
		if e != nil {
			return nil, e
		}
		a.Model, e = c.number(0, 5)
		if e != nil {
			return nil, e
		}
		a.Raw, e = c.take()
		if e != nil {
			return nil, e
		}
		a.AppendMask, e = c.take()
		if e != nil {
			return nil, e
		}
		bits, e := strconv.ParseUint(a.AppendMask, 16, 16)
		if e != nil {
			return nil, fmt.Errorf("extended/unverified accessory append mask")
		}
		// Supported flags have exactly one documented field each. Reserved/relay/
		// extension flags require additional documentation/fixtures and are rejected.
		const supported = uint64(1<<0 | 1<<1 | 1<<2 | 1<<3 | 1<<4 | 1<<5 | 1<<9 | 1<<10 | 1<<11 | 1<<13)
		if bits&^supported != 0 {
			return nil, fmt.Errorf("unsupported accessory optional fields")
		}
		for bit := uint(0); bit < 16; bit++ {
			if bits&(1<<bit) == 0 {
				continue
			}
			value, e := c.take()
			if e != nil {
				return nil, e
			}
			if value == "" {
				continue
			}
			switch bit {
			case 0:
				a.Name = value
			case 1:
				if len(value) != 12 || strings.Trim(strings.ToUpper(value), "0123456789ABCDEF") != "" {
					return nil, fmt.Errorf("invalid accessory MAC")
				}
				a.MAC = strings.ToUpper(value)
			case 2:
				if value != "0" && value != "1" {
					return nil, fmt.Errorf("invalid accessory status")
				}
				v := value == "1"
				a.Connected = &v
			case 3:
				n, e := strconv.Atoi(value)
				if e != nil || n < 0 || n > 5000 {
					return nil, fmt.Errorf("invalid accessory voltage")
				}
				a.BatteryMV = &n
			case 4, 11:
				n, e := strconv.ParseFloat(value, 64)
				if e != nil || math.IsNaN(n) || math.IsInf(n, 0) || n < -40 || n > 80 {
					return nil, fmt.Errorf("invalid accessory temperature")
				}
				a.TemperatureC = &n
			case 5:
				n, e := strconv.Atoi(value)
				if e != nil || n < 0 || n > 100 {
					return nil, fmt.Errorf("invalid humidity")
				}
			case 9:
				n, e := strconv.Atoi(value)
				if e != nil || n < 0 || n > 500 {
					return nil, fmt.Errorf("invalid tire pressure")
				}
			case 10:
				v, e := time.Parse("20060102150405", value)
				if e != nil {
					return nil, fmt.Errorf("invalid sensor time")
				}
				a.SensorTime = &v
			case 13:
				n, e := strconv.Atoi(value)
				if e != nil || n < 0 || n > 100 {
					return nil, fmt.Errorf("invalid battery percent")
				}
				a.BatteryPercent = &n
			}
		}
		if a.Type == 27 && a.Model == 0 {
			value, e := DecodeCFS300B(a.Raw)
			if e != nil {
				return nil, e
			}
			a.CFS = &value
		}
		out = append(out, a)
	}
	if mask&0x8000 != 0 {
		rat, err := c.number(0, 4)
		if err != nil || (rat != 0 && rat != 1 && rat != 4) {
			return nil, fmt.Errorf("invalid RAT")
		}
		band, err := c.number(0, 1900)
		if err != nil {
			return nil, err
		}
		valid := map[int]bool{0: true, 1: true, 2: true, 3: true, 4: true, 5: true, 7: true, 8: true, 20: true, 28: true, 850: true, 900: true, 1800: true, 1900: true}
		if !valid[band] {
			return nil, fmt.Errorf("invalid radio band")
		}
		report.RAT = &rat
		report.Band = &band
	}
	if c.index != len(fields)-2 {
		return nil, fmt.Errorf("unverified ERI trailing fields")
	}
	send, e := c.take()
	if e != nil {
		return nil, e
	}
	if report.SendTime, e = time.Parse("20060102150405", send); e != nil {
		return nil, fmt.Errorf("invalid ERI send time")
	}
	seq, e := c.take()
	if e != nil || len(seq) != 4 {
		return nil, fmt.Errorf("invalid ERI sequence")
	}
	if _, e = strconv.ParseUint(seq, 16, 16); e != nil {
		return nil, fmt.Errorf("invalid ERI sequence")
	}
	report.Accessories = out
	return report, nil
}

func decodeERIPositionGroup(fields []string) (*ERIReport, cursor, error) {
	report := &ERIReport{}
	var e error
	c := cursor{fields: fields, index: 7}
	count, e := c.number(1, 15)
	if e != nil {
		return nil, c, e
	}
	for i := 0; i < count; i++ {
		// GPS fields are decoded from the documented per-position group.
		start := c.index
		if e = c.skip(11); e != nil {
			return nil, c, e
		}
		f := fields[start:c.index]
		var position *ERIPosition
		if f[4] != "" || f[5] != "" {
			p := ERIPosition{Accuracy: f[0], FixStatus: "unknown"}
			accuracy, err := strconv.Atoi(f[0])
			if err != nil || accuracy < 0 || accuracy > 50 {
				return nil, c, fmt.Errorf("invalid ERI GNSS accuracy")
			}
			p.FixStatus = "no_fix"
			if accuracy > 0 {
				p.FixStatus = "valid"
				v := float64(accuracy)
				p.HDOP = &v
			}
			for j, target := range []*float64{&p.Speed, &p.Altitude, &p.Longitude, &p.Latitude} {
				index := []int{1, 3, 4, 5}[j]
				v, e := strconv.ParseFloat(f[index], 64)
				if e != nil || math.IsNaN(v) || math.IsInf(v, 0) {
					return nil, c, fmt.Errorf("unavailable/invalid ERI position numeric")
				}
				*target = v
			}
			p.Heading, e = strconv.Atoi(f[2])
			if e != nil || p.Heading < 0 || p.Heading > 359 || p.Speed < 0 || p.Speed > 999.9 || p.Longitude < -180 || p.Longitude > 180 || p.Latitude < -90 || p.Latitude > 90 {
				return nil, c, fmt.Errorf("invalid ERI position bounds")
			}
			at, e := time.Parse("20060102150405", f[6])
			if e != nil {
				return nil, c, fmt.Errorf("invalid ERI GNSS time")
			}
			p.GPSTime = &at
			position = &p
		}
		text, e := c.take()
		if e != nil {
			return nil, c, e
		}
		appendMask, e := strconv.ParseUint(text, 16, 8)
		if e != nil || appendMask&^uint64(3) != 0 {
			return nil, c, fmt.Errorf("unsupported ERI position append mask")
		}
		if appendMask&1 != 0 {
			sat, err := c.number(0, 72)
			if err != nil {
				return nil, c, err
			}
			if position != nil {
				position.Satellites = &sat
			}
		}
		if appendMask&2 != 0 {
			if _, e = c.number(0, 4); e != nil {
				return nil, c, e
			}
		}
		if position != nil {
			report.Positions = append(report.Positions, *position)
		}
	}

	return report, c, nil
}

// DecodeERIPositions validates the GPS group and report send time even when
// accessory payloads are unsupported.
func DecodeERIPositions(fields []string) (*ERIReport, error) {
	if len(fields) < 10 || fields[1] != "80201E0306" {
		return nil, fmt.Errorf("unverified ERI version")
	}
	mask, err := strconv.ParseUint(fields[4], 16, 32)
	if err != nil || (mask != 0x100 && mask != 0x8104) {
		return nil, fmt.Errorf("unsupported ERI sections mask")
	}
	report, _, err := decodeERIPositionGroup(fields)
	if err != nil {
		return nil, err
	}
	sent, err := time.Parse("20060102150405", fields[len(fields)-2])
	if err != nil {
		return nil, fmt.Errorf("invalid ERI send time")
	}
	report.SendTime = sent
	return report, nil
}
