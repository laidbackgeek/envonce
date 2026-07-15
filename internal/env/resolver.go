// Package env resolves envonce's environment: it merges env.d/*.env groups,
// resolves @keychain: references via macOS Keychain, and emits export lines for
// shells and service wrappers.
package env

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/laidbackgeek/envonce/internal/config"
	"github.com/laidbackgeek/envonce/internal/envfile"
	"github.com/laidbackgeek/envonce/internal/paths"
)

// ExportContext describes an export request: either shell mode or a specific service.
type ExportContext struct {
	ForShell    bool
	ServiceName string
	Groups      []string // override groups in shell mode; falls back to cfg.GroupsForShell() when empty
}

// Resolver is the env-resolution core: it merges groups, reads env.d/*.env,
// resolves @keychain: refs, and emits export lines.
type Resolver struct {
	cfg    *config.Config
	envDir string
	kc     KeyChainResolver
}

// New builds a Resolver wired with the config, env directory, and KeyChainResolver.
func New(cfg *config.Config, envDir string, kc KeyChainResolver) *Resolver {
	return &Resolver{cfg: cfg, envDir: envDir, kc: kc}
}

func (r *Resolver) groupsFor(ctx ExportContext) ([]string, string) {
	if ctx.ForShell {
		if len(ctx.Groups) > 0 {
			return ctx.Groups, "shell"
		}
		return r.cfg.GroupsForShell(), "shell"
	}
	return r.cfg.GroupsForService(ctx.ServiceName), ctx.ServiceName
}

// Export merges env.d/*.env in group order (default first; later groups override
// earlier ones), resolves @keychain: refs (resolved values are single-quoted),
// and emits export KEY=value lines.
func (r *Resolver) Export(ctx ExportContext) ([]string, error) {
	groups, who := r.groupsFor(ctx)
	merged := map[string]string{}
	order := []string{}
	for _, g := range groups {
		entries, err := envfile.LoadFile(filepath.Join(r.envDir, g+".env"))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) { // missing group file = empty group (doctor warns separately)
				continue
			}
			return nil, err
		}
		for _, e := range entries {
			if _, exists := merged[e.Key]; exists {
				logConflict(e.Key, g)
			}
			if _, ok := merged[e.Key]; !ok {
				order = append(order, e.Key)
			}
			merged[e.Key] = e.Value
		}
	}
	var lines []string
	for _, k := range order {
		v := merged[k]
		if strings.HasPrefix(v, "@keychain:") {
			ref := strings.TrimPrefix(v, "@keychain:")
			resolved, err := r.kc.Resolve(ref)
			if err != nil {
				return nil, fmt.Errorf("%s: %s=%s failed to resolve: %v", who, k, v, err)
			}
			lines = append(lines, fmt.Sprintf("export %s=%s", k, shellSingleQuote(resolved)))
		} else {
			lines = append(lines, fmt.Sprintf("export %s=%s", k, v))
		}
	}
	return lines, nil
}

// shellSingleQuote wraps s in single quotes and escapes any embedded single quotes.
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// logConflict appends a KEY-conflict warning to logs/envonce.log.
// Best-effort: if the log path is unwritable or can't be resolved, it's
// silently ignored and never affects the resolution result.
func logConflict(key, group string) {
	dir, err := paths.LogsDir()
	if err != nil {
		return
	}
	// Ensure the log dir exists (OpenFile doesn't create parents).
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	logPath := filepath.Join(dir, "envonce.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	line := fmt.Sprintf("%s conflict: key %s in group %s overridden\n",
		time.Now().Format("2006/01/02 15:04:05"), key, group)
	_, _ = f.WriteString(line)
}
