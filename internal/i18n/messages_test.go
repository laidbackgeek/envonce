package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCatalog_Complete verifies that every catalog key has non-empty ZH and EN text,
// guarding against missing translations (both applyI18n and T.T rely on both values being present).
func TestCatalog_Complete(t *testing.T) {
	for key, msgs := range catalog {
		zh, okZh := msgs[ZH]
		en, okEn := msgs[EN]
		assert.True(t, okZh && zh != "", "key %q missing non-empty ZH", key)
		assert.True(t, okEn && en != "", "key %q missing non-empty EN", key)
	}
}
