package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"

	"github.com/laidbackgeek/envonce/internal/config"
	"github.com/laidbackgeek/envonce/internal/envfile"
	"github.com/laidbackgeek/envonce/internal/i18n"
	"github.com/laidbackgeek/envonce/internal/paths"
	"github.com/spf13/cobra"
)

// NewTakeCmd builds the `service take` command: imports a service from brew and hands it to envonce.
func NewTakeCmd(d deps) *cobra.Command {
	return &cobra.Command{
		Use:         "take NAME",
		Short:       T.T(i18n.SvcTakeShort),
		Annotations: map[string]string{annShortKey: i18n.SvcTakeShort},
		Args:        cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			name := args[0]
			info, err := d.brew.ImportService(name)
			if err != nil {
				return err
			}
			cfg, err := loadCfg()
			if err != nil {
				return err
			}
			def := config.ServiceDef{
				Source:           "brew",
				Binary:           info.Binary,
				Args:             info.Args,
				KeepAlive:        info.KeepAlive,
				RunAtLoad:        info.RunAtLoad,
				ThrottleInterval: DefaultThrottleInterval,
				StdoutLog:        info.StdoutLog,
				StderrLog:        info.StderrLog,
			}
			// Migrate the brew plist's env vars into the service's same-named group so they
			// aren't silently lost on takeover. Only keys absent from the group are written
			// (user-configured same-named keys aren't overwritten); keys are sorted for determinism.
			if len(info.Env) > 0 {
				f, err := groupFile(name)
				if err != nil {
					return err
				}
				entries, _ := envfile.LoadFile(f) // a missing group is treated as empty
				have := make(map[string]bool, len(entries))
				for _, e := range entries {
					have[e.Key] = true
				}
				keys := make([]string, 0, len(info.Env))
				for k := range info.Env {
					if !have[k] {
						keys = append(keys, k)
					}
				}
				sort.Strings(keys)
				for _, k := range keys {
					entries = envfile.Set(entries, k, info.Env[k])
				}
				if len(keys) > 0 {
					if err := envfile.SaveFile(f, entries); err != nil {
						return err
					}
					fmt.Fprintf(c.OutOrStdout(), "%s\n", T.T(i18n.SummaryTakeMigrated, len(keys), name))
				}
				def.Groups = appendGroup(def.Groups, name)
			}
			cfg.Services[name] = def
			if err := saveCfg(cfg); err != nil {
				return err
			}
			if err := d.brew.StopService(name); err != nil {
				fmt.Fprintf(c.OutOrStderr(), "%s\n", T.T(i18n.ErrBrewStopFailed, err))
			}
			if err := syncService(d.launchd, name, true); err != nil {
				return err
			}
			fmt.Fprintf(c.OutOrStdout(), "%s\n", T.T(i18n.SummaryTakeDone, name, name))
			fmt.Fprintf(c.OutOrStdout(), "%s\n", T.T(i18n.SummaryTakeNextRoll, name, name))
			return nil
		},
	}
}

// NewDropCmd builds the `service drop` command: unloads an envonce-managed service
// and removes it from config. With --restore-brew on a brew-origin service, it hands
// the service back to brew by running `brew services start`.
func NewDropCmd(d deps) *cobra.Command {
	var restoreBrew bool
	c := &cobra.Command{
		Use:         "drop NAME",
		Short:       T.T(i18n.SvcDropShort),
		Annotations: map[string]string{annShortKey: i18n.SvcDropShort},
		Args:        cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			name := args[0]
			label := d.launchd.LabelFor(name)
			_ = d.launchd.Bootout(label)
			// Clean up the plist + wrapper artifacts (ignore if absent)
			removeIfExists(paths.PlistPath(label))
			if wp, err := paths.WrapperPath(name); err == nil {
				removeIfExists(wp)
			}
			cfg, err := loadCfg()
			if err != nil {
				return err
			}
			src := cfg.Services[name].Source
			delete(cfg.Services, name)
			if err := saveCfg(cfg); err != nil {
				return err
			}
			fmt.Fprintf(c.OutOrStdout(), "%s\n", T.T(i18n.SummaryDropDone, name))
			if restoreBrew && src == "brew" {
				// drop's main work is already complete; a failed brew restart is non-fatal.
				if err := d.brew.StartService(name); err != nil {
					fmt.Fprintf(c.OutOrStderr(), "%s\n", T.T(i18n.ErrBrewStartFailed, name, err))
				} else {
					fmt.Fprintf(c.OutOrStdout(), "%s\n", T.T(i18n.SummaryDropRestoreBrew, name))
				}
			}
			return nil
		},
	}
	c.Flags().BoolVar(&restoreBrew, "restore-brew", false, T.T(i18n.FlagRestoreBrew))
	_ = c.Flags().SetAnnotation("restore-brew", annFlagKey, []string{i18n.FlagRestoreBrew})
	return c
}

// removeIfExists deletes the file at path, silently ignoring a missing file (drop is idempotent).
func removeIfExists(path string) {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		// Unexpected errors are ignored (best-effort cleanup; must not abort the drop flow)
		_ = err
	}
}

// appendGroup appends name to groups (if absent) and returns the new slice.
// Used during take so the service references the migration's same-named group.
func appendGroup(groups []string, name string) []string {
	for _, g := range groups {
		if g == name {
			return groups
		}
	}
	return append(groups, name)
}
