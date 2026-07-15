package env

import (
	"os/exec"
	"strings"
)

// KeyChainResolver resolves an @keychain:<ref> into its real value.
type KeyChainResolver interface {
	Resolve(ref string) (string, error)
}

// commandRunner abstracts os/exec for testing.
type commandRunner func(name string, args ...string) ([]byte, error)

type securityKeyChain struct {
	run commandRunner
}

// NewSecurityKeyChain returns a KeyChainResolver backed by /usr/bin/security
// (find-generic-password), used to resolve @keychain:<ref> values at runtime.
func NewSecurityKeyChain() KeyChainResolver {
	return &securityKeyChain{run: func(name string, args ...string) ([]byte, error) {
		return exec.Command(name, args...).Output()
	}}
}

func (s *securityKeyChain) Resolve(ref string) (string, error) {
	out, err := s.run("/usr/bin/security", "find-generic-password", "-s", ref, "-w")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
