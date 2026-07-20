package cli

import (
	"testing"

	"github.com/laidbackgeek/envonce/internal/i18n"
	"github.com/laidbackgeek/envonce/internal/launchd"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// TestApplyI18n_Idempotent verifies that applyI18n is idempotent (repeated calls yield the same result) and resets after switching language.
func TestApplyI18n_Idempotent(t *testing.T) {
	t.Cleanup(func() { T = i18n.New(i18n.EN) })

	T = i18n.New(i18n.ZH)
	root := &cobra.Command{Use: "root", Annotations: map[string]string{annShortKey: i18n.RootShort}}
	sub := &cobra.Command{Use: "sub", Annotations: map[string]string{annShortKey: i18n.SvcAddShort}}
	root.AddCommand(sub)

	applyI18n(root)
	zhRoot, zhSub := root.Short, sub.Short
	assert.NotEmpty(t, zhRoot)
	assert.NotEmpty(t, zhSub)

	applyI18n(root) // repeated call yields the same result (idempotent)
	assert.Equal(t, zhRoot, root.Short)
	assert.Equal(t, zhSub, sub.Short)

	T = i18n.New(i18n.EN) // after switching language, reset to English
	applyI18n(root)
	assert.NotEqual(t, zhRoot, root.Short)
	assert.NotEqual(t, zhSub, sub.Short)
}

// TestRenderLiveness_LangSwitch verifies that the status labels produced by
// renderLiveness (used by service status/list) switch with the language.
func TestRenderLiveness_LangSwitch(t *testing.T) {
	t.Cleanup(func() { T = i18n.New(i18n.EN) })

	running := launchd.StatusInfo{Loaded: true, PID: 1, LastExit: launchd.LastExitNever}
	unloaded := launchd.StatusInfo{Loaded: false}

	T = i18n.New(i18n.ZH)
	zhRun, _ := renderLiveness(running, "x")
	zhUnloaded, _ := renderLiveness(unloaded, "x")
	assert.Equal(t, "运行中 (pid=1)", zhRun)
	assert.Equal(t, "未加载", zhUnloaded)

	T = i18n.New(i18n.EN)
	enRun, _ := renderLiveness(running, "x")
	enUnloaded, _ := renderLiveness(unloaded, "x")
	assert.Equal(t, "Running (pid=1)", enRun)
	assert.Equal(t, "Not loaded", enUnloaded)
}
