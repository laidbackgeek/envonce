package env

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/laidbackgeek/envonce/internal/config"
	"github.com/stretchr/testify/assert"
)

type fakeKC struct {
	t    *testing.T
	vals map[string]string
}

func (f fakeKC) Resolve(ref string) (string, error) {
	if v, ok := f.vals[ref]; ok {
		return v, nil
	}
	f.t.Fatalf("fakeKC: unexpected ref %q (not in vals)", ref) // should not be reached
	return "", nil
}

func writeEnv(t *testing.T, dir, name, content string) {
	t.Helper()
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	assert.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}

func TestExport_Shell_LiteralAndKeychain(t *testing.T) {
	dir := t.TempDir()
	writeEnv(t, dir, "default.env", "PATH=/bin\nTOKEN=@keychain:gh\n")
	cfg := config.Default()
	r := New(cfg, dir, fakeKC{t: t, vals: map[string]string{"gh": "ghp_x"}})
	lines, err := r.Export(ExportContext{ForShell: true})
	assert.NoError(t, err)
	assert.Equal(t, []string{
		"export PATH=/bin",
		"export TOKEN='ghp_x'",
	}, lines)
}

func TestExport_Service_MergesDefaultAndExtra(t *testing.T) {
	// isolate XDG so the conflict warning doesn't hit the real ~/.config/envonce/logs/
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	dir := t.TempDir()
	writeEnv(t, dir, "default.env", "A=1\nB=2\n")
	writeEnv(t, dir, "work.env", "B=22\nC=3\n")
	cfg := config.Default()
	cfg.Services["ollama"] = config.ServiceDef{Groups: []string{"work"}}
	r := New(cfg, dir, fakeKC{t: t})
	lines, err := r.Export(ExportContext{ServiceName: "ollama"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"export A=1", "export B=22", "export C=3"}, lines)
}

func TestExport_KeyConflict_LogsWarning(t *testing.T) {
	// isolate XDG_CONFIG_HOME so logs/envonce.log lands in the temp dir
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	dir := t.TempDir()
	writeEnv(t, dir, "default.env", "KEY=from-default\n")
	writeEnv(t, dir, "work.env", "KEY=from-work\n")
	cfg := config.Default()
	cfg.Services["ollama"] = config.ServiceDef{Groups: []string{"work"}}
	r := New(cfg, dir, fakeKC{t: t})
	lines, err := r.Export(ExportContext{ServiceName: "ollama"})
	assert.NoError(t, err)
	// the merge result is still last-wins — behavior unchanged
	assert.Equal(t, []string{"export KEY=from-work"}, lines)
	// and the warning was appended to envonce.log
	logBytes, err := os.ReadFile(filepath.Join(xdg, "envonce", "logs", "envonce.log"))
	assert.NoError(t, err)
	assert.Contains(t, string(logBytes), "conflict: key KEY in group work overridden")
	assert.Regexp(t, `\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} conflict:`, string(logBytes))
}

// TestExport_ShellGroupsOverride_NoMutation verifies that ctx.Groups takes effect in shell mode,
// and that cfg.Shell.Groups is no longer mutated (item 3 regression).
func TestExport_ShellGroupsOverride_NoMutation(t *testing.T) {
	dir := t.TempDir()
	writeEnv(t, dir, "default.env", "A=1\n")
	writeEnv(t, dir, "work.env", "B=2\n")
	cfg := config.Default() // Shell.Groups = ["default"]
	r := New(cfg, dir, fakeKC{t: t})
	lines, err := r.Export(ExportContext{ForShell: true, Groups: []string{"work"}})
	assert.NoError(t, err)
	assert.Equal(t, []string{"export B=2"}, lines)
	assert.Equal(t, []string{"default"}, cfg.Shell.Groups) // key point: cfg was not mutated
}

func TestExport_KeychainFailure_IsError(t *testing.T) {
	dir := t.TempDir()
	writeEnv(t, dir, "default.env", "TOKEN=@keychain:missing\n")
	r := New(config.Default(), dir, errKC{})
	_, err := r.Export(ExportContext{ForShell: true})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TOKEN")
}

type errKC struct{}

func (errKC) Resolve(ref string) (string, error) { return "", errBoom }

var errBoom = strErr("boom")

type strErr string

func (s strErr) Error() string { return string(s) }
