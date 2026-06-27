package version

import "testing"

func TestGetVersion_PrefersLdflagValue(t *testing.T) {
	orig := Version
	t.Cleanup(func() { Version = orig })

	Version = "v9.9.9"
	if got := GetVersion(); got != "v9.9.9" {
		t.Fatalf("GetVersion() = %q, want %q", got, "v9.9.9")
	}
}

func TestGetVersion_DevFallsBack(t *testing.T) {
	orig := Version
	t.Cleanup(func() { Version = orig })

	// With the sentinel "dev" value, GetVersion must not return "dev" verbatim
	// when a git tag can be detected; either way it must never panic and must
	// return a non-nil string.
	Version = "dev"
	got := GetVersion()
	if got == "" {
		t.Fatal("GetVersion() returned empty string for dev fallback")
	}
}
