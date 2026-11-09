package main

import (
	"strings"
	"testing"
	"time"
)

func TestObservedFuelSignal(t *testing.T) {
	for _, tc := range []struct {
		word string
		want int
	}{{"020104001A", 260}, {"07138001F3", 4992}} {
		raw := "+RESP:GTERI,80201E0306,861252080093110,GV30CEU,00008104,,10,1,0,,,,,,,0510,0001,7D06,0337D73E,01,0,0.0,,,,,100,410000,,0,0,1,00,27,0," + tc.word + ",2017,CFS300B,E04E7A648774,1,24,100,4,3,20261102073826,4044$"
		m, err := parseMessage(raw, time.UTC)
		if err != nil {
			t.Fatal(err)
		}
		marker := -1
		for i, v := range m.Fields {
			if v == "CFS300B" {
				marker = i
				break
			}
		}
		got, ok := observedFuelValue(m, marker)
		if !ok || got != tc.want {
			t.Fatalf("%s => %d %v", tc.word, got, ok)
		}
		m.Fields[marker-2] = "invalid"
		if _, ok := observedFuelValue(m, marker); ok {
			t.Fatal("invalid candidate accepted")
		}
		m.Fields[marker-2] = tc.word
		m.Version = "unverified"
		if _, ok := observedFuelValue(m, marker); ok {
			t.Fatal("unverified version accepted")
		}
		if strings.TrimSpace(raw) == "" {
			t.Fatal("empty sample")
		}
	}
}
