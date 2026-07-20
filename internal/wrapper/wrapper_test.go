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

// TestGenerate_ArgsWithSpacesAreQuoted is the regression test for the bug where
// an arg containing spaces (e.g. a shell `-c "..."` command) was joined with
// bare spaces and got word-split by /bin/sh, so `zsh -c 'exec ccr serve ...'`
// degraded into `zsh -c exec ccr serve ...` and the target never ran.
func TestGenerate_ArgsWithSpacesAreQuoted(t *testing.T) {
	d := WrapperData{
		EnvonceBin:  "/opt/homebrew/bin/envonce",
		ServiceName: "ccr",
		Binary:      "/bin/zsh",
		Args:        []string{"-l", "-c", "exec ccr serve --daemon-child --no-open --host 127.0.0.1 --port 31171"},
	}
	got := Generate(d)
	// The multi-word arg must be single-quoted so sh treats it as exactly one word.
	assert.Contains(t, got, "exec /bin/zsh -l -c 'exec ccr serve --daemon-child --no-open --host 127.0.0.1 --port 31171' \"$@\"")
	// …and must NOT appear in the word-split, quote-stripped form.
	assert.NotContains(t, got, "exec /bin/zsh -l -c exec ccr serve")
}

func TestShellQuote(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", "''"},                                                              // empty arg preserved
		{"serve", "serve"},                                                      // bare safe word
		{"/opt/homebrew/opt/ollama/bin/ollama", "/opt/homebrew/opt/ollama/bin/ollama"},
		{"-c", "-c"},                                                            // leading dash, safe
		{"--flag=value", "--flag=value"},                                        // = is safe, stays bare
		{"host:port", "host:port"},                                              // : safe
		{"a,b@c%d", "a,b@c%d"},                                                  // , @ % safe
		{"echo hello world", "'echo hello world'"},                              // spaces → single-quoted
		{"$HOME", "'$HOME'"},                                                    // $ would expand → quoted
		{"back`tick", "'back`tick'"},                                            // backtick → quoted
		{`it's`, `'it'"'"'s'`},                                                  // embedded single quote
	}
	for _, c := range cases {
		assert.Equal(t, c.want, shellQuote(c.in), "input %q", c.in)
	}
}
