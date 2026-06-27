package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeConfig writes a .spot_config into a temp HOME and points HOME at it.
func writeConfig(t *testing.T, contents string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if contents != "" {
		if err := os.WriteFile(filepath.Join(home, ".spot_config"), []byte(contents), 0600); err != nil {
			t.Fatalf("write config: %v", err)
		}
	}
}

func TestLoad_MissingConfig(t *testing.T) {
	writeConfig(t, "") // no .spot_config at all
	_, err := Load(context.Background(), LoadOptions{})
	if err == nil {
		t.Fatal("expected error when config is missing")
	}
	if !strings.Contains(err.Error(), "configure") {
		t.Fatalf("error should mention 'configure', got: %v", err)
	}
}

func TestLoad_RequireOrg(t *testing.T) {
	writeConfig(t, "region: ord\n") // org not set
	_, err := Load(context.Background(), LoadOptions{RequireOrg: true})
	if err == nil || !strings.Contains(err.Error(), "organization") {
		t.Fatalf("expected organization-required error, got: %v", err)
	}
}

func TestLoad_RequireRegion(t *testing.T) {
	writeConfig(t, "org: my-org\n") // region not set
	_, err := Load(context.Background(), LoadOptions{RequireRegion: true})
	if err == nil || !strings.Contains(err.Error(), "region") {
		t.Fatalf("expected region-required error, got: %v", err)
	}
}

func TestLoad_OptsOverrideConfigForRequireChecks(t *testing.T) {
	// Config has neither org nor region, but the caller supplies both via opts,
	// so the Require* checks must pass (the error, if any, then comes from client
	// creation/auth which we don't reach assertions on here).
	writeConfig(t, "refreshToken: x\n")
	_, err := Load(context.Background(), LoadOptions{
		Org: "o", Region: "r", RequireOrg: true, RequireRegion: true,
	})
	if err != nil && (strings.Contains(err.Error(), "organization not specified") ||
		strings.Contains(err.Error(), "region not specified")) {
		t.Fatalf("Require* checks should pass when opts provide values, got: %v", err)
	}
}
