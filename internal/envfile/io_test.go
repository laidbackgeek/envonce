package envfile

import (
	"errors"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSet_UpdateExisting(t *testing.T) {
	e := []Entry{{Key: "A", Value: "1", Line: 1}}
	got := Set(e, "A", "2")
	assert.Equal(t, "2", got[0].Value)
	assert.Len(t, got, 1)
}

func TestSet_AppendNew(t *testing.T) {
	got := Set(nil, "A", "1")
	assert.Equal(t, []Entry{{Key: "A", Value: "1"}}, got)
}

func TestUnset(t *testing.T) {
	e := []Entry{{Key: "A", Value: "1"}, {Key: "B", Value: "2"}}
	got := Unset(e, "A")
	assert.Len(t, got, 1)
	assert.Equal(t, "B", got[0].Key)
}

func TestSaveAndLoadFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "default.env")
	assert.NoError(t, SaveFile(p, []Entry{{Key: "X", Value: "1"}}))
	got, err := LoadFile(p)
	assert.NoError(t, err)
	assert.Equal(t, "1", got[0].Value)
}

func TestLoadFile_Missing(t *testing.T) {
	_, err := LoadFile(filepath.Join(t.TempDir(), "nope.env"))
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}

func TestFormat_RoundTrip(t *testing.T) {
	orig := "A=1\nB=2\n"
	got, err := Parse(orig)
	assert.NoError(t, err)
	assert.Equal(t, orig, Format(got))
}
