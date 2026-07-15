package config

import (
	"errors"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefault(t *testing.T) {
	c := Default()
	assert.Equal(t, []string{"default"}, c.Shell.Groups)
	assert.NotNil(t, c.Services)
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	c := Default()
	c.Services["ollama"] = ServiceDef{
		Source: "brew", Binary: "/opt/homebrew/opt/ollama/bin/ollama",
		Args: []string{"serve"}, Groups: []string{"work"},
		KeepAlive: true, RunAtLoad: true, ThrottleInterval: 10,
	}
	assert.NoError(t, c.Save(p))

	got, err := Load(p)
	assert.NoError(t, err)
	assert.Equal(t, "brew", got.Services["ollama"].Source)
	assert.Equal(t, []string{"serve"}, got.Services["ollama"].Args)
}

func TestLoad_Missing(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "x.toml"))
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}

func TestGroupsForService_IncludesDefault(t *testing.T) {
	c := Default()
	c.Services["x"] = ServiceDef{Groups: []string{"work", "golang"}}
	// default first, de-duplicated, order preserved
	assert.Equal(t, []string{"default", "work", "golang"}, c.GroupsForService("x"))
}

func TestGroupsForService_DefaultNoDup(t *testing.T) {
	c := Default()
	c.Services["x"] = ServiceDef{Groups: []string{"default", "work"}}
	assert.Equal(t, []string{"default", "work"}, c.GroupsForService("x"))
}

func TestGroupsForShell(t *testing.T) {
	c := Default()
	assert.Equal(t, []string{"default"}, c.GroupsForShell())
	c.Shell.Groups = []string{"default", "golang"}
	assert.Equal(t, []string{"default", "golang"}, c.GroupsForShell())
}
