package launchd

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type fakeRunner struct {
	calls [][]string
	outs  map[string][]byte
	errs  map[string]error
	// seq returns results in call order (taking precedence over outs/errs), to simulate "first N calls fail, the (N+1)th succeeds".
	seq []seqResult
}

type seqResult struct {
	out []byte
	err error
}

func (f *fakeRunner) Run(name string, args ...string) ([]byte, error) {
	key := name + " " + joinArgs(args)
	f.calls = append(f.calls, append([]string{name}, args...))
	if len(f.seq) > 0 {
		r := f.seq[0]
		f.seq = f.seq[1:]
		return r.out, r.err
	}
	if e, ok := f.errs[key]; ok {
		return nil, e
	}
	if o, ok := f.outs[key]; ok {
		return o, nil
	}
	return nil, nil
}

// fakeExitErr fakes a child-process exit code and satisfies the exitCoder interface (isomorphic to the real *exec.ExitError).
type fakeExitErr struct{ code int }

func (e fakeExitErr) Error() string { return fmt.Sprintf("exit status %d", e.code) }
func (e fakeExitErr) ExitCode() int { return e.code }

// exitErr is a constructor helper for fakeExitErr, so call sites write exitErr(5) instead of a struct literal.
func exitErr(code int) fakeExitErr { return fakeExitErr{code: code} }

func joinArgs(a []string) string {
	return strings.Join(a, " ")
}

func TestBootstrap_Command(t *testing.T) {
	f := &fakeRunner{}
	s := &Service{runner: f, uid: 501}
	_ = s.Bootstrap("/p.plist")
	assert.Len(t, f.calls, 1)
	assert.Equal(t, "launchctl", f.calls[0][0])
	assert.Contains(t, f.calls[0], "bootstrap")
	assert.Contains(t, f.calls[0], "gui/501")
	assert.Contains(t, f.calls[0], "/p.plist")
}

func TestBootout_Command(t *testing.T) {
	f := &fakeRunner{}
	s := &Service{runner: f, uid: 501}
	_ = s.Bootout("com.envonce.ollama")
	assert.Contains(t, f.calls[0], "bootout")
	assert.Contains(t, f.calls[0], "gui/501/com.envonce.ollama")
}

func TestIsLoaded(t *testing.T) {
	f := &fakeRunner{errs: map[string]error{}}
	f.outs = map[string][]byte{"launchctl print gui/501/com.envonce.ollama": []byte("pid = 123")}
	s := &Service{runner: f, uid: 501}
	assert.True(t, s.IsLoaded("com.envonce.ollama"))

	f2 := &fakeRunner{errs: map[string]error{"launchctl print gui/501/com.envonce.ollama": errors.New("no")}}
	s2 := &Service{runner: f2, uid: 501}
	assert.False(t, s2.IsLoaded("com.envonce.ollama"))
}

func TestLabelFor(t *testing.T) {
	s := New()
	assert.Equal(t, "com.envonce.ollama", s.LabelFor("ollama"))
}

func TestBootstrap_RetriesOnExitFive(t *testing.T) {
	sleep = func(time.Duration) {}
	t.Cleanup(func() { sleep = time.Sleep })

	f := &fakeRunner{seq: []seqResult{
		{err: exitErr(5)},
		{err: exitErr(5)},
		{}, // 3rd call succeeds
	}}
	s := &Service{runner: f, uid: 501}
	err := s.Bootstrap("/p.plist")
	assert.NoError(t, err)
	assert.Len(t, f.calls, 3, "should retry until the 3rd call succeeds")
}

func TestBootstrap_NoRetryOnOtherExitCode(t *testing.T) {
	sleep = func(time.Duration) {}
	t.Cleanup(func() { sleep = time.Sleep })

	f := &fakeRunner{seq: []seqResult{{err: exitErr(1)}}}
	s := &Service{runner: f, uid: 501}
	err := s.Bootstrap("/p.plist")
	assert.Error(t, err)
	assert.Len(t, f.calls, 1, "exit code 1 isn't a race; should not retry")
}

func TestBootstrap_RetriesExhausted(t *testing.T) {
	sleep = func(time.Duration) {}
	t.Cleanup(func() { sleep = time.Sleep })

	// seq unset; every call hits the errs map and always returns exit code 5
	f := &fakeRunner{errs: map[string]error{"launchctl bootstrap gui/501 /p.plist": exitErr(5)}}
	s := &Service{runner: f, uid: 501}
	err := s.Bootstrap("/p.plist")
	assert.Error(t, err)
	assert.Len(t, f.calls, bootstrapMaxAttempts, "should retry up to the limit then return the error")
}

func TestRealRun_IncludesStderr(t *testing.T) {
	_, err := realRunner{}.Run("sh", "-c", "echo boom 1>&2; exit 5")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "boom", "stderr should be attached to the error")
	assert.Contains(t, err.Error(), "exit status 5")
}

func TestBootout_RetriesOnExitFive(t *testing.T) {
	sleep = func(time.Duration) {}
	t.Cleanup(func() { sleep = time.Sleep })

	f := &fakeRunner{seq: []seqResult{
		{err: exitErr(5)},
		{err: exitErr(5)},
		{}, // 3rd call succeeds
	}}
	s := &Service{runner: f, uid: 501}
	assert.NoError(t, s.Bootout("com.envonce.x"))
	assert.Len(t, f.calls, 3, "should retry until the 3rd call succeeds")
}

func TestBootout_ExitThreeIsSuccess(t *testing.T) {
	sleep = func(time.Duration) {}
	t.Cleanup(func() { sleep = time.Sleep })

	// exit code 3 = label is gone, treated as success (idempotent), no retry
	f := &fakeRunner{seq: []seqResult{{err: exitErr(3)}}}
	s := &Service{runner: f, uid: 501}
	assert.NoError(t, s.Bootout("com.envonce.x"))
	assert.Len(t, f.calls, 1, "exit code 3 is success; should not retry")
}

func TestBootout_NoRetryOnOtherExitCode(t *testing.T) {
	sleep = func(time.Duration) {}
	t.Cleanup(func() { sleep = time.Sleep })

	f := &fakeRunner{seq: []seqResult{{err: exitErr(1)}}}
	s := &Service{runner: f, uid: 501}
	assert.Error(t, s.Bootout("com.envonce.x"))
	assert.Len(t, f.calls, 1, "exit code 1 should not retry")
}

func TestBootout_RetriesExhausted(t *testing.T) {
	sleep = func(time.Duration) {}
	t.Cleanup(func() { sleep = time.Sleep })

	// seq unset; every call hits the errs map and always returns exit code 5
	f := &fakeRunner{errs: map[string]error{"launchctl bootout gui/501/com.envonce.x": exitErr(5)}}
	s := &Service{runner: f, uid: 501}
	assert.Error(t, s.Bootout("com.envonce.x"))
	assert.Len(t, f.calls, bootoutMaxAttempts, "should retry up to the limit then return the error")
}

// realPrint is a trimmed slice of `launchctl print gui/$UID/com.envonce.X` output,
// kept as a fixture for ParseStatus tests.
const realPrint = `	program = /Users/x/.config/envonce/services/x.wrapper.sh
	program arguments = {
	}
	state = running
	pid = 3030
	last exit code = (never exited)
	runs = 1
	environment = {
		SSH_AUTH_SOCK => /var/run/...
		PATH => /usr/bin:/bin
	}
		state = active
`

func TestParseStatus_Running(t *testing.T) {
	info := ParseStatus([]byte(realPrint))
	assert.Equal(t, 3030, info.PID)
	assert.Equal(t, LastExitNever, info.LastExit)
	assert.Equal(t, 1, info.Runs)
	info.Loaded = true
	assert.Equal(t, LivenessRunning, info.Liveness())
}

func TestParseStatus_NestedStateIgnored(t *testing.T) {
	// the per-spawn "state = active" lines under environment must not be parsed
	// as fields; only pid / last exit code / runs are read.
	info := ParseStatus([]byte(realPrint))
	assert.Equal(t, 3030, info.PID, "pid parsed from top-level, nested state ignored")
}

func TestParseStatus_Crashed(t *testing.T) {
	info := ParseStatus([]byte("\tpid = 0\n\tlast exit code = 126\n\truns = 7\n"))
	info.Loaded = true
	assert.Equal(t, LivenessCrashed, info.Liveness(), "non-zero exit + no pid = crash-loop")
	assert.Equal(t, 7, info.Runs)
}

func TestParseStatus_CleanExitIsIdle(t *testing.T) {
	info := ParseStatus([]byte("\tlast exit code = 0\n\truns = 1\n"))
	info.Loaded = true
	assert.Equal(t, LivenessIdle, info.Liveness(), "exit 0 is a clean exit, not crashed")
}

func TestParseStatus_NeverExitedIsIdle(t *testing.T) {
	info := ParseStatus([]byte("\tlast exit code = (never exited)\n\truns = 0\n"))
	info.Loaded = true
	assert.Equal(t, LivenessIdle, info.Liveness())
}

func TestInspect_Loaded(t *testing.T) {
	f := &fakeRunner{outs: map[string][]byte{
		"launchctl print gui/501/com.envonce.x": []byte("\tpid = 42\n\tlast exit code = (never exited)\n\truns = 1\n"),
	}}
	s := &Service{runner: f, uid: 501}
	info, err := s.Inspect("com.envonce.x")
	assert.NoError(t, err)
	assert.True(t, info.Loaded)
	assert.Equal(t, 42, info.PID)
	assert.Equal(t, LivenessRunning, info.Liveness())
	assert.Equal(t, []string{"launchctl", "print", "gui/501/com.envonce.x"}, f.calls[0])
}

func TestInspect_NotLoaded(t *testing.T) {
	f := &fakeRunner{errs: map[string]error{"launchctl print gui/501/com.envonce.x": exitErr(3)}}
	s := &Service{runner: f, uid: 501}
	info, err := s.Inspect("com.envonce.x")
	assert.Error(t, err)
	assert.False(t, info.Loaded)
	assert.Equal(t, LivenessUnloaded, info.Liveness())
}
