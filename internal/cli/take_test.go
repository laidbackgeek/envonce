package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/laidbackgeek/envonce/internal/brew"
	"github.com/laidbackgeek/envonce/internal/config"
	"github.com/laidbackgeek/envonce/internal/paths"
	"github.com/stretchr/testify/assert"
)

// fakeBrew replaces the real brew.Client so tests don't invoke brew.
type fakeBrew struct {
	info    *brew.ServiceInfo
	stopped string
	started string
}

func (f *fakeBrew) ImportService(name string) (*brew.ServiceInfo, error) {
	f.info.Name = name
	return f.info, nil
}
func (f *fakeBrew) StopService(name string) error  { f.stopped = name; return nil }
func (f *fakeBrew) StartService(name string) error { f.started = name; return nil }

// loadCfgMust reads the current config.toml, fataling on error; test-only.
func loadCfgMust(t *testing.T) *config.Config {
	t.Helper()
	cfg, err := loadCfg()
	assert.NoError(t, err)
	return cfg
}

func TestServiceTake_Ollama(t *testing.T) {
	setupHome(t)
	fl := &fakeLaunchd{loaded: map[string]bool{}}
	fb := &fakeBrew{info: &brew.ServiceInfo{
		Binary: "/opt/homebrew/opt/ollama/bin/ollama", Args: []string{"serve"},
		KeepAlive: true, RunAtLoad: true,
	}}
	root := newRootCmd(deps{launchd: fl, brew: fb})
	root.SetArgs([]string{"service", "take", "ollama"})
	assert.NoError(t, root.Execute())

	assert.Equal(t, "ollama", fb.stopped) // brew services stop was called
	cfg := loadCfgMust(t)
	assert.Equal(t, "brew", cfg.Services["ollama"].Source)
	assert.Equal(t, []string{"serve"}, cfg.Services["ollama"].Args)
	// the wrapper was generated
	matches, err := filepath.Glob(filepath.Join(mustConfigDir(t), "services", "ollama.wrapper.sh"))
	assert.NoError(t, err)
	assert.Len(t, matches, 1)
}

// TestServiceDrop_RemovesPlistAndWrapper verifies that drop removes the plist and wrapper files.
func TestServiceDrop_RemovesPlistAndWrapper(t *testing.T) {
	setupHome(t)
	fl := &fakeLaunchd{loaded: map[string]bool{}}

	// First add a service (via add, to avoid depending on brew)
	root := newRootCmd(deps{launchd: fl, brew: brew.New()})
	root.SetArgs([]string{"service", "add", "myapp", "--binary", "/usr/bin/true"})
	assert.NoError(t, root.Execute())

	label := fl.LabelFor("myapp")
	plistPath := paths.PlistPath(label)
	wrapperPath, err := paths.WrapperPath("myapp")
	assert.NoError(t, err)

	// both files should exist (add generates them via syncService)
	_, err = os.Stat(plistPath)
	assert.NoError(t, err)
	_, err = os.Stat(wrapperPath)
	assert.NoError(t, err)

	// drop (reuse the same fl instance, matching the original global semantics)
	root2 := newRootCmd(deps{launchd: fl, brew: brew.New()})
	root2.SetArgs([]string{"service", "drop", "myapp"})
	assert.NoError(t, root2.Execute())

	// both plist + wrapper should be gone
	_, err = os.Stat(plistPath)
	assert.True(t, errors.Is(err, fs.ErrNotExist), "plist should be removed after drop")
	_, err = os.Stat(wrapperPath)
	assert.True(t, errors.Is(err, fs.ErrNotExist), "wrapper should be removed after drop")

	// and removed from config too
	cfg := loadCfgMust(t)
	_, exists := cfg.Services["myapp"]
	assert.False(t, exists)
}

// TestServiceTake_MigratesEnvVars verifies that take migrates the brew plist's
// EnvironmentVariables into the service's same-named group and makes the service reference it.
func TestServiceTake_MigratesEnvVars(t *testing.T) {
	setupHome(t)
	fl := &fakeLaunchd{loaded: map[string]bool{}}
	fb := &fakeBrew{info: &brew.ServiceInfo{
		Binary: "/opt/homebrew/opt/ollama/bin/ollama", Args: []string{"serve"},
		KeepAlive: true, RunAtLoad: true,
		Env: map[string]string{"OLLAMA_FLASH_ATTENTION": "1", "OLLAMA_KV_CACHE_TYPE": "q8_0"},
	}}
	root := newRootCmd(deps{launchd: fl, brew: fb})
	root.SetArgs([]string{"service", "take", "ollama"})
	assert.NoError(t, root.Execute())

	// the same-named group file is created, containing both migrated vars
	d, err := paths.EnvDir()
	assert.NoError(t, err)
	data, err := os.ReadFile(filepath.Join(d, "ollama.env"))
	assert.NoError(t, err)
	body := string(data)
	assert.Contains(t, body, "OLLAMA_FLASH_ATTENTION=1")
	assert.Contains(t, body, "OLLAMA_KV_CACHE_TYPE=q8_0")

	// the service references the ollama group
	cfg := loadCfgMust(t)
	assert.Contains(t, cfg.Services["ollama"].Groups, "ollama")
}

// TestServiceTake_DoesNotOverwriteExistingEnv verifies that migration doesn't overwrite same-named keys the user already configured in the group.
func TestServiceTake_DoesNotOverwriteExistingEnv(t *testing.T) {
	setupHome(t)
	fl := &fakeLaunchd{loaded: map[string]bool{}}
	// pre-create the ollama group with a user-defined OLLAMA_FLASH_ATTENTION
	d, err := paths.EnvDir()
	assert.NoError(t, err)
	assert.NoError(t, os.MkdirAll(d, 0o755))
	assert.NoError(t, os.WriteFile(filepath.Join(d, "ollama.env"), []byte("OLLAMA_FLASH_ATTENTION=user-set\n"), 0o644))

	fb := &fakeBrew{info: &brew.ServiceInfo{
		Binary: "/opt/homebrew/opt/ollama/bin/ollama", Args: []string{"serve"},
		Env: map[string]string{"OLLAMA_FLASH_ATTENTION": "1", "OLLAMA_KV_CACHE_TYPE": "q8_0"},
	}}
	root := newRootCmd(deps{launchd: fl, brew: fb})
	root.SetArgs([]string{"service", "take", "ollama"})
	assert.NoError(t, root.Execute())

	data, err := os.ReadFile(filepath.Join(d, "ollama.env"))
	assert.NoError(t, err)
	body := string(data)
	assert.Contains(t, body, "OLLAMA_FLASH_ATTENTION=user-set", "should not overwrite the user-configured value")
	assert.Contains(t, body, "OLLAMA_KV_CACHE_TYPE=q8_0", "new key should be migrated")
}

// TestServiceDrop_RestoreBrew_CallsStart verifies that `drop --restore-brew` on a
// brew-origin service actually runs `brew services start` (hands it back to brew).
func TestServiceDrop_RestoreBrew_CallsStart(t *testing.T) {
	setupHome(t)
	fl := &fakeLaunchd{loaded: map[string]bool{}}
	fb := &fakeBrew{info: &brew.ServiceInfo{
		Binary: "/opt/homebrew/opt/ollama/bin/ollama", Args: []string{"serve"},
	}}

	// take ollama from brew first (sets source = "brew")
	root := newRootCmd(deps{launchd: fl, brew: fb})
	root.SetArgs([]string{"service", "take", "ollama"})
	assert.NoError(t, root.Execute())

	fb.started = "" // reset before drop

	// drop --restore-brew should hand back to brew via brew services start
	root2 := newRootCmd(deps{launchd: fl, brew: fb})
	root2.SetArgs([]string{"service", "drop", "ollama", "--restore-brew"})
	assert.NoError(t, root2.Execute())

	assert.Equal(t, "ollama", fb.started, "drop --restore-brew should hand back to brew via brew services start")
}
