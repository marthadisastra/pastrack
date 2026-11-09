package protocol

import "testing"

func TestObservedCFS300BWords(t *testing.T) {
	v, e := DecodeCFS300B("560104001A")
	if e != nil || v.TiltRaw != 0x56 || v.FuelHeightRaw != 260 || v.VolumeRaw != 26 {
		t.Fatal(v, e)
	}
	for _, p := range []string{"", "560104001", "560104001A00", "56010Z001A"} {
		if _, e = DecodeCFS300B(p); e == nil {
			t.Fatal("invalid accepted")
		}
	}
}
