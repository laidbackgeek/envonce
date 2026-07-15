package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/laidbackgeek/envonce/internal/paths"
	"github.com/stretchr/testify/assert"
)

func TestInit_CreatesLayoutAndZshrc(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("SHELL", "/bin/zsh")

	root := NewRootCmd()
	root.SetArgs([]string{"init"})
	assert.NoError(t, root.Execute())

	dir, _ := paths.ConfigDir()
	for _, p := range []string{"env.d/default.env", "config.toml", ".initialized"} {
		_, err := os.Stat(filepath.Join(dir, p))
		assert.NoError(t, err, p)
	}
	zshrc, _ := os.ReadFile(filepath.Join(home, ".zshrc"))
	assert.Contains(t, string(zshrc), `eval "$(envonce shell-init)"`)
	assert.Contains(t, string(zshrc), ">>> envonce >>>")
}

func TestInit_Idempotent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("SHELL", "/bin/zsh")
	for i := 0; i < 2; i++ {
		root := NewRootCmd()
		root.SetArgs([]string{"init"})
		assert.NoError(t, root.Execute())
	}
	zshrc, _ := os.ReadFile(filepath.Join(home, ".zshrc"))
	assert.Equal(t, 1, bytes.Count(zshrc, []byte(">>> envonce >>>")))
}

func TestInit_Uninstall(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("SHELL", "/bin/zsh")
	r := NewRootCmd()
	r.SetArgs([]string{"init"})
	assert.NoError(t, r.Execute())
	r2 := NewRootCmd()
	r2.SetArgs([]string{"init", "--uninstall"})
	assert.NoError(t, r2.Execute())
	zshrc, _ := os.ReadFile(filepath.Join(home, ".zshrc"))
	assert.NotContains(t, string(zshrc), "envonce")
}

func TestFirstRunBanner(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	out := &bytes.Buffer{}
	PrintFirstRunBanner(out)
	assert.Contains(t, out.String(), "envonce init")
}
