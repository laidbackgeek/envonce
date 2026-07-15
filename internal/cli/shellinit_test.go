package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShellInit_OutputsExports(t *testing.T) {
	setupHome(t)
	root := NewRootCmd()
	root.SetArgs([]string{"env", "set", "FOO=bar"})
	assert.NoError(t, root.Execute())

	root2 := NewRootCmd()
	out := &bytes.Buffer{}
	root2.SetOut(out)
	root2.SetArgs([]string{"shell-init"})
	assert.NoError(t, root2.Execute())
	assert.Contains(t, out.String(), "export FOO=bar")
}
