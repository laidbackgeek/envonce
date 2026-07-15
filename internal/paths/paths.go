// Package paths computes envonce's filesystem locations (config, env, services,
// logs, state directories, and the generated wrapper/plist paths), following the
// XDG base-directory specification.
package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

// ConfigDir returns the envonce config directory: $XDG_CONFIG_HOME/envonce,
// falling back to ~/.config/envonce when XDG_CONFIG_HOME is unset.
func ConfigDir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "envonce"), nil
}

// EnvDir returns the env.d directory holding the *.env group files.
func EnvDir() (string, error) { d, err := ConfigDir(); return filepath.Join(d, "env.d"), err }

// ServicesDir returns the directory holding generated wrapper scripts.
func ServicesDir() (string, error) { d, err := ConfigDir(); return filepath.Join(d, "services"), err }

// LogsDir returns the directory for envonce and per-service logs.
func LogsDir() (string, error) { d, err := ConfigDir(); return filepath.Join(d, "logs"), err }

// StateDir returns the directory for runtime state.
func StateDir() (string, error) { d, err := ConfigDir(); return filepath.Join(d, "state"), err }

// ConfigFile returns the path to config.toml.
func ConfigFile() (string, error) { d, err := ConfigDir(); return filepath.Join(d, "config.toml"), err }

// InitializedMarker returns the path to the .initialized sentinel written by `envonce init`.
func InitializedMarker() (string, error) {
	d, err := ConfigDir()
	return filepath.Join(d, ".initialized"), err
}

// WrapperPath returns the path to a service's generated wrapper script.
func WrapperPath(name string) (string, error) {
	d, err := ServicesDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, fmt.Sprintf("%s.wrapper.sh", name)), nil
}

// PlistPath returns the absolute path ~/Library/LaunchAgents/<label>.plist.
func PlistPath(label string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", label+".plist")
}
