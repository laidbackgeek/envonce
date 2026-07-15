package envfile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse_Basic(t *testing.T) {
	in := "# comment\n\nFOO=bar\nGOPATH=$HOME/go\nTOKEN=@keychain:gh\nEMPTY=\n"
	got, err := Parse(in)
	assert.NoError(t, err)
	assert.Equal(t, []Entry{
		{Key: "FOO", Value: "bar", Line: 3},
		{Key: "GOPATH", Value: "$HOME/go", Line: 4},
		{Key: "TOKEN", Value: "@keychain:gh", Line: 5},
		{Key: "EMPTY", Value: "", Line: 6},
	}, got)
}

func TestParse_ValueContainsEquals(t *testing.T) {
	got, err := Parse("URL=a=b&c=d\n")
	assert.NoError(t, err)
	assert.Equal(t, "a=b&c=d", got[0].Value)
}

func TestParse_InvalidLine(t *testing.T) {
	_, err := Parse("NOTHING_HERE\n")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid line")
}

func TestParse_InvalidKey(t *testing.T) {
	_, err := Parse("1FOO=bar\n")
	assert.Error(t, err)
}
