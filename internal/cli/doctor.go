package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/laidbackgeek/envonce/internal/config"
	"github.com/laidbackgeek/envonce/internal/i18n"
	"github.com/laidbackgeek/envonce/internal/paths"
	"github.com/spf13/cobra"
)

// CheckResult holds the result of a single self-check.
type CheckResult struct {
	Name    string
	OK      bool
	Message string
}

// RunDoctor runs the environment self-checks and returns per-item results (testable).
func RunDoctor() []CheckResult {
	var rs []CheckResult

	// 1. initialization marker
	if marker, err := paths.InitializedMarker(); err != nil {
		rs = append(rs, CheckResult{Name: "initialized", OK: false, Message: T.T(i18n.DoctorMsgNoConfigDir)})
	} else if _, err := os.Stat(marker); err != nil {
		rs = append(rs, CheckResult{Name: "initialized", OK: false, Message: T.T(i18n.DoctorMsgNotInit)})
	} else {
		rs = append(rs, CheckResult{Name: "initialized", OK: true})
	}

	// 2. brew reachable
	if _, err := exec.LookPath("brew"); err != nil {
		rs = append(rs, CheckResult{Name: "brew", OK: false, Message: T.T(i18n.DoctorMsgBrewMissing)})
	} else {
		rs = append(rs, CheckResult{Name: "brew", OK: true})
	}

	// 3. /usr/bin/security present
	if _, err := os.Stat("/usr/bin/security"); err != nil {
		rs = append(rs, CheckResult{Name: "security", OK: false, Message: T.T(i18n.DoctorMsgSecurityMissing)})
	} else {
		rs = append(rs, CheckResult{Name: "security", OK: true})
	}

	// 4. brew leftover-plist collision: for each managed service, check homebrew.mxcl.<name>.plist
	if cfgPath, err := paths.ConfigFile(); err == nil {
		if cfg, err := config.Load(cfgPath); err == nil {
			for name := range cfg.Services {
				brewPlist := filepath.Join(homeLaunchAgents(), "homebrew.mxcl."+name+".plist")
				if _, err := os.Stat(brewPlist); err == nil {
					rs = append(rs, CheckResult{
						Name:    "brew-plist-collision:" + name,
						OK:      false,
						Message: T.T(i18n.DoctorMsgBrewPlistCollision, brewPlist),
					})
				}
			}
		}
	}

	// 5. XDG drift: the ConfigDir recorded in .initialized differs from the current one (stale after copy/move)
	if marker, err := paths.InitializedMarker(); err == nil {
		if data, err := os.ReadFile(marker); err == nil {
			recorded := strings.TrimSpace(string(data))
			if recorded != "" { // empty marker = written by an older init, skip (backward compat)
				if cur, err := paths.ConfigDir(); err == nil && recorded != cur {
					rs = append(rs, CheckResult{
						Name:    "xdg-drift",
						OK:      false,
						Message: T.T(i18n.DoctorMsgXDGDrift, recorded, cur),
					})
				}
			}
		}
	}

	return rs
}

// homeLaunchAgents returns $HOME/Library/LaunchAgents.
func homeLaunchAgents() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents")
}

// NewDoctorCmd builds the doctor self-check subcommand.
func NewDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "doctor",
		Short:       T.T(i18n.DoctorShort),
		Annotations: map[string]string{annShortKey: i18n.DoctorShort},
		RunE: func(c *cobra.Command, args []string) error {
			for _, r := range RunDoctor() {
				mark := "✓"
				if !r.OK {
					mark = "✗"
				}
				fmt.Fprintf(c.OutOrStdout(), "%s %s %s\n", mark, r.Name, r.Message)
			}
			return nil
		},
	}
}
