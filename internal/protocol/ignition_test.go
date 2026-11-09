package protocol

import (
	"strings"
	"testing"
)

func TestIgnitionReport(t *testing.T) {
	p := "+RESP:GTIGL,80201E0306,000000000000000,GV30CEU,,01,1,5,0.0,0,65.4,106.872339,-6.133696,20261103094920,0510,0001,7D06,033CA45D,01,8,0.0,20261103094921,4E95$"
	r, e := DecodeIgnition(p)
	if e != nil || r.State != "OFF" || r.Source != "GTIGL" {
		t.Fatal(r, e)
	}
	r, e = DecodeIgnition(strings.Replace(p, ",01,1,5,", ",00,1,5,", 1))
	if e != nil || r.State != "ON" {
		t.Fatal(r, e)
	}
	if _, e = DecodeIgnition(strings.Replace(p, ",01,1,5,", ",02,1,5,", 1)); e == nil {
		t.Fatal("unknown accepted")
	}
}

func TestIgnitionDedicatedEventsAndBuffer(t *testing.T) {
	for _, kind := range []string{"GTIGN", "GTIGF"} {
		p := "+BUFF:" + kind + ",80201E0306,000000000000000,GV30CEU,1200,1,0.0,0,65.4,106.872339,-6.133696,20261103094920,0510,0001,7D06,033CA45D,01,8,0.0,20261103094921,4E95$"
		r, e := DecodeIgnition(p)
		if e != nil {
			t.Fatal(kind, e)
		}
		want := "ON"
		if strings.HasSuffix(kind, "F") {
			want = "OFF"
		}
		if r.State != want || r.At.Format("20060102150405") != "20261103094921" {
			t.Fatal(r)
		}
	}
}

func TestVirtualIgnitionDedicatedLayout(t *testing.T) {
	for _, kind := range []string{"GTVGN", "GTVGF"} {
		payload := "+RESP:" + kind + ",80201E0306,000000000000000,GV30CEU,00,2,1200,1,0.0,0,65.4,106.872339,-6.133696,20261103094920,0510,0001,7D06,033CA45D,01,8,0.0,20261103094921,4E95$"
		report, err := DecodeIgnition(payload)
		if err != nil {
			t.Fatal(err)
		}
		want := "ON"
		if kind == "GTVGF" {
			want = "OFF"
		}
		if report.State != want || report.GPSTime.Format("20060102150405") != "20261103094920" {
			t.Fatal(report)
		}
	}
}
