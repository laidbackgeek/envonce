package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/laidbackgeek/envonce/internal/config"
	"github.com/laidbackgeek/envonce/internal/i18n"
	"github.com/laidbackgeek/envonce/internal/paths"
	"github.com/spf13/cobra"
)

const (
	markerBegin = "# >>> envonce >>>"
	markerEnd   = "# <<< envonce <<<"
)

// IsInitialized reports whether the .initialized marker exists under the config directory.
func IsInitialized() bool {
	p, err := paths.InitializedMarker()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// PrintFirstRunBanner writes the first-run notice to w (called by root's
// PersistentPreRunE when not initialized). Single-line form: an ASCII box
// misaligns when CJK and Latin text mix, so a single ⚠ line is used instead.
func PrintFirstRunBanner(w io.Writer) {
	fmt.Fprintf(w, "\n⚠ %s\n\n", T.T(i18n.BannerNotInit))
}

// NewInitCmd builds the init subcommand; --uninstall removes the shell integration (keeping config and data).
func NewInitCmd() *cobra.Command {
	var uninstall bool
	c := &cobra.Command{
		Use:           "init",
		Short:         T.T(i18n.InitShort),
		Annotations:   map[string]string{annShortKey: i18n.InitShort},
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, args []string) error {
			if uninstall {
				return uninstallShell()
			}
			return doInit(c.OutOrStdout())
		},
	}
	c.Flags().BoolVar(&uninstall, "uninstall", false, T.T(i18n.FlagUninstall))
	_ = c.Flags().SetAnnotation("uninstall", annFlagKey, []string{i18n.FlagUninstall})
	return c
}

func doInit(w io.Writer) error {
	dir, err := paths.ConfigDir()
	if err != nil {
		return err
	}
	for _, sub := range []string{"env.d", "services", "logs", "state"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			return err
		}
	}
	defEnv := filepath.Join(dir, "env.d", "default.env")
	if _, err := os.Stat(defEnv); errors.Is(err, fs.ErrNotExist) {
		if err := os.WriteFile(defEnv, []byte(""), 0o644); err != nil {
			return err
		}
	}
	cfgPath, _ := paths.ConfigFile()
	if _, err := os.Stat(cfgPath); errors.Is(err, fs.ErrNotExist) {
		if err := config.Default().Save(cfgPath); err != nil {
			return err
		}
	}
	if err := wireShellRC(); err != nil {
		return err
	}
	marker, _ := paths.InitializedMarker()
	// .initialized records the ConfigDir at init time so doctor can detect XDG drift
	// (the marker goes stale once the config dir is copied or moved).
	if err := os.WriteFile(marker, []byte(dir), 0o644); err != nil {
		return err
	}

	fmt.Fprintf(w, "%s\n", T.T(i18n.SummaryInitDir, dir))
	fmt.Fprintf(w, "%s\n", T.T(i18n.SummaryInitFiles))
	fmt.Fprintf(w, "%s\n", T.T(i18n.SummaryInitShell, strings.TrimSpace(markerBegin)))
	fmt.Fprintf(w, "%s\n\n", T.T(i18n.SummaryInitMarker))
	fmt.Fprintf(w, "%s\n", T.T(i18n.SummaryInitNextRoll))
	return nil
}

// rcFile returns ~/.zshrc or ~/.bashrc based on $SHELL.
func rcFile() (string, error) {
	shell := os.Getenv("SHELL")
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	name := ".zshrc"
	if strings.Contains(shell, "bash") {
		name = ".bashrc"
	}
	return filepath.Join(home, name), nil
}

// wireShellRC idempotently appends the marker-wrapped eval line to the shell rc file.
func wireShellRC() error {
	rc, err := rcFile()
	if err != nil {
		return err
	}
	data, _ := os.ReadFile(rc)
	if strings.Contains(string(data), markerBegin) {
		return nil // idempotent
	}
	block := fmt.Sprintf("\n%s\neval \"$(envonce shell-init)\"\n%s\n", markerBegin, markerEnd)
	return os.WriteFile(rc, append(data, []byte(block)...), 0o644)
}

// uninstallShell removes the marker block from the shell rc file (keeping the rest).
func uninstallShell() error {
	rc, err := rcFile()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(rc)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	s := string(data)
	for {
		b := strings.Index(s, markerBegin)
		if b < 0 {
			break
		}
		e := strings.Index(s[b:], markerEnd)
		if e < 0 {
			break
		}
		s = s[:b] + s[b+e+len(markerEnd):]
	}
	return os.WriteFile(rc, []byte(s), 0o644)
}
