package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroupCreate_List_Rename_Delete(t *testing.T) {
	setupHome(t)
	r := NewRootCmd()
	r.SetArgs([]string{"group", "create", "work"})
	assert.NoError(t, r.Execute())
	_, err := os.Stat(filepath.Join(mustConfigDir(t), "env.d", "work.env"))
	assert.NoError(t, err)

	r2 := NewRootCmd()
	r2.SetArgs([]string{"group", "rename", "work", "prod"})
	assert.NoError(t, r2.Execute())
	_, err = os.Stat(filepath.Join(mustConfigDir(t), "env.d", "prod.env"))
	assert.NoError(t, err)

	r3 := NewRootCmd()
	r3.SetArgs([]string{"group", "delete", "prod"})
	assert.NoError(t, r3.Execute())
	_, err = os.Stat(filepath.Join(mustConfigDir(t), "env.d", "prod.env"))
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}
