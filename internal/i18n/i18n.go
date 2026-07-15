// Package i18n provides envonce's bilingual (Chinese/English) message catalog
// and a translator that resolves message keys to localized strings at runtime.
package i18n

import (
	"fmt"
	"os"
	"strings"
)

// Lang is a supported UI language.
type Lang string

const (
	// ZH is Chinese.
	ZH Lang = "zh"
	// EN is English.
	EN Lang = "en"
)

// Detect determines the language by priority: override > ENVONCE_LANG > LC_ALL > LC_MESSAGES > LANG > EN.
func Detect(override string) Lang {
	if o := strings.ToLower(strings.TrimSpace(override)); o != "" {
		if strings.HasPrefix(o, "zh") {
			return ZH
		}
		return EN
	}
	for _, key := range []string{"ENVONCE_LANG", "LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(key); v != "" {
			if strings.HasPrefix(strings.ToLower(v), "zh") {
				return ZH
			}
			return EN
		}
	}
	return EN
}

// Translator resolves message keys to localized strings for a single language.
type Translator struct{ lang Lang }

// New returns a Translator for the given language.
func New(lang Lang) *Translator { return &Translator{lang: lang} }

// Lang returns the translator's language.
func (t *Translator) Lang() Lang { return t.lang }

// T returns the localized text for key, formatting it with args when provided.
// Unknown keys fall back to the key itself.
func (t *Translator) T(key string, args ...any) string {
	if msgs, ok := catalog[key]; ok {
		if s, ok := msgs[t.lang]; ok {
			if len(args) > 0 {
				return fmt.Sprintf(s, args...)
			}
			return s
		}
	}
	return key
}
