package runtime

import "testing"

func TestCurrent(t *testing.T) {
	cases := []struct {
		env  string
		want Role
	}{
		{"", RoleAll},
		{"all", RoleAll},
		{"ALL", RoleAll},
		{" all ", RoleAll},
		{"api", RoleAPI},
		{"API", RoleAPI},
		{"worker", RoleWorker},
		{"Worker", RoleWorker},
		{"bogus", RoleAll},
	}
	for _, tc := range cases {
		t.Run(tc.env, func(t *testing.T) {
			t.Setenv(envVar, tc.env)
			if got := Current(); got != tc.want {
				t.Fatalf("Current()=%q want %q", got, tc.want)
			}
		})
	}
}

func TestIsAPIIsWorker(t *testing.T) {
	cases := []struct {
		env             string
		wantAPI, wantWk bool
	}{
		{"", true, true},
		{"all", true, true},
		{"api", true, false},
		{"worker", false, true},
		{"junk", true, true},
	}
	for _, tc := range cases {
		t.Run(tc.env, func(t *testing.T) {
			t.Setenv(envVar, tc.env)
			if got := IsAPI(); got != tc.wantAPI {
				t.Fatalf("IsAPI()=%v want %v", got, tc.wantAPI)
			}
			if got := IsWorker(); got != tc.wantWk {
				t.Fatalf("IsWorker()=%v want %v", got, tc.wantWk)
			}
		})
	}
}
