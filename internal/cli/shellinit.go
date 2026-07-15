package cli

import (
	"fmt"

	"github.com/laidbackgeek/envonce/internal/config"
	"github.com/laidbackgeek/envonce/internal/env"
	"github.com/laidbackgeek/envonce/internal/i18n"
	"github.com/laidbackgeek/envonce/internal/paths"
	"github.com/spf13/cobra"
)

// NewShellInitCmd builds the `shell-init` command that prints the export lines a
// shell should eval (auto-wired into ~/.zshrc by `envonce init`).
func NewShellInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "shell-init",
		Short:       T.T(i18n.ShellInitShort),
		Long:        T.T(i18n.ShellInitLong),
		Annotations: map[string]string{annShortKey: i18n.ShellInitShort, annLongKey: i18n.ShellInitLong},
		RunE: func(c *cobra.Command, args []string) error {
			cfgPath, err := paths.ConfigFile()
			if err != nil {
				return err
			}
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}
			envDir, err := paths.EnvDir()
			if err != nil {
				return err
			}
			lines, err := env.New(cfg, envDir, env.NewSecurityKeyChain()).Export(env.ExportContext{ForShell: true})
			if err != nil {
				return err
			}
			for _, l := range lines {
				fmt.Fprintln(c.OutOrStdout(), l)
			}
			return nil
		},
	}
}
