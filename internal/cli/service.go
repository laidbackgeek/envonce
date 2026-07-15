package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/laidbackgeek/envonce/internal/config"
	"github.com/laidbackgeek/envonce/internal/env"
	"github.com/laidbackgeek/envonce/internal/i18n"
	"github.com/laidbackgeek/envonce/internal/paths"
	"github.com/laidbackgeek/envonce/internal/plist"
	"github.com/laidbackgeek/envonce/internal/wrapper"
	"github.com/spf13/cobra"
)

// DefaultThrottleInterval is the default number of seconds for launchd restart throttling.
const DefaultThrottleInterval = 10

// LaunchdClient abstracts the cli's dependency on launchd so tests can inject fakes.
type LaunchdClient interface {
	Bootstrap(plistPath string) error
	Bootout(label string) error
	IsLoaded(label string) bool
	Print(label string) (string, error)
	LabelFor(name string) string
}

// realLaunchdClient is a thin wrapper forwarding LaunchdClient methods to launchd_glue.go.
type realLaunchdClient struct{}

func (realLaunchdClient) Bootstrap(p string) error       { return ldBootstrap(p) }
func (realLaunchdClient) Bootout(l string) error         { return ldBootout(l) }
func (realLaunchdClient) IsLoaded(l string) bool         { return ldIsLoaded(l) }
func (realLaunchdClient) Print(l string) (string, error) { return ldPrint(l) }
func (realLaunchdClient) LabelFor(n string) string       { return ldLabelFor(n) }

// selfBinPath returns the absolute path of the current executable, written into the wrapper's ENVONCE=.
func selfBinPath() string {
	p, _ := os.Executable()
	return p
}

// ensureServiceDirs ensures the services and logs directories exist.
func ensureServiceDirs() error {
	for _, g := range []func() (string, error){paths.ServicesDir, paths.LogsDir} {
		d, err := g()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func loadCfg() (*config.Config, error) {
	p, err := paths.ConfigFile()
	if err != nil {
		return nil, err
	}
	return config.Load(p)
}

func saveCfg(c *config.Config) error {
	p, err := paths.ConfigFile()
	if err != nil {
		return err
	}
	return c.Save(p)
}

// syncService regenerates wrapper+plist per the service definition in config.toml; reloads it when bootstrap=true.
func syncService(lc LaunchdClient, name string, bootstrap bool) error {
	if err := ensureServiceDirs(); err != nil {
		return err
	}
	cfg, err := loadCfg()
	if err != nil {
		return err
	}
	svc, ok := cfg.Services[name]
	if !ok {
		return fmt.Errorf("%s", T.T(i18n.ErrUnknownService, name))
	}
	label := lc.LabelFor(name)
	wrapperPath, err := paths.WrapperPath(name)
	if err != nil {
		return err
	}
	wd := wrapper.WrapperData{EnvonceBin: selfBinPath(), ServiceName: name, Binary: svc.Binary, Args: svc.Args}
	if err := os.WriteFile(wrapperPath, []byte(wrapper.Generate(wd)), 0o755); err != nil {
		return err
	}
	logsDir, _ := paths.LogsDir()
	stdout := svc.StdoutLog
	if stdout == "" {
		stdout = filepath.Join(logsDir, name+".out.log")
	}
	stderr := svc.StderrLog
	if stderr == "" {
		stderr = filepath.Join(logsDir, name+".err.log")
	}
	throttle := svc.ThrottleInterval
	if throttle == 0 {
		throttle = DefaultThrottleInterval
	}
	pd := plist.PlistData{
		Label:            label,
		WrapperPath:      wrapperPath,
		RunAtLoad:        svc.RunAtLoad,
		KeepAlive:        svc.KeepAlive,
		ThrottleInterval: throttle,
		StdoutPath:       stdout,
		StderrPath:       stderr,
	}
	xml, err := plist.Generate(pd)
	if err != nil {
		return err
	}
	plistPath := paths.PlistPath(label)
	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(plistPath, []byte(xml), 0o644); err != nil {
		return err
	}
	if bootstrap {
		_ = lc.Bootout(label)
		return lc.Bootstrap(plistPath)
	}
	return nil
}

// statusLabel maps the launchd load state to a localized status label (for service status / list output).
func statusLabel(loaded bool) string {
	if loaded {
		return T.T(i18n.StatusRunning)
	}
	return T.T(i18n.StatusNotLoaded)
}

// NewServiceCmd builds the service-management command group: add/start/stop/restart/status/sync/list/take/drop.
func NewServiceCmd(d deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "service",
		Short:       T.T(i18n.ServiceShort),
		Annotations: map[string]string{annShortKey: i18n.ServiceShort},
	}
	cmd.AddCommand(serviceAddCmd(d.launchd), serviceStartCmd(d.launchd), serviceStopCmd(d.launchd),
		serviceRestartCmd(d.launchd), serviceStatusCmd(d.launchd), serviceSyncCmd(d.launchd), serviceListCmd(d.launchd),
		NewTakeCmd(d), NewDropCmd(d))
	return cmd
}

func serviceAddCmd(lc LaunchdClient) *cobra.Command {
	var binary string
	var keepAlive, runAtLoad bool
	c := &cobra.Command{
		Use:         "add NAME [-- BINARY_ARGS...]",
		Short:       T.T(i18n.SvcAddShort),
		Annotations: map[string]string{annShortKey: i18n.SvcAddShort},
		Args:        cobra.MinimumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			name := args[0]
			rest := args[1:]
			cfg, err := loadCfg()
			if err != nil {
				return err
			}
			cfg.Services[name] = config.ServiceDef{
				Source:           "manual",
				Binary:           binary,
				Args:             rest,
				KeepAlive:        keepAlive,
				RunAtLoad:        runAtLoad,
				ThrottleInterval: DefaultThrottleInterval,
			}
			if err := saveCfg(cfg); err != nil {
				return err
			}
			if err := syncService(lc, name, true); err != nil {
				return err
			}
			fmt.Fprintf(c.OutOrStdout(), "%s\n", T.T(i18n.SummarySvcAddDone, name, name, name))
			return nil
		},
	}
	c.Flags().StringVar(&binary, "binary", "", T.T(i18n.FlagBinary))
	_ = c.Flags().SetAnnotation("binary", annFlagKey, []string{i18n.FlagBinary})
	c.Flags().BoolVar(&keepAlive, "keep-alive", true, T.T(i18n.FlagKeepAlive))
	_ = c.Flags().SetAnnotation("keep-alive", annFlagKey, []string{i18n.FlagKeepAlive})
	c.Flags().BoolVar(&runAtLoad, "run-at-load", true, T.T(i18n.FlagRunAtLoad))
	_ = c.Flags().SetAnnotation("run-at-load", annFlagKey, []string{i18n.FlagRunAtLoad})
	return c
}

func serviceStartCmd(lc LaunchdClient) *cobra.Command {
	return &cobra.Command{
		Use:         "start NAME",
		Short:       T.T(i18n.SvcStartShort),
		Annotations: map[string]string{annShortKey: i18n.SvcStartShort},
		Args:        cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, a []string) error {
			if err := syncService(lc, a[0], true); err != nil {
				return err
			}
			fmt.Fprintf(c.OutOrStdout(), "%s\n", T.T(i18n.SummarySvcStartDone, a[0], a[0], a[0]))
			return nil
		},
	}
}

func serviceStopCmd(lc LaunchdClient) *cobra.Command {
	return &cobra.Command{
		Use:         "stop NAME",
		Short:       T.T(i18n.SvcStopShort),
		Annotations: map[string]string{annShortKey: i18n.SvcStopShort},
		Args:        cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, a []string) error {
			label := lc.LabelFor(a[0])
			if err := lc.Bootout(label); err != nil {
				return err
			}
			fmt.Fprintf(c.OutOrStdout(), "%s\n", T.T(i18n.SummarySvcStopDone, a[0], a[0], a[0]))
			return nil
		},
	}
}

func serviceRestartCmd(lc LaunchdClient) *cobra.Command {
	return &cobra.Command{
		Use:         "restart NAME",
		Short:       T.T(i18n.SvcRestartShort),
		Annotations: map[string]string{annShortKey: i18n.SvcRestartShort},
		Args:        cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, a []string) error {
			label := lc.LabelFor(a[0])
			_ = lc.Bootout(label)
			if err := syncService(lc, a[0], true); err != nil {
				return err
			}
			fmt.Fprintf(c.OutOrStdout(), "%s\n", T.T(i18n.SummarySvcRestartDone, a[0], a[0], a[0]))
			return nil
		},
	}
}

func serviceSyncCmd(lc LaunchdClient) *cobra.Command {
	return &cobra.Command{
		Use:         "sync NAME",
		Short:       T.T(i18n.SvcSyncShort),
		Annotations: map[string]string{annShortKey: i18n.SvcSyncShort},
		Args:        cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, a []string) error {
			if err := syncService(lc, a[0], false); err != nil {
				return err
			}
			fmt.Fprintf(c.OutOrStdout(), "%s\n", T.T(i18n.SummarySvcSyncDone, a[0], a[0], a[0]))
			return nil
		},
	}
}

func serviceStatusCmd(lc LaunchdClient) *cobra.Command {
	return &cobra.Command{
		Use:         "status NAME",
		Short:       T.T(i18n.SvcStatusShort),
		Annotations: map[string]string{annShortKey: i18n.SvcStatusShort},
		Args:        cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, a []string) error {
			label := lc.LabelFor(a[0])
			loaded := lc.IsLoaded(label)
			fmt.Fprintf(c.OutOrStdout(), "%s: %s\n", a[0], statusLabel(loaded))
			// env health check: can we resolve export lines for this service?
			cfg, err := loadCfg()
			if err != nil {
				return err
			}
			envDir, _ := paths.EnvDir()
			_, err = env.New(cfg, envDir, env.NewSecurityKeyChain()).Export(env.ExportContext{ServiceName: a[0]})
			if err != nil {
				fmt.Fprintf(c.OutOrStdout(), "%s\n", T.T(i18n.EnvParseWarn, err))
			} else {
				fmt.Fprintf(c.OutOrStdout(), "%s\n", T.T(i18n.EnvParseOK))
			}
			fmt.Fprintf(c.OutOrStdout(), "%s\n", T.T(i18n.SummarySvcStatusRelated, a[0], a[0]))
			return nil
		},
	}
}

func serviceListCmd(lc LaunchdClient) *cobra.Command {
	return &cobra.Command{
		Use:         "list",
		Short:       T.T(i18n.SvcListShort),
		Annotations: map[string]string{annShortKey: i18n.SvcListShort},
		RunE: func(c *cobra.Command, a []string) error {
			cfg, err := loadCfg()
			if err != nil {
				return err
			}
			names := make([]string, 0, len(cfg.Services))
			for name := range cfg.Services {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				label := lc.LabelFor(name)
				fmt.Fprintf(c.OutOrStdout(), "%s\t%s\n", name, statusLabel(lc.IsLoaded(label)))
			}
			fmt.Fprintf(c.OutOrStdout(), "%s\n", T.T(i18n.SummarySvcListRelated))
			return nil
		},
	}
}
