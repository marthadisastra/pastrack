package protocol

import (
	"strings"
	"testing"
	"time"
)

const observedGTCRA = "+RESP:GTCRA,80201E0306,000000000000000,,00,0,,,,,,,0510,0001,7D06,0337D716,00,20261105061536,992E$"

func TestDecodeObservedGV30CEUGTCRA(t *testing.T) {
	r, err := DecodeCrash(observedGTCRA)
	if err != nil {
		t.Fatal(err)
	}
	if r.MessageType != "GTCRA" || r.CrashCounter != 0 || !r.SendTime.Equal(mustTime("20261105061536")) || r.GPSTime != nil {
		t.Fatalf("%+v", r)
	}
}
func TestDecodeDocumentedGTECR(t *testing.T) {
	raw := "+RESP:GTECR,80201E0306,000000000000000,GV30CEU,BD,0,49.1,270,36.8,117.071316,31.832131,20240416062629,0460,0000,550B,B7B1,01,10,104,0,,,20240416165742,1DE4$"
	r, err := DecodeCrash(raw)
	if err != nil {
		t.Fatal(err)
	}
	if r.CrashCounter != 0xBD || r.TriggerDegrees == nil || *r.TriggerDegrees != 104 || r.EngineStatus == nil || *r.EngineStatus {
		t.Fatalf("%+v", r)
	}
}
func TestDecodeCrashDataAndAcceleration(t *testing.T) {
	data := strings.Repeat("0001FFFF0052", 100)
	raw := "+RESP:GTCRD,80201E0306,000000000000000,,00,18,5,1," + data + ",20261105061536,992F$"
	r, err := DecodeCrash(raw)
	if err != nil {
		t.Fatal(err)
	}
	if r.TotalFrames == nil || *r.TotalFrames != 5 || r.FrameNumber == nil || *r.FrameNumber != 1 {
		t.Fatalf("%+v", r)
	}
	samples, err := DecodeAccelerationSamples(r.AccelerationHex)
	if err != nil || len(samples) != 100 || samples[0] != [3]int16{1, -1, 82} {
		t.Fatalf("samples=%v err=%v", samples[:1], err)
	}
	acc := strings.Repeat("0001FFFF0052", 75)
	r, err = DecodeCrash("+RESP:GTACC,80201E0306,000000000000000,," + acc + ",,,,20261105061536,9930$")
	if err != nil || len(r.AccelerationHex) != 900 {
		t.Fatalf("GTACC %+v %v", r, err)
	}
}
func TestRejectInvalidCrashPayloads(t *testing.T) {
	values := []string{strings.Replace(observedGTCRA, "GTCRA", "GTXYZ", 1), strings.Replace(observedGTCRA, "80201E0306", "80201E0100", 1), strings.Replace(observedGTCRA, ",00,0,", ",GG,0,", 1), "+RESP:GTCRD,80201E0306,000000000000000,,00,18,5,6,000100010052,20261105061536,992F$"}
	for _, raw := range values {
		if _, err := DecodeCrash(raw); err == nil {
			t.Fatalf("invalid accepted: %.40s", raw)
		}
	}
}
func mustTime(value string) time.Time { v, _ := time.Parse("20060102150405", value); return v }
