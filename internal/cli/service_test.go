package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/laidbackgeek/envonce/internal/brew"
	"github.com/laidbackgeek/envonce/internal/paths"
	"github.com/stretchr/testify/assert"
)

// fakeLaunchd replaces real launchctl calls so tests don't touch the system launchd.
type fakeLaunchd struct{ loaded map[string]bool }

func (f *fakeLaunchd) Bootstrap(p string) error       { return nil }
func (f *fakeLaunchd) Bootout(l string) error         { f.loaded[l] = false; return nil }
func (f *fakeLaunchd) IsLoaded(l string) bool         { return f.loaded[l] }
func (f *fakeLaunchd) Print(l string) (string, error) { return "", nil }
func (f *fakeLaunchd) LabelFor(n string) string       { return "com.envonce." + n }

func TestServiceAdd_SyncGenerates(t *testing.T) {
	setupHome(t)
	fl := &fakeLaunchd{loaded: map[string]bool{}}
	root := newRootCmd(deps{launchd: fl, brew: brew.New()})
	root.SetArgs([]string{"service", "add", "myapp", "--binary", "/usr/bin/python3", "--", "-m", "http.server"})
	assert.NoError(t, root.Execute())

	dir, _ := paths.ConfigDir()
	_, err := os.Stat(filepath.Join(dir, "services", "myapp.wrapper.sh"))
	assert.NoError(t, err)
	// use paths.PlistPath(label) instead of the non-existent PlistPathForName from the brief.
	_, err = os.Stat(paths.PlistPath(fl.LabelFor("myapp")))
	assert.NoError(t, err)
}

// addService is a test helper that registers a service via `service add`.
func addService(t *testing.T, fl *fakeLaunchd, name string) {
	t.Helper()
	root := newRootCmd(deps{launchd: fl, brew: brew.New()})
	root.SetArgs([]string{"service", "add", name, "--binary", "/usr/bin/true"})
	assert.NoError(t, root.Execute())
}

func TestServiceStart_Runs(t *testing.T) {
	setupHome(t)
	fl := &fakeLaunchd{loaded: map[string]bool{}}
	addService(t, fl, "myapp")

	out := &bytes.Buffer{}
	root := newRootCmd(deps{launchd: fl, brew: brew.New()})
	root.SetOut(out)
	root.SetArgs([]string{"service", "start", "myapp"})
	assert.NoError(t, root.Execute())
	assert.Contains(t, out.String(), "myapp")
}

func TestServiceStop_Bootouts(t *testing.T) {
	setupHome(t)
	fl := &fakeLaunchd{loaded: map[string]bool{}}
	addService(t, fl, "myapp")

	fl.loaded[fl.LabelFor("myapp")] = true // simulate the service being loaded
	root := newRootCmd(deps{launchd: fl, brew: brew.New()})
	root.SetArgs([]string{"service", "stop", "myapp"})
	assert.NoError(t, root.Execute())
	assert.False(t, fl.loaded[fl.LabelFor("myapp")], "stop should bootout (unload) the service")
}

func TestServiceRestart_Runs(t *testing.T) {
	setupHome(t)
	fl := &fakeLaunchd{loaded: map[string]bool{}}
	addService(t, fl, "myapp")

	out := &bytes.Buffer{}
	root := newRootCmd(deps{launchd: fl, brew: brew.New()})
	root.SetOut(out)
	root.SetArgs([]string{"service", "restart", "myapp"})
	assert.NoError(t, root.Execute())
	assert.Contains(t, out.String(), "myapp")
}

func TestServiceStatus_PrintsLabel(t *testing.T) {
	setupHome(t)
	fl := &fakeLaunchd{loaded: map[string]bool{}}
	addService(t, fl, "myapp")

	fl.loaded[fl.LabelFor("myapp")] = true
	out := &bytes.Buffer{}
	root := newRootCmd(deps{launchd: fl, brew: brew.New()})
	root.SetOut(out)
	root.SetArgs([]string{"--lang", "en", "service", "status", "myapp"})
	assert.NoError(t, root.Execute())
	assert.Contains(t, out.String(), "myapp")
	assert.Contains(t, out.String(), "Running")
}

func TestServiceSync_Regenerates(t *testing.T) {
	setupHome(t)
	fl := &fakeLaunchd{loaded: map[string]bool{}}
	addService(t, fl, "myapp")

	wrapperPath, err := paths.WrapperPath("myapp")
	assert.NoError(t, err)
	assert.NoError(t, os.Remove(wrapperPath)) // corrupt, to verify sync regenerates it

	root := newRootCmd(deps{launchd: fl, brew: brew.New()})
	root.SetArgs([]string{"service", "sync", "myapp"})
	assert.NoError(t, root.Execute())
	_, err = os.Stat(wrapperPath)
	assert.NoError(t, err, "sync should regenerate the wrapper")
}

func TestServiceList_Sorted(t *testing.T) {
	setupHome(t)
	fl := &fakeLaunchd{loaded: map[string]bool{}}
	// names chosen so insertion order != sorted order
	addService(t, fl, "zeta")
	addService(t, fl, "alpha")

	out := &bytes.Buffer{}
	root := newRootCmd(deps{launchd: fl, brew: brew.New()})
	root.SetOut(out)
	root.SetArgs([]string{"service", "list"})
	assert.NoError(t, root.Execute())
	body := out.String()
	assert.Contains(t, body, "alpha")
	assert.Contains(t, body, "zeta")
	assert.True(t, strings.Index(body, "alpha") < strings.Index(body, "zeta"), "list should be sorted")
}
