package paths

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigDir_DefaultXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	got, err := ConfigDir()
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".config", "envonce"), got)
}

func TestConfigDir_RespectsXDG(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	got, err := ConfigDir()
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(xdg, "envonce"), got)
}

func TestSubdirs(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", t.TempDir())
	base, _ := ConfigDir()

	type c struct {
		name string
		got  string
		want string
	}
	cases := []c{
		{"EnvDir", must(EnvDir()), filepath.Join(base, "env.d")},
		{"ServicesDir", must(ServicesDir()), filepath.Join(base, "services")},
		{"LogsDir", must(LogsDir()), filepath.Join(base, "logs")},
		{"StateDir", must(StateDir()), filepath.Join(base, "state")},
		{"ConfigFile", must(ConfigFile()), filepath.Join(base, "config.toml")},
		{"InitializedMarker", must(InitializedMarker()), filepath.Join(base, ".initialized")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.got)
		})
	}
}

func must(s string, _ error) string { return s }

func TestPlistPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	_ = os.Getenv("HOME")
	got := PlistPath("com.envonce.ollama")
	assert.True(t, filepath.IsAbs(got))
	assert.Contains(t, got, "Library/LaunchAgents/com.envonce.ollama.plist")
}
