package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/laidbackgeek/envonce/internal/config"
	"github.com/laidbackgeek/envonce/internal/paths"
	"github.com/stretchr/testify/assert"
)

func setupHome(t *testing.T) string {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	dir, _ := paths.ConfigDir()
	for _, sub := range []string{"env.d", "services", "logs", "state"} {
		_ = os.MkdirAll(filepath.Join(dir, sub), 0o755)
	}
	_ = os.WriteFile(filepath.Join(dir, "env.d", "default.env"), []byte(""), 0o644)
	_ = config.Default().Save(filepath.Join(dir, "config.toml"))
	_ = os.WriteFile(filepath.Join(dir, ".initialized"), []byte{}, 0o644)
	return home
}

func mustConfigDir(t *testing.T) string {
	t.Helper()
	d, err := paths.ConfigDir()
	assert.NoError(t, err)
	return d
}

func TestEnvSet_Get(t *testing.T) {
	setupHome(t)
	root := NewRootCmd()
	root.SetArgs([]string{"env", "set", "FOO=bar"})
	assert.NoError(t, root.Execute())

	root2 := NewRootCmd()
	out := &bytes.Buffer{}
	root2.SetOut(out)
	root2.SetArgs([]string{"env", "get", "FOO"})
	assert.NoError(t, root2.Execute())
	assert.Contains(t, out.String(), "bar")
}

func TestEnvExport_DefaultGroup(t *testing.T) {
	setupHome(t)
	root := NewRootCmd()
	root.SetArgs([]string{"env", "set", "PATH=/bin"})
	assert.NoError(t, root.Execute())

	root2 := NewRootCmd()
	out := &bytes.Buffer{}
	root2.SetOut(out)
	root2.SetArgs([]string{"env", "export", "--groups", "default"})
	assert.NoError(t, root2.Execute())
	assert.Contains(t, out.String(), "export PATH=/bin")
}

func TestEnvSet_UsesGroup(t *testing.T) {
	setupHome(t)
	root := NewRootCmd()
	root.SetArgs([]string{"env", "set", "X=1", "--group", "golang"})
	assert.NoError(t, root.Execute())
	_, err := os.Stat(filepath.Join(mustConfigDir(t), "env.d", "golang.env"))
	assert.NoError(t, err)
}
