package database

import "testing"

func TestProductionRuntimeIsExplicit(t *testing.T) {
	if RuntimeTarget("127.0.0.1", "pastrack_v2_stage_20261103", "production") != nil {
		t.Fatal("explicit v2 rejected")
	}
	for _, name := range []string{"pastrack", "pastrack_dev_local", "pastrack_v2_test_local", "pastrack_v2_unsafe;"} {
		if RuntimeTarget("127.0.0.1", name, "production") == nil {
			t.Fatal("unsafe target accepted", name)
		}
	}
	if RuntimeTarget("127.0.0.1", "pastrack_v2_stage_20261103", "") == nil {
		t.Fatal("missing production mode bypassed local guard")
	}
}
