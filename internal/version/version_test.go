package version

import "testing"

func TestVersionHasDevelopmentDefault(t *testing.T) {
	if Version != "dev" {
		t.Fatalf("Version = %q, want %q", Version, "dev")
	}
}
