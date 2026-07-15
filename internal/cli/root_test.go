package cli

import (
	"bytes"
	"testing"

	"github.com/laidbackgeek/envonce/internal/i18n"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestNewRootCmd_Help(t *testing.T) {
	root := NewRootCmd()
	// the root command's Use is the binary name
	assert.Equal(t, "envonce", root.Name())

	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetArgs([]string{"--lang", "en", "--help"})
	assert.NoError(t, root.Execute())
	// --help goes through the custom HelpFunc: after applyI18n, root Long is the English fallback (help shows Long when present)
	assert.Contains(t, out.String(), "frees env vars")
}

// With no args, RunE calls cmd.Help() which returns nil, so Execute returns 0.
func TestExecute_ZeroExitOnNoArgs(t *testing.T) {
	setupHome(t) // isolate HOME so the result doesn't depend on the host's ~/.config/envonce
	code := Execute()
	assert.Equal(t, 0, code)
}

// withRunnable attaches an empty subcommand to root and runs it,
// so root's PersistentPreRunE is triggered (a direct --help is short-circuited
// by cobra early, so PersistentPreRunE never runs).
func withRunnable(args []string) error {
	root := NewRootCmd()
	root.AddCommand(&cobra.Command{
		Use: "noop",
		Run: func(cmd *cobra.Command, args []string) {},
	})
	root.SetArgs(args)
	return root.Execute()
}

func TestRoot_LangFlag(t *testing.T) {
	// --lang zh
	assert.NoError(t, withRunnable([]string{"--lang", "zh", "noop"}))
	assert.Equal(t, i18n.ZH, T.Lang())

	// --lang en
	assert.NoError(t, withRunnable([]string{"--lang", "en", "noop"}))
	assert.Equal(t, i18n.EN, T.Lang())
}

func TestRoot_VersionFlag(t *testing.T) {
	SetVersion("1.2.3-test")
	t.Cleanup(func() { SetVersion("dev") })

	out := &bytes.Buffer{}
	root := NewRootCmd()
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"--version"})
	assert.NoError(t, root.Execute())
	assert.Contains(t, out.String(), "1.2.3-test")
	assert.Contains(t, out.String(), "envonce")
}

// TestHelp_LangSwitch verifies that after --help goes through the custom HelpFunc,
// a subcommand's Short switches with --lang.
// This is the core of full i18n: --help short-circuits PersistentPreRunE, and the HelpFunc covers for it.
func TestHelp_LangSwitch(t *testing.T) {
	t.Run("zh", func(t *testing.T) {
		root := NewRootCmd()
		out := &bytes.Buffer{}
		root.SetOut(out)
		root.SetArgs([]string{"--lang", "zh", "--help"})
		assert.NoError(t, root.Execute())
		assert.Contains(t, out.String(), "管理后台服务") // service Short in Chinese
	})
	t.Run("en", func(t *testing.T) {
		root := NewRootCmd()
		out := &bytes.Buffer{}
		root.SetOut(out)
		root.SetArgs([]string{"--lang", "en", "--help"})
		assert.NoError(t, root.Execute())
		assert.Contains(t, out.String(), "Manage background services")
		assert.NotContains(t, out.String(), "管理后台服务")
	})
}

// TestHelp_SubcommandLangSwitch verifies that a subcommand's --help (the service subtree) also switches with --lang.
func TestHelp_SubcommandLangSwitch(t *testing.T) {
	t.Run("zh", func(t *testing.T) {
		root := NewRootCmd()
		out := &bytes.Buffer{}
		root.SetOut(out)
		root.SetArgs([]string{"--lang", "zh", "service", "--help"})
		assert.NoError(t, root.Execute())
		assert.Contains(t, out.String(), "添加并启动一个后台服务") // svc add Short in Chinese
	})
	t.Run("en", func(t *testing.T) {
		root := NewRootCmd()
		out := &bytes.Buffer{}
		root.SetOut(out)
		root.SetArgs([]string{"--lang", "en", "service", "--help"})
		assert.NoError(t, root.Execute())
		assert.Contains(t, out.String(), "Add and start a background service")
	})
}

// TestError_LangSwitch verifies that cli error messages switch with --lang.
// It uses `env set NOEQ` to trigger ErrInvalidKV (arg validation runs before config is read, so it needs no environment).
func TestError_LangSwitch(t *testing.T) {
	run := func(lang string) error {
		root := NewRootCmd()
		root.SetOut(&bytes.Buffer{})
		root.SetErr(&bytes.Buffer{})
		root.SetArgs([]string{"--lang", lang, "env", "set", "NOEQ"})
		return root.Execute()
	}
	t.Run("zh", func(t *testing.T) {
		err := run("zh")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "参数须为 KEY=VALUE")
	})
	t.Run("en", func(t *testing.T) {
		err := run("en")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Argument must be KEY=VALUE")
	})
}
