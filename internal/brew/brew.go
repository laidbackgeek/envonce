// Package brew parses the launchd plist that Homebrew generates and
// normalizes Cellar versioned paths to their opt symlink form, used when
// envonce imports a brew-managed service (e.g. ollama).
package brew

import (
	"os"
	"os/exec"
	"path"
	"strings"

	"howett.net/plist"
)

// ServiceInfo describes a launchd service parsed from a brew plist.
type ServiceInfo struct {
	Name      string
	Binary    string
	Args      []string
	KeepAlive bool
	RunAtLoad bool
	StdoutLog string
	StderrLog string
	Env       map[string]string // EnvironmentVariables from the brew plist; migrated into env.d on take
}

// Client abstracts importing, stopping, and starting brew services so tests can inject fakes.
type Client interface {
	ImportService(name string) (*ServiceInfo, error)
	StopService(name string) error
	StartService(name string) error
}

type realClient struct {
	run func(name string, args ...string) ([]byte, error)
}

// New returns a Client that uses the real exec.Command.
func New() Client {
	return &realClient{run: func(n string, a ...string) ([]byte, error) { return exec.Command(n, a...).Output() }}
}

// NormalizeBinaryToOpt converts /opt/homebrew/Cellar/<x>/<ver>/<tail> into
// /opt/homebrew/opt/<x>/<tail> (same for the Intel prefix /usr/local).
// Paths already in opt form or otherwise unrecognized are returned as-is.
func NormalizeBinaryToOpt(p string) string {
	for _, prefix := range []string{"/opt/homebrew/Cellar/", "/usr/local/Cellar/"} {
		if strings.HasPrefix(p, prefix) {
			rest := strings.TrimPrefix(p, prefix)
			parts := strings.SplitN(rest, "/", 3) // <name>/<version>/<tail>
			if len(parts) == 3 {
				base := strings.TrimSuffix(prefix, "Cellar/") // -> /opt/homebrew/ or /usr/local/
				return path.Join(base, "opt", parts[0], parts[2])
			}
		}
	}
	return p
}

// ReadBrewPlist parses brew-generated plist data (binary or XML) into a ServiceInfo.
// ProgramArguments[0] is normalized via NormalizeBinaryToOpt. KeepAlive/RunAtLoad may
// be a bool or a dict (conditional keep-alive) in the plist; only the bool semantics are kept.
func ReadBrewPlist(name string, data []byte) (*ServiceInfo, error) {
	var raw struct {
		Label             string            `plist:"Label"`
		ProgramArguments  []string          `plist:"ProgramArguments"`
		KeepAlive         any               `plist:"KeepAlive"`
		RunAtLoad         any               `plist:"RunAtLoad"`
		StandardOutPath   string            `plist:"StandardOutPath"`
		StandardErrorPath string            `plist:"StandardErrorPath"`
		EnvironmentVars   map[string]string `plist:"EnvironmentVariables"`
	}
	if _, err := plist.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	info := &ServiceInfo{
		Name:      name,
		StdoutLog: raw.StandardOutPath,
		StderrLog: raw.StandardErrorPath,
		Env:       raw.EnvironmentVars,
	}
	if len(raw.ProgramArguments) > 0 {
		info.Binary = NormalizeBinaryToOpt(raw.ProgramArguments[0])
		info.Args = raw.ProgramArguments[1:]
	}
	info.KeepAlive = truthy(raw.KeepAlive)
	info.RunAtLoad = truthy(raw.RunAtLoad)
	return info, nil
}

// truthy returns v's value under bool semantics; non-bools (e.g. a dict-form conditional keep-alive) are treated as false.
func truthy(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	default:
		return false
	}
}

// userHome returns the current user's home directory, or "" on failure (very rare).
func userHome() string {
	h, _ := os.UserHomeDir()
	return h
}

// ImportService reads the launchd plist brew generated for name and parses it into a ServiceInfo.
// If the plist isn't materialized yet (brew never started it), it first runs
// `brew services start <name>` to trigger generation (errors ignored), then reads it.
func (c *realClient) ImportService(name string) (*ServiceInfo, error) {
	brewPlist := userHome() + "/Library/LaunchAgents/homebrew.mxcl." + name + ".plist"
	if _, err := os.Stat(brewPlist); err != nil {
		// Materialize the plist; ignore errors (brew may be missing or the service never started).
		_, _ = c.run("brew", "services", "start", name)
	}
	data, err := os.ReadFile(brewPlist)
	if err != nil {
		return nil, err
	}
	return ReadBrewPlist(name, data)
}

// StopService runs `brew services stop <name>` to stop the brew-managed service.
func (c *realClient) StopService(name string) error {
	_, err := c.run("brew", "services", "stop", name)
	return err
}

// StartService runs `brew services start <name>` to hand a service back to brew management.
func (c *realClient) StartService(name string) error {
	_, err := c.run("brew", "services", "start", name)
	return err
}
