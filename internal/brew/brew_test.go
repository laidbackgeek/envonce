package brew

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadBrewPlist_Ollama(t *testing.T) {
	data, _ := os.ReadFile("testdata/homebrew.ollama.plist")
	info, err := ReadBrewPlist("ollama", data)
	assert.NoError(t, err)
	assert.Equal(t, "/opt/homebrew/opt/ollama/bin/ollama", info.Binary)
	assert.Equal(t, []string{"serve"}, info.Args)
	assert.True(t, info.KeepAlive)
	assert.True(t, info.RunAtLoad)
	assert.Equal(t, "/opt/homebrew/var/log/ollama.log", info.StdoutLog)
	assert.Equal(t, map[string]string{
		"OLLAMA_FLASH_ATTENTION": "1",
		"OLLAMA_KV_CACHE_TYPE":   "q8_0",
	}, info.Env)
}

func TestNormalizeBinaryToOpt_AlreadyOpt(t *testing.T) {
	in := "/opt/homebrew/opt/ollama/bin/ollama"
	assert.Equal(t, in, NormalizeBinaryToOpt(in))
}

func TestNormalizeBinaryToOpt_CellarToOpt(t *testing.T) {
	in := "/opt/homebrew/Cellar/ollama/0.1.42/bin/ollama"
	assert.Equal(t, "/opt/homebrew/opt/ollama/bin/ollama", NormalizeBinaryToOpt(in))
}

func TestNormalizeBinaryToOpt_IntelPrefix(t *testing.T) {
	in := "/usr/local/Cellar/ollama/0.1.42/bin/ollama"
	assert.Equal(t, "/usr/local/opt/ollama/bin/ollama", NormalizeBinaryToOpt(in))
}
