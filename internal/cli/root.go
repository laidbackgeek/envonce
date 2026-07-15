// Package cli implements envonce's command-line interface: the cobra command tree
// (env, service, group, init, doctor, shell-init) and the glue wiring in the
// launchd, brew, config, and i18n dependencies.
package cli

import (
	"fmt"
	"os"

	"github.com/laidbackgeek/envonce/internal/brew"
	"github.com/laidbackgeek/envonce/internal/i18n"
	"github.com/spf13/cobra"
)

var langFlag string

// appVersion is the version, injected from main via SetVersion (ldflags).
var appVersion = "dev"

// SetVersion is called from main to pass the ldflags-injected version into the cli package.
func SetVersion(v string) { appVersion = v }

// T is the global translator, initialized by root in PersistentPreRun per --lang/env.
var T = i18n.New(i18n.EN)

// deps aggregates the external clients the cli depends on; injected into the command
// tree via newRootCmd (avoids package-level mutable globals).
type deps struct {
	launchd LaunchdClient
	brew    brew.Client
}

// defaultDeps returns the real production dependencies.
func defaultDeps() deps {
	return deps{launchd: realLaunchdClient{}, brew: brew.New()}
}

// newRootCmd builds the root command and injects dependency d; in-package tests inject fakes.
func newRootCmd(d deps) *cobra.Command {
	var showVersion bool
	root := &cobra.Command{
		Use:         "envonce",
		Short:       T.T(i18n.RootShort),
		Long:        T.T(i18n.RootLong),
		Annotations: map[string]string{annShortKey: i18n.RootShort, annLongKey: i18n.RootLong},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			T = i18n.New(i18n.Detect(langFlag))
			applyI18n(cmd.Root())
			if !IsInitialized() && !skipBanner(cmd) {
				PrintFirstRunBanner(os.Stderr)
			}
			return nil
		},
		// Run the root itself when no subcommand is given: handle --version.
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				fmt.Fprintf(cmd.OutOrStdout(), "envonce %s\n", appVersion)
				return nil
			}
			return cmd.Help()
		},
		SilenceUsage: true,
	}
	root.PersistentFlags().StringVar(&langFlag, "lang", "", T.T(i18n.FlagLang))
	_ = root.PersistentFlags().SetAnnotation("lang", annFlagKey, []string{i18n.FlagLang})
	root.Flags().BoolVarP(&showVersion, "version", "v", false, T.T(i18n.FlagVersion))
	_ = root.Flags().SetAnnotation("version", annFlagKey, []string{i18n.FlagVersion})
	root.AddCommand(NewEnvCmd())
	root.AddCommand(NewShellInitCmd())
	root.AddCommand(NewInitCmd())
	root.AddCommand(NewServiceCmd(d))
	root.AddCommand(NewGroupCmd())
	root.AddCommand(NewDoctorCmd())

	// --help is short-circuited by cobra early, so PersistentPreRunE (which detects
	// language) never runs. Set the language explicitly inside the HelpFunc and call
	// applyI18n, then delegate to cobra's default rendering.
	// Note: never call cmd.Help() inside this closure — it would recurse into itself.
	defaultHelp := root.HelpFunc()
	root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		T = i18n.New(i18n.Detect(langFlag)) // langFlag is already populated by ParseFlags
		applyI18n(cmd.Root())
		defaultHelp(cmd, args)
	})
	return root
}

// NewRootCmd is the production entry point (used by cmd/envonce and tests that don't inject), wiring the default dependencies.
func NewRootCmd() *cobra.Command {
	return newRootCmd(defaultDeps())
}

// skipBanner reports whether the command should skip the first-run banner (init/help/completion themselves).
func skipBanner(cmd *cobra.Command) bool {
	name := cmd.Name()
	if name == "init" || name == "help" || name == "completion" || name == "--help" {
		return true
	}
	if cmd.Flags().Changed("version") {
		return true
	}
	return false
}

// Execute builds and runs the root command, returning the exit code.
func Execute() int {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
