package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/laidbackgeek/envonce/internal/i18n"
	"github.com/laidbackgeek/envonce/internal/paths"
	"github.com/spf13/cobra"
)

// NewGroupCmd builds the env-group management command group: create/list/rename/delete.
// Groups map to env.d/*.env files, referenced by name from env set --group / service config.
func NewGroupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "group",
		Short:       T.T(i18n.GroupShort),
		Annotations: map[string]string{annShortKey: i18n.GroupShort},
	}
	cmd.AddCommand(groupCreateCmd(), groupListCmd(), groupRenameCmd(), groupDeleteCmd())
	return cmd
}

func groupCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "create NAME",
		Short:       T.T(i18n.GroupCreateShort),
		Annotations: map[string]string{annShortKey: i18n.GroupCreateShort},
		Args:        cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, a []string) error {
			name := a[0]
			p, err := groupFile(name)
			if err != nil {
				return err
			}
			if _, err := os.Stat(p); err == nil {
				return fmt.Errorf("%s", T.T(i18n.ErrGroupExists, name))
			}
			if err := os.WriteFile(p, []byte(""), 0o644); err != nil {
				return err
			}
			fmt.Fprintf(c.OutOrStdout(), "%s\n", T.T(i18n.SummaryGroupCreateDone, name, p, name, name))
			return nil
		},
	}
}

func groupListCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "list",
		Short:       T.T(i18n.GroupListShort),
		Annotations: map[string]string{annShortKey: i18n.GroupListShort},
		RunE: func(c *cobra.Command, a []string) error {
			d, err := paths.EnvDir()
			if err != nil {
				return err
			}
			ents, _ := os.ReadDir(d)
			for _, e := range ents {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".env") {
					continue
				}
				fmt.Fprintf(c.OutOrStdout(), "%s\n", strings.TrimSuffix(e.Name(), ".env"))
			}
			fmt.Fprintf(c.OutOrStdout(), "%s\n", T.T(i18n.SummaryGroupListRelated))
			return nil
		},
	}
}

func groupRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "rename OLD NEW",
		Short:       T.T(i18n.GroupRenameShort),
		Annotations: map[string]string{annShortKey: i18n.GroupRenameShort},
		Args:        cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, a []string) error {
			oldName, newName := a[0], a[1]
			d, err := paths.EnvDir()
			if err != nil {
				return err
			}
			oldPath := filepath.Join(d, oldName+".env")
			newPath := filepath.Join(d, newName+".env")
			if _, err := os.Stat(newPath); err == nil {
				return fmt.Errorf("%s", T.T(i18n.ErrGroupExists, newName))
			}
			if err := os.Rename(oldPath, newPath); err != nil {
				return err
			}
			fmt.Fprintf(c.OutOrStdout(), "%s\n", T.T(i18n.SummaryGroupRenameDone, oldName, newName, newName, newName, oldName))
			return nil
		},
	}
}

func groupDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "delete NAME",
		Short:       T.T(i18n.GroupDeleteShort),
		Annotations: map[string]string{annShortKey: i18n.GroupDeleteShort},
		Args:        cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, a []string) error {
			name := a[0]
			d, err := paths.EnvDir()
			if err != nil {
				return err
			}
			p := filepath.Join(d, name+".env")
			if err := os.Remove(p); err != nil {
				return err
			}
			fmt.Fprintf(c.OutOrStdout(), "%s\n", T.T(i18n.SummaryGroupDeleteDone, name, p, name))
			return nil
		},
	}
}
