package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const sample = "+RESP:GTSTT,80201E0100,861252080093813,GV30CEU,41,0,0.6,64,24.9,106.872516,-6.133900,20261028061901,0510,0001,7D06,0337D73E,01,7,20261028100958,0422$"

func TestObservedGTSTT(t *testing.T) {
	m, e := parseMessage(sample, time.UTC)
	if e != nil {
		t.Fatal(e)
	}
	if m.IMEI != "861252080093813" || m.Position == nil || m.Position.Latitude != -6.133900 || m.Position.SendTime.Format(time.RFC3339) != "2026-10-28T10:09:58Z" {
		t.Fatalf("wrong parsed message: %+v", m)
	}
}
func TestObservedGTSTTWithoutGPSFix(t *testing.T) {
	raw := "+RESP:GTSTT,80201E0306,861252080093110,GV30CEU,41,0,,,,,,,0510,0001,7D06,0337D73E,01,0,20261102065406,3E08$"
	m, err := parseMessage(raw, time.UTC)
	if err != nil || m.Position != nil || m.IMEI != "861252080093110" {
		t.Fatalf("no-fix report must be retained as partial: %+v %v", m, err)
	}
}
func TestInvalidAndUnverified(t *testing.T) {
	cases := []string{strings.Join(strings.Split(sample, ",")[:12], ",") + "$", strings.Replace(sample, "0.6", "NaN", 1), strings.Replace(sample, "-6.133900", "-96", 1), strings.Replace(sample, "80201E0100", "80201E0200", 1), strings.Replace(sample, "20261028061901", "20260230061901", 1), strings.TrimSuffix(sample, "$")}
	for _, s := range cases {
		if _, e := parseMessage(s, time.UTC); e == nil {
			t.Errorf("accepted invalid/unverified message: %s", s)
		}
	}
}
func TestUnknownFuelPreservesFields(t *testing.T) {
	s := "+RESP:GTXXX,80201E0100,861252080093813,GV30CEU,CFS300B,123.45$"
	m, e := parseMessage(s, time.UTC)
	if e != nil || m.Position != nil || strings.Join(m.Fields, ",") != "CFS300B,123.45" {
		t.Fatalf("raw fuel fields lost: %+v %v", m, e)
	}
}
func TestBufferedMessage(t *testing.T) {
	if _, e := parseMessage(strings.Replace(sample, "+RESP:", "+BUFF:", 1), time.UTC); e != nil {
		t.Fatal(e)
	}
}
func TestFrameStream(t *testing.T) {
	// MultiReader simulates TCP fragments, followed by a coalesced second frame.
	r := bufio.NewReaderSize(io.MultiReader(strings.NewReader(sample[:17]), strings.NewReader(sample[17:]+sample)), 16)
	for i := 0; i < 2; i++ {
		s, e := readFrame(r)
		if e != nil || s != sample {
			t.Fatalf("bad frame %d: %v", i, e)
		}
	}
	if _, e := readFrame(r); e != io.EOF {
		t.Fatal(e)
	}
	if _, e := readFrame(bufio.NewReader(bytes.NewBufferString(strings.Repeat("X", maxFrame+1) + "$"))); e == nil {
		t.Fatal("oversized frame accepted")
	}
}
func TestDurableSpool(t *testing.T) {
	dir := t.TempDir()
	r := Record{Payload: sample, Received: time.Now().UTC(), Peer: "test"}
	if e := enqueue(dir, r); e != nil {
		t.Fatal(e)
	}
	files, e := filepath.Glob(filepath.Join(dir, "*.json"))
	if e != nil || len(files) != 1 {
		t.Fatal(files, e)
	}
	b, e := os.ReadFile(files[0])
	if e != nil {
		t.Fatal(e)
	}
	var got Record
	if e = json.Unmarshal(b, &got); e != nil || got.Payload != sample {
		t.Fatal(e)
	}
	info, _ := os.Stat(files[0])
	if info.Mode().Perm() != 0600 {
		t.Fatalf("insecure spool permissions: %v", info.Mode())
	}
}

func TestGPSHealthReportsWithoutDeviceName(t *testing.T) {
	if m, err := parseMessage(strings.Replace(sample, ",GV30CEU,", ",,", 1), time.UTC); err != nil || m.Position == nil || m.Fields[8] != "0510" {
		t.Fatalf("unnamed GTSTT: %+v %v", m, err)
	}
	for _, tt := range []struct {
		kind, field string
		voltage     bool
	}{
		{"GTBPL", "3.62,", true}, {"GTBTC", "", false},
	} {
		raw := "+RESP:" + tt.kind + ",80201E0306,123456789012345,," + tt.field + "0,4.3,92,70.0,121.354335,31.222073,20261104102900,0460,0000,18D8,6141,05,1,20261104102900,11F0$"
		m, err := parseMessage(raw, time.UTC)
		if err != nil {
			t.Fatal(err)
		}
		sent, volts, err := gpsBatteryReport(m)
		if err != nil || sent.Format(time.RFC3339) != "2026-11-04T10:29:00Z" || (volts != nil) != tt.voltage || (volts != nil && *volts != 3.62) {
			t.Fatalf("battery report %s: %v %v %v", tt.kind, sent, volts, err)
		}
	}
}

func TestACKRetainsDeviceIdentity(t *testing.T) {
	m, err := parseMessage("+ACK:GTDOS,80201E0306,123456789012345,,000A,20261105094921,4E95$", time.UTC)
	if err != nil || m.IMEI != "123456789012345" || m.Type != "ACK" {
		t.Fatalf("ACK identity lost: %+v %v", m, err)
	}
}
