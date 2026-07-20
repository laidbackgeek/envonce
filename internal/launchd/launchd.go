// Package launchd wraps the macOS launchctl subcommands (bootstrap/bootout/print)
// with bounded retry/backoff, absorbing the I/O race (exit code 5) and treating
// an already-unloaded label (exit code 3) as success for idempotent operations.
package launchd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Runner abstracts os/exec command execution so tests can inject fakes.
type Runner interface {
	Run(name string, args ...string) ([]byte, error)
}

// realRunner invokes the real os/exec.Command.
type realRunner struct{}

func (realRunner) Run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if msg := bytes.TrimSpace(stderr.Bytes()); len(msg) > 0 {
			// Attach the child process's stderr to the error; otherwise only a
			// bare "exit status N" remains and the caller can't see launchctl's
			// real diagnostics (e.g. "Bootstrap failed: 5: ...").
			return stdout.Bytes(), fmt.Errorf("%w: %s", err, msg)
		}
		return stdout.Bytes(), err
	}
	return stdout.Bytes(), nil
}

// Service wraps the launchctl subcommands (bootstrap/bootout/print).
type Service struct {
	runner Runner
	uid    int
}

// New returns a Service using the real exec runner and the current user's uid.
func New() *Service {
	return &Service{runner: realRunner{}, uid: os.Getuid()}
}

// guiTarget returns the current user's GUI Aqua session target, e.g. "gui/501".
func (s *Service) guiTarget() string { return "gui/" + strconv.Itoa(s.uid) }

// bootstrapMaxAttempts is the retry limit for Bootstrap when launchctl returns exit code 5 (an I/O race).
const bootstrapMaxAttempts = 5

// bootoutMaxAttempts is the retry limit for Bootout when launchctl returns exit code 5 (an I/O race).
const bootoutMaxAttempts = 5

// sleep abstracts time.Sleep so tests can swap in a no-op and avoid real backoff slowing them down.
var sleep = time.Sleep

// exitCoder is implemented by *exec.ExitError; it unifies exit-code extraction across the fake and real runners.
type exitCoder interface{ ExitCode() int }

// Bootstrap loads the given plist into the current user's domain.
//
// launchctl bootout is asynchronous: a bootstrap of the same label right
// after it frequently hits "Bootstrap failed: 5: Input/output error". Exit
// code 5 is retried a bounded number of times with exponential backoff
// (0.5/1/2/4s) to absorb the race; other errors (corrupt plist, etc.) are
// returned immediately since retrying won't help.
func (s *Service) Bootstrap(plistPath string) error {
	var lastErr error
	for attempt := 0; attempt < bootstrapMaxAttempts; attempt++ {
		_, err := s.runner.Run("launchctl", "bootstrap", s.guiTarget(), plistPath)
		if err == nil {
			return nil
		}
		lastErr = err
		var ec exitCoder
		if !errors.As(err, &ec) || ec.ExitCode() != 5 {
			return err
		}
		if attempt < bootstrapMaxAttempts-1 {
			sleep(time.Duration(500*(1<<attempt)) * time.Millisecond)
		}
	}
	return lastErr
}

// Bootout unloads the given label from the current user's domain.
//
// Like Bootstrap, bootout can hit the I/O race (exit code 5), so it retries
// with bounded backoff. Exit code 3 ("No such process") means the label is
// already gone — the desired state is reached, so it's treated as success;
// this keeps idempotent operations like drop from erroring on an already
// unloaded label and leaving a stray job behind.
func (s *Service) Bootout(label string) error {
	var lastErr error
	for attempt := 0; attempt < bootoutMaxAttempts; attempt++ {
		_, err := s.runner.Run("launchctl", "bootout", s.guiTarget()+"/"+label)
		if err == nil {
			return nil
		}
		var ec exitCoder
		if !errors.As(err, &ec) {
			return err
		}
		switch ec.ExitCode() {
		case 3:
			return nil // label already gone, treat as success (idempotent)
		case 5:
			lastErr = err // I/O race, retry
		default:
			return err
		}
		if attempt < bootoutMaxAttempts-1 {
			sleep(time.Duration(500*(1<<attempt)) * time.Millisecond)
		}
	}
	return lastErr
}

// Print returns the launchctl print output for the given label.
func (s *Service) Print(label string) (string, error) {
	out, err := s.runner.Run("launchctl", "print", s.guiTarget()+"/"+label)
	return string(out), err
}

// IsLoaded reports whether the given label is loaded (print success counts as loaded).
func (s *Service) IsLoaded(label string) bool {
	_, err := s.runner.Run("launchctl", "print", s.guiTarget()+"/"+label)
	return err == nil
}

// LabelFor builds the standard launchd label for a service: com.envonce.<name>.
func (s *Service) LabelFor(name string) string { return "com.envonce." + name }

// LastExitNever is the literal launchctl prints for a job that has never exited.
const LastExitNever = "(never exited)"

// Liveness is the semantic, launchd-derived running state of one label. It exists
// because "loaded" ≠ "running": a job whose wrapper dies on every launch stays
// loaded (launchd keeps it registered) even though no live process exists, which
// used to be misreported as Running when only the load state was checked.
type Liveness int

const (
	// LivenessUnloaded: the label is not present in the user's launchd domain.
	LivenessUnloaded Liveness = iota
	// LivenessRunning: a live process exists (pid > 0). Because the wrapper ends in
	// `exec`, launchd's pid is the final target process, not an intermediate shell.
	LivenessRunning
	// LivenessCrashed: loaded, no live process, and the last exit was non-zero —
	// i.e. the wrapper keeps dying and launchd is throttle-restarting it.
	LivenessCrashed
	// LivenessIdle: loaded, no live process, and it has either never run or exited
	// cleanly (code 0) — e.g. a non-keepalive job that finished, or never started.
	LivenessIdle
)

// StatusInfo is the parsed, liveness-relevant subset of `launchctl print` output.
type StatusInfo struct {
	Loaded   bool   // true when the label is present in the domain (print succeeded)
	PID      int    // >0 means a live process; 0 means none
	LastExit string // raw "last exit code" value, e.g. "(never exited)", "0", "1", "127"
	Runs     int    // number of times launchd has started the job
}

// Liveness derives the semantic running state from the parsed fields.
func (s StatusInfo) Liveness() Liveness {
	switch {
	case !s.Loaded:
		return LivenessUnloaded
	case s.PID > 0:
		return LivenessRunning
	case s.LastExit != "" && s.LastExit != LastExitNever && s.LastExit != "0":
		return LivenessCrashed
	default:
		return LivenessIdle
	}
}

// ParseStatus extracts the liveness-relevant fields from `launchctl print` output.
// It does not set Loaded (the caller knows whether print succeeded); Inspect sets
// it after a successful print. Only the top-level pid / last exit code / runs keys
// are read, so nested sections (environment vars, per-spawn states) are ignored.
func ParseStatus(out []byte) StatusInfo {
	info := StatusInfo{}
	for _, line := range strings.Split(string(out), "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "pid":
			info.PID, _ = strconv.Atoi(strings.TrimSpace(v))
		case "last exit code":
			info.LastExit = strings.TrimSpace(v)
		case "runs":
			info.Runs, _ = strconv.Atoi(strings.TrimSpace(v))
		}
	}
	return info
}

// Inspect returns the liveness picture for a label: it runs `launchctl print` and
// parses it. A print error (label not loaded) yields Loaded=false along with the
// error, so callers can treat "not loaded" and "launchctl blew up" uniformly.
func (s *Service) Inspect(label string) (StatusInfo, error) {
	out, err := s.runner.Run("launchctl", "print", s.guiTarget()+"/"+label)
	if err != nil {
		return StatusInfo{Loaded: false}, err
	}
	info := ParseStatus(out)
	info.Loaded = true
	return info, nil
}
