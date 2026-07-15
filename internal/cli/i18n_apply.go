package cli

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// annotation keys: record which i18n message key backs each command Short/Long
// and flag.Usage. applyI18n uses them to reset these fields at runtime
// (after the language is known).
const (
	annShortKey = "envonce.short.key"
	annLongKey  = "envonce.long.key"
	annFlagKey  = "envonce.flag.key"
)

// applyI18n walks the command tree and, per annotation, resets Short/Long/flag.Usage
// to the current T language. Idempotent: pure assignment, repeated calls are consistent.
//
// Why this is needed: cobra evaluates Short/Long eagerly during NewRootCmd, when the
// global T is still the default EN, while language detection happens later in
// PersistentPreRunE. Worse, --help short-circuits PersistentPreRunE so it never runs.
// Hence this function is called from both PersistentPreRunE (normal execution) and the
// root HelpFunc (--help short-circuit).
func applyI18n(root *cobra.Command) {
	walkCmd(root, func(cmd *cobra.Command) {
		if k, ok := cmd.Annotations[annShortKey]; ok {
			cmd.Short = T.T(k)
		}
		if k, ok := cmd.Annotations[annLongKey]; ok {
			cmd.Long = T.T(k)
		}
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if ks, ok := f.Annotations[annFlagKey]; ok && len(ks) > 0 {
				f.Usage = T.T(ks[0])
			}
		})
	})
}

// walkCmd walks the command tree depth-first, calling fn on every command (including itself).
func walkCmd(cmd *cobra.Command, fn func(*cobra.Command)) {
	fn(cmd)
	for _, sub := range cmd.Commands() {
		walkCmd(sub, fn)
	}
}
