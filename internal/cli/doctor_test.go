package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/laidbackgeek/envonce/internal/config"
	"github.com/laidbackgeek/envonce/internal/paths"
	"github.com/stretchr/testify/assert"
)

func TestRunDoctor_Reports(t *testing.T) {
	setupHome(t)
	results := RunDoctor()
	assert.NotEmpty(t, results)
	// each result has Name and OK/Message
	for _, r := range results {
		assert.NotEmpty(t, r.Name)
	}
	// the three basic checks should appear: initialized / brew / security
	names := map[string]bool{}
	for _, r := range results {
		names[r.Name] = true
	}
	assert.True(t, names["initialized"], "missing initialized check")
	assert.True(t, names["brew"], "missing brew check")
	assert.True(t, names["security"], "missing security check")
}

func TestRunDoctor_DetectsBrewPlistCollision(t *testing.T) {
	home := setupHome(t)

	// register a managed service ollama in config.toml
	cfgPath, err := paths.ConfigFile()
	assert.NoError(t, err)
	cfg, err := config.Load(cfgPath)
	assert.NoError(t, err)
	cfg.Services["ollama"] = config.ServiceDef{Source: "brew", Binary: "/opt/homebrew/bin/ollama"}
	assert.NoError(t, cfg.Save(cfgPath))

	// fake a brew leftover plist under the temp HOME
	launchAgentsDir := filepath.Join(home, "Library", "LaunchAgents")
	assert.NoError(t, os.MkdirAll(launchAgentsDir, 0o755))
	brewPlist := filepath.Join(launchAgentsDir, "homebrew.mxcl.ollama.plist")
	assert.NoError(t, os.WriteFile(brewPlist, []byte("<plist/>"), 0o644))

	results := RunDoctor()

	// there must be a non-OK brew-plist-collision:ollama entry
	var collision *CheckResult
	for i := range results {
		if results[i].Name == "brew-plist-collision:ollama" && !results[i].OK {
			collision = &results[i]
			break
		}
	}
	if collision == nil {
		t.Fatalf("expected a non-OK brew-plist-collision:ollama check; got %+v", results)
	}
	assert.Contains(t, collision.Message, brewPlist)
}

func TestRunDoctor_DetectsXDGDrift(t *testing.T) {
	setupHome(t)
	// .initialized records an old path different from the current ConfigDir
	marker, err := paths.InitializedMarker()
	assert.NoError(t, err)
	assert.NoError(t, os.MkdirAll(filepath.Dir(marker), 0o755))
	assert.NoError(t, os.WriteFile(marker, []byte("/old/config/envonce"), 0o644))

	results := RunDoctor()
	var drift *CheckResult
	for i := range results {
		if results[i].Name == "xdg-drift" && !results[i].OK {
			drift = &results[i]
			break
		}
	}
	if drift == nil {
		t.Fatalf("expected a non-OK xdg-drift check; got %+v", results)
	}
	assert.Contains(t, drift.Message, "/old/config/envonce")
}

func TestRunDoctor_NoDriftWhenConsistent(t *testing.T) {
	setupHome(t)
	// .initialized records the current ConfigDir → no drift warning expected
	cur, err := paths.ConfigDir()
	assert.NoError(t, err)
	marker, err := paths.InitializedMarker()
	assert.NoError(t, err)
	assert.NoError(t, os.MkdirAll(filepath.Dir(marker), 0o755))
	assert.NoError(t, os.WriteFile(marker, []byte(cur), 0o644))

	for _, r := range RunDoctor() {
		if r.Name == "xdg-drift" {
			t.Fatalf("xdg-drift should not warn when paths match, got %+v", r)
		}
	}
}

func TestRunDoctor_NoDriftForLegacyEmptyMarker(t *testing.T) {
	setupHome(t)
	// an empty .initialized from an older init → not checked (backward compat, no warning)
	marker, err := paths.InitializedMarker()
	assert.NoError(t, err)
	assert.NoError(t, os.MkdirAll(filepath.Dir(marker), 0o755))
	assert.NoError(t, os.WriteFile(marker, []byte{}, 0o644))

	for _, r := range RunDoctor() {
		if r.Name == "xdg-drift" {
			t.Fatalf("an empty (legacy) marker should not trigger xdg-drift, got %+v", r)
		}
	}
}
