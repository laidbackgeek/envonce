package env

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// fakeRunner replaces the security command runner
type fakeRunner struct {
	out string
	err error
	got []string
}

func (f *fakeRunner) Run(name string, args ...string) ([]byte, error) {
	f.got = append([]string{name}, args...)
	return []byte(f.out), f.err
}

func TestSecurityKeyChain_Resolve(t *testing.T) {
	f := &fakeRunner{out: "ghp_secret\n"}
	kc := &securityKeyChain{run: f.Run}
	got, err := kc.Resolve("github-token")
	assert.NoError(t, err)
	assert.Equal(t, "ghp_secret", got)
	assert.Equal(t, []string{"/usr/bin/security", "find-generic-password", "-s", "github-token", "-w"}, f.got)
}

func TestSecurityKeyChain_ResolveError(t *testing.T) {
	f := &fakeRunner{err: errors.New("exit 44")}
	kc := &securityKeyChain{run: f.Run}
	_, err := kc.Resolve("missing")
	assert.Error(t, err)
}
