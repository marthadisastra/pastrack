package platform

import (
	"reflect"
	"testing"
)

func TestLiveDeltaOnlyChangedFields(t *testing.T) {
	old := liveRows{Rows: map[string]map[string]any{"1": {"id": 1, "speed": 10, "fuel": 50}, "2": {"id": 2}}, Order: []string{"1", "2"}}
	next := liveRows{Rows: map[string]map[string]any{"1": {"id": 1, "speed": 11, "fuel": 50}, "3": {"id": 3, "fuel": nil}}, Order: []string{"3", "1"}}
	p := diffLive(old, next)
	if !reflect.DeepEqual(p.Upserts["1"], map[string]any{"speed": 11}) {
		t.Fatal(p)
	}
	if len(p.Removed) != 1 || p.Removed[0] != "2" {
		t.Fatal(p)
	}
	if !reflect.DeepEqual(p.Order, next.Order) {
		t.Fatal(p)
	}
	unchanged := diffLive(next, next)
	if len(unchanged.Upserts) != 0 || len(unchanged.Removed) != 0 || unchanged.Order != nil {
		t.Fatal("unchanged values broadcast", unchanged)
	}
}

func TestLivePaginationMetadataChangesWithoutChangingRows(t *testing.T) {
	state := liveRows{Rows: map[string]map[string]any{"1": {"id": 1}}, Order: []string{"1"}, Page: map[string]any{"has_more": true, "next_cursor": "1"}}
	next := liveRows{Rows: state.Rows, Order: state.Order, Page: map[string]any{"has_more": false, "next_cursor": nil}}
	patch := diffLive(state, next)
	if patch.Page == nil || patch.Page["has_more"] != false || len(patch.Upserts) != 0 {
		t.Fatal(patch)
	}
}
