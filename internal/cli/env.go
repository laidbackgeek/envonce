package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/laidbackgeek/envonce/internal/config"
	"github.com/laidbackgeek/envonce/internal/env"
	"github.com/laidbackgeek/envonce/internal/envfile"
	"github.com/laidbackgeek/envonce/internal/i18n"
	"github.com/laidbackgeek/envonce/internal/paths"
	"github.com/spf13/cobra"
)

// NewEnvCmd builds the `env` command group: set/get/unset/list/export.
func NewEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "env",
		Short:       T.T(i18n.EnvShort),
		Annotations: map[string]string{annShortKey: i18n.EnvShort},
	}
	cmd.AddCommand(envSetCmd(), envGetCmd(), envUnsetCmd(), envListCmd(), envExportCmd())
	return cmd
}

func groupFile(group string) (string, error) {
	d, err := paths.EnvDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(d, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(d, group+".env"), nil
}

func envSetCmd() *cobra.Command {
	var group string
	c := &cobra.Command{
		Use:  "set KEY=VALUE",
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			if group == "" {
				group = "default"
			}
			kv := args[0]
			idx := strings.Index(kv, "=")
			if idx < 0 {
				return fmt.Errorf("%s", T.T(i18n.ErrInvalidKV))
			}
			key, value := kv[:idx], kv[idx+1:]
			f, err := groupFile(group)
			if err != nil {
				return err
			}
			entries, _ := envfile.LoadFile(f) // a missing file is treated as empty
			entries = envfile.Set(entries, key, value)
			if err := envfile.SaveFile(f, entries); err != nil {
				return err
			}
			fmt.Fprintf(c.OutOrStdout(), "✓ %s\n", T.T(i18n.MsgWroteEnv, f, kv))
			fmt.Fprintf(c.OutOrStdout(), "%s\n%s\n", T.T(i18n.SummaryEnvSetNext, group), T.T(i18n.SummaryEnvSetRollback, key, group))
			return nil
		},
	}
	c.Flags().StringVar(&group, "group", "", T.T(i18n.FlagGroup))
	_ = c.Flags().SetAnnotation("group", annFlagKey, []string{i18n.FlagGroup})
	return c
}

func envGetCmd() *cobra.Command {
	var group string
	c := &cobra.Command{
		Use:  "get KEY",
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			if group == "" {
				group = "default"
			}
			f, err := groupFile(group)
			if err != nil {
				return err
			}
			entries, err := envfile.LoadFile(f)
			if err != nil {
				return err
			}
			for _, e := range entries {
				if e.Key == args[0] {
					fmt.Fprintln(c.OutOrStdout(), e.Value)
					return nil
				}
			}
			return fmt.Errorf("%s", T.T(i18n.ErrNotFound, args[0]))
		},
	}
	c.Flags().StringVar(&group, "group", "", T.T(i18n.FlagGroup))
	_ = c.Flags().SetAnnotation("group", annFlagKey, []string{i18n.FlagGroup})
	return c
}

func envUnsetCmd() *cobra.Command {
	var group string
	c := &cobra.Command{
		Use:  "unset KEY",
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			if group == "" {
				group = "default"
			}
			f, err := groupFile(group)
			if err != nil {
				return err
			}
			entries, err := envfile.LoadFile(f)
			if err != nil {
				return err
			}
			entries = envfile.Unset(entries, args[0])
			if err := envfile.SaveFile(f, entries); err != nil {
				return err
			}
			fmt.Fprintf(c.OutOrStdout(), "%s\n", T.T(i18n.SummaryEnvUnsetDone, args[0], group, group, args[0], group))
			return nil
		},
	}
	c.Flags().StringVar(&group, "group", "", T.T(i18n.FlagGroup))
	_ = c.Flags().SetAnnotation("group", annFlagKey, []string{i18n.FlagGroup})
	return c
}

func envListCmd() *cobra.Command {
	var group string
	c := &cobra.Command{
		Use: "list",
		RunE: func(c *cobra.Command, args []string) error {
			if group == "" {
				group = "default"
			}
			f, err := groupFile(group)
			if err != nil {
				return err
			}
			entries, err := envfile.LoadFile(f)
			if err != nil {
				return err
			}
			for _, e := range entries {
				fmt.Fprintf(c.OutOrStdout(), "%s=%s\n", e.Key, e.Value)
			}
			fmt.Fprintf(c.OutOrStdout(), "%s\n", T.T(i18n.SummaryEnvListRelated, group))
			return nil
		},
	}
	c.Flags().StringVar(&group, "group", "", T.T(i18n.FlagGroup))
	_ = c.Flags().SetAnnotation("group", annFlagKey, []string{i18n.FlagGroup})
	return c
}

func envExportCmd() *cobra.Command {
	var groupsFlag, serviceFlag string
	c := &cobra.Command{
		Use: "export",
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
			r := env.New(cfg, envDir, env.NewSecurityKeyChain())
			var ctx env.ExportContext
			switch {
			case serviceFlag != "":
				ctx = env.ExportContext{ServiceName: serviceFlag}
			default:
				ctx = env.ExportContext{ForShell: true}
				if groupsFlag != "" {
					ctx.Groups = strings.Split(groupsFlag, ",")
				}
			}
			lines, err := r.Export(ctx)
			if err != nil {
				return err
			}
			for _, l := range lines {
				fmt.Fprintln(c.OutOrStdout(), l)
			}
			return nil
		},
	}
	c.Flags().StringVar(&groupsFlag, "groups", "", T.T(i18n.FlagGroups))
	_ = c.Flags().SetAnnotation("groups", annFlagKey, []string{i18n.FlagGroups})
	c.Flags().StringVar(&serviceFlag, "service", "", T.T(i18n.FlagService))
	_ = c.Flags().SetAnnotation("service", annFlagKey, []string{i18n.FlagService})
	return c
}
