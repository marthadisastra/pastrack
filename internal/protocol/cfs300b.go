// Package protocol contains decoders whose field meaning is backed by device documentation.
package protocol

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// CFS300B is the five-byte raw accessory value documented in GV30CEU v0306 p202.
// The manual does not specify scaling/units: none of these fields is liters.
type CFS300B struct {
	TiltRaw       uint8  `json:"tilt_raw"`
	FuelHeightRaw uint16 `json:"fuel_height_raw"`
	VolumeRaw     uint16 `json:"volume_raw"`
}

func DecodeCFS300B(payload string) (CFS300B, error) {
	var value CFS300B
	if len(payload) != 10 {
		return value, fmt.Errorf("CFS300B requires exactly five hex bytes")
	}
	b, err := hex.DecodeString(payload)
	if err != nil {
		return value, fmt.Errorf("invalid CFS300B hex")
	}
	value.TiltRaw = b[0]
	value.FuelHeightRaw = binary.BigEndian.Uint16(b[1:3])
	value.VolumeRaw = binary.BigEndian.Uint16(b[3:5])
	return value, nil
}
