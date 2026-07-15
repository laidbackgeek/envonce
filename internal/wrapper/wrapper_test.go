package wrapper

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerate_Golden(t *testing.T) {
	d := WrapperData{
		EnvonceBin:  "/opt/homebrew/bin/envonce",
		ServiceName: "ollama",
		Binary:      "/opt/homebrew/opt/ollama/bin/ollama",
		Args:        []string{"serve"},
	}
	got := Generate(d)
	want, _ := os.ReadFile("testdata/ollama.sh")
	assert.Equal(t, string(want), got)
}
