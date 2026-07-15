package cli

import (
	"testing"

	"github.com/laidbackgeek/envonce/internal/i18n"
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

// TestStatusLabel_LangSwitch verifies that the status labels in service status/list switch with the language.
func TestStatusLabel_LangSwitch(t *testing.T) {
	t.Cleanup(func() { T = i18n.New(i18n.EN) })

	T = i18n.New(i18n.ZH)
	assert.Equal(t, "运行中", statusLabel(true))
	assert.Equal(t, "未加载", statusLabel(false))

	T = i18n.New(i18n.EN)
	assert.Equal(t, "Running", statusLabel(true))
	assert.Equal(t, "Not loaded", statusLabel(false))
}
