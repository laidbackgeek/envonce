package plist

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerate_Golden(t *testing.T) {
	d := PlistData{
		Label: "com.envonce.ollama", RunAtLoad: true, KeepAlive: true,
		ThrottleInterval: 10,
		WrapperPath:      "/Users/example/.config/envonce/services/ollama.wrapper.sh",
		StdoutPath:       "/Users/example/.config/envonce/logs/ollama.out.log",
		StderrPath:       "/Users/example/.config/envonce/logs/ollama.err.log",
	}
	got, err := Generate(d)
	assert.NoError(t, err)
	want, _ := os.ReadFile("testdata/ollama.plist")
	assert.Equal(t, string(want), got)
}

func TestGenerate_EscapesXML(t *testing.T) {
	got, err := Generate(PlistData{Label: "a&b<c>", WrapperPath: "/x"})
	assert.NoError(t, err)
	assert.Contains(t, got, "a&amp;b&lt;c&gt;")
}
