package main

import "strconv"

// observedFuelValue is an experimental signal, not a protocol-verified fuel level.
// In observed GV30CEU GTERI reports, the 16-bit word inside the ten hex digits
// two fields before a CFS300B marker rose as the probe was immersed in water.
func observedFuelValue(m Message, marker int) (int, bool) {
	if m.Type != "GTERI" || m.Version != "80201E0306" || m.Model != "GV30CEU" || marker < 2 || marker >= len(m.Fields) || m.Fields[marker] != "CFS300B" {
		return 0, false
	}
	word := m.Fields[marker-2]
	if len(word) != 10 {
		return 0, false
	}
	for _, ch := range word {
		if !((ch >= '0' && ch <= '9') || (ch >= 'A' && ch <= 'F') || (ch >= 'a' && ch <= 'f')) {
			return 0, false
		}
	}
	v, err := strconv.ParseUint(word[2:6], 16, 16)
	return int(v), err == nil
}
