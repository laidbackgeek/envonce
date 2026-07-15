// Package envfile parses, serializes, and edits the env.d/*.env files that store
// envonce's environment-variable groups (one KEY=VALUE per line, '#' comments).
package envfile

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Entry is a single KEY=VALUE line parsed from an env.d file; Line is its 1-based source line.
type Entry struct {
	Key   string
	Value string
	Line  int
}

var keyRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Parse parses the contents of an env.d file. Rules: lines starting with #
// are comments, blank lines are skipped, KEY=VALUE is split on the first =,
// KEY must be a valid identifier, and the value is kept verbatim (including
// @keychain: refs).
func Parse(content string) ([]Entry, error) {
	var out []Entry
	for i, raw := range strings.Split(content, "\n") {
		lineNo := i + 1
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			return nil, fmt.Errorf("env.d:%d: invalid line %q (missing '=')", lineNo, line)
		}
		key := strings.TrimSpace(line[:idx])
		if !keyRe.MatchString(key) {
			return nil, fmt.Errorf("env.d:%d: invalid KEY %q", lineNo, key)
		}
		value := line[idx+1:]
		out = append(out, Entry{Key: key, Value: value, Line: lineNo})
	}
	return out, nil
}

// Format serializes entries into env.d file text: one KEY=VALUE per line.
func Format(entries []Entry) string {
	var b strings.Builder
	for _, e := range entries {
		b.WriteString(e.Key)
		b.WriteByte('=')
		b.WriteString(e.Value)
		b.WriteByte('\n')
	}
	return b.String()
}

// Set sets key=value: updates in place if key exists, otherwise appends a new entry.
func Set(entries []Entry, key, value string) []Entry {
	for i, e := range entries {
		if e.Key == key {
			entries[i].Value = value
			return entries
		}
	}
	return append(entries, Entry{Key: key, Value: value})
}

// Unset removes every entry whose key matches.
func Unset(entries []Entry, key string) []Entry {
	out := entries[:0]
	for _, e := range entries {
		if e.Key != key {
			out = append(out, e)
		}
	}
	return out
}

// LoadFile reads and parses the env.d file at path.
func LoadFile(path string) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(string(data))
}

// SaveFile serializes entries and writes them to path (overwriting).
func SaveFile(path string, entries []Entry) error {
	return os.WriteFile(path, []byte(Format(entries)), 0o644)
}
