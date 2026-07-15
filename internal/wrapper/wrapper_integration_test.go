package wrapper

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// execShell returns a cmd that runs the given wrapper script via /bin/sh.
func execShell(wrapperPath string) *exec.Cmd {
	return exec.Command("sh", wrapperPath)
}

// TestIntegration_FailLoud_OnExportFailure proves the corrected wrapper fails
// loud: when `envonce env export` (here: /usr/bin/false) exits non-zero with no
// stdout, the wrapper MUST exit 1 and the service binary MUST NOT run.
//
// On the OLD (broken) wrapper (`if ! eval "$(...)"`), command substitution
// swallows the non-zero exit, eval "" returns 0, the guard passes, and the
// service binary (/usr/bin/true) runs → exit 0. This test fails on old code,
// passes on the fix.
func TestIntegration_FailLoud_OnExportFailure(t *testing.T) {
	dir := t.TempDir()
	wrapperPath := filepath.Join(dir, "x.wrapper.sh")
	content := Generate(WrapperData{
		EnvonceBin:  "/usr/bin/false", // export "command" exits 1, no stdout
		ServiceName: "x",
		Binary:      "/usr/bin/true", // the service — must NOT run
	})
	assert.NoError(t, os.WriteFile(wrapperPath, []byte(content), 0o755))

	cmd := execShell(wrapperPath)
	err := cmd.Run()
	assert.Error(t, err, "wrapper must exit non-zero when env export fails")
	if ee, ok := err.(*exec.ExitError); ok {
		assert.NotEqual(t, 0, ee.ExitCode(), "exit code must be non-zero")
	}
}

// TestIntegration_Success_RunsService proves the happy path: when env export
// succeeds and prints valid export lines, the wrapper evals them and execs the
// service binary, which inherits the exported env.
func TestIntegration_Success_RunsService(t *testing.T) {
	dir := t.TempDir()

	// helper script that plays the role of a successful `envonce env export`
	exporterPath := filepath.Join(dir, "fake-export.sh")
	exporter := "#!/bin/sh\nprintf 'export WRAPPER_TEST=ok\\n'\n"
	assert.NoError(t, os.WriteFile(exporterPath, []byte(exporter), 0o755))

	// service "binary" is a script that prints the env var then exits 0
	svcPath := filepath.Join(dir, "fake-svc.sh")
	svc := "#!/bin/sh\necho \"svc got WRAPPER_TEST=$WRAPPER_TEST\"\n"
	assert.NoError(t, os.WriteFile(svcPath, []byte(svc), 0o755))

	wrapperPath := filepath.Join(dir, "ok.wrapper.sh")
	content := Generate(WrapperData{
		EnvonceBin:  exporterPath,
		ServiceName: "ok",
		Binary:      svcPath,
	})
	assert.NoError(t, os.WriteFile(wrapperPath, []byte(content), 0o755))

	out, err := execShell(wrapperPath).CombinedOutput()
	assert.NoError(t, err, "wrapper must exit 0 on success; output: %s", out)
	assert.Contains(t, string(out), "svc got WRAPPER_TEST=ok")
}
