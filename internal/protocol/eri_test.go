package protocol

import (
	"strings"
	"testing"
)

// Constructed from the official field table, not an observed physical-device fixture.
func TestDocumentedERILayout(t *testing.T) {
	raw := "+RESP:GTERI,80201E0306,000000000000001,GV30CEU,00000100,,10,1,0,0.0,0,0.0,,,,0510,0001,7D06,0337D73E,01,0,0.0,0000000:00:00,,,,100,000000,,1,0,27,0,560104001A,0007,CFS300B,000000000001,1,20261103000000,0001"
	fields := strings.Split(raw, ",")
	a, e := DecodeERIAccessories(fields)
	if e != nil {
		t.Fatal(e)
	}
	if len(a) != 1 || a[0].CFS.FuelHeightRaw != 260 || a[0].Connected == nil || !*a[0].Connected {
		t.Fatal(a)
	}

	valid := strings.Replace(raw, "0,0.0,0,0.0,,,,", "2,20.0,90,10.0,106.0,-6.0,20261103000000,", 1)
	report, err := DecodeERI(strings.Split(valid, ","))
	if err != nil || len(report.Positions) != 1 || report.Positions[0].FixStatus != "valid" || report.Positions[0].HDOP == nil {
		t.Fatal("documented GPS group", report, err)
	}
	fields[4] = "00000104"
	if _, e = DecodeERIAccessories(fields); e == nil {
		t.Fatal("unsupported CAN section guessed")
	}
}

// Sanitized actual server receipt: device identity/MAC and coordinates replaced.
// Accuracy=0 and satellites=0 must stay no_fix despite nonempty last-known coordinates.
func TestActualObserved8104(t *testing.T) {
	raw := "+RESP:GTERI,80201E0306,000000000000001,GV30CEU,00008104,,10,1,0,0.0,0,18.3,106.000000,-6.000000,20261102102808,0510,0001,7D06,0337D73E,01,0,0.0,,,,,10,420000,,0,0,1,00,27,0,04125801D5,2017,CFS300B,000000000001,1,28,100,4,3,20261102103414,4917"
	report, err := DecodeERI(strings.Split(raw, ","))
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Accessories) != 1 || report.Accessories[0].CFS.FuelHeightRaw != 0x1258 || report.Accessories[0].CFS.VolumeRaw != 0x1D5 || report.Positions[0].FixStatus != "no_fix" || len(report.OpaqueCANFields) != 2 {
		t.Fatal("actual receipt mismatch")
	}
	fields := strings.Split(raw, ",")
	fields[29] = "1"
	if _, err = DecodeERI(fields); err == nil {
		t.Fatal("unverified CAN interpreted")
	}
}

func TestERIGPSIsIndependentFromInvalidSensorPayload(t *testing.T) {
	raw := "+RESP:GTERI,80201E0306,000000000000001,GV30CEU,00008104,,10,1,2,40.0,90,18.3,106.000000,-6.000000,20261104102808,0510,0001,7D06,0337D73E,01,5,0.0,,,,,10,420000,,0,0,1,00,27,0,INVALID,2017,CFS300B,000000000001,1,28,100,4,3,20261104103414,4917"
	fields := strings.Split(raw, ",")
	if _, err := DecodeERI(fields); err == nil {
		t.Fatal("invalid accessory accepted")
	}
	report, err := DecodeERIPositions(fields)
	if err != nil || len(report.Positions) != 1 || report.Positions[0].Speed != 40 || report.Positions[0].FixStatus != "valid" {
		t.Fatalf("valid GPS lost: %+v %v", report, err)
	}
	fields[13] = ""
	if _, err := DecodeERIPositions(fields); err == nil {
		t.Fatal("invalid GPS accepted")
	}
}
