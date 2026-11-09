package database

import "testing"

func TestMigrationDestinationGuard(t *testing.T) {
	for _, v := range [][2]string{{"76.13.195.48", "pastrack_v2_local"}, {"127.0.0.1", "pastrack"}, {"localhost", "pastrack_;_local"}} {
		if LocalTarget(v[0], v[1]) == nil {
			t.Fatal("unsafe target accepted")
		}
	}
	if LocalTarget("127.0.0.1", "pastrack_v2_local") != nil {
		t.Fatal("local rejected")
	}
}
