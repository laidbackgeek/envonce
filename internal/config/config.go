// Package config defines envonce's TOML configuration model (managed services
// and shell groups) and provides loading, saving, and group-resolution helpers.
package config

import (
	"os"

	"github.com/pelletier/go-toml/v2"
)

// ShellConfig holds the shell-integration settings, currently just the env groups
// active in interactive shells.
type ShellConfig struct {
	Groups []string `toml:"groups"`
}

// ServiceDef describes a single managed background service: how to run it, which
// env groups it consumes, and the launchd lifecycle flags envonce applies.
type ServiceDef struct {
	Source           string   `toml:"source"`
	Binary           string   `toml:"binary"`
	Args             []string `toml:"args"`
	Groups           []string `toml:"groups"`
	KeepAlive        bool     `toml:"keep_alive"`
	RunAtLoad        bool     `toml:"run_at_load"`
	ThrottleInterval int      `toml:"throttle_interval"`
	StdoutLog        string   `toml:"stdout_log,omitempty"`
	StderrLog        string   `toml:"stderr_log,omitempty"`
}

// Config is the root configuration object, serialized to config.toml.
type Config struct {
	Shell    ShellConfig           `toml:"shell"`
	Services map[string]ServiceDef `toml:"services"`
}

// Default returns the initial config: the "default" shell group and an empty service map.
func Default() *Config {
	return &Config{
		Shell:    ShellConfig{Groups: []string{"default"}},
		Services: map[string]ServiceDef{},
	}
}

// Load reads and parses the TOML config at path, filling defaults for any missing fields.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	c := Default()
	if err := toml.Unmarshal(data, c); err != nil {
		return nil, err
	}
	if len(c.Shell.Groups) == 0 {
		c.Shell.Groups = []string{"default"}
	}
	if c.Services == nil {
		c.Services = map[string]ServiceDef{}
	}
	return c, nil
}

// Save serializes the config to path as TOML.
func (c *Config) Save(path string) error {
	data, err := toml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// GroupsForService returns default first, then the service's extra groups (de-duplicated, order preserved).
func (c *Config) GroupsForService(name string) []string {
	svc, ok := c.Services[name]
	if !ok {
		return []string{"default"}
	}
	groups := []string{"default"}
	seen := map[string]bool{"default": true}
	for _, g := range svc.Groups {
		if !seen[g] {
			groups = append(groups, g)
			seen[g] = true
		}
	}
	return groups
}

// GroupsForShell returns the shell's active groups, defaulting to ["default"] when unset.
func (c *Config) GroupsForShell() []string {
	if len(c.Shell.Groups) == 0 {
		return []string{"default"}
	}
	return c.Shell.Groups
}
