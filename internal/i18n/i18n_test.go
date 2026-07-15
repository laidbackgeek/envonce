package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetect_OverrideWins(t *testing.T) {
	t.Setenv("LANG", "en_US.UTF-8")
	assert.Equal(t, ZH, Detect("zh"))
	assert.Equal(t, EN, Detect("en"))
}

func TestDetect_FromEnv(t *testing.T) {
	t.Setenv("ENVONCE_LANG", "")
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "zh_CN.UTF-8")
	assert.Equal(t, ZH, Detect(""))
	t.Setenv("LANG", "en_US.UTF-8")
	assert.Equal(t, EN, Detect(""))
}

func TestDetect_DefaultEN(t *testing.T) {
	t.Setenv("LANG", "fr_FR.UTF-8")
	assert.Equal(t, EN, Detect(""))
}

func TestTranslator_T_ZH(t *testing.T) {
	tr := New(ZH)
	assert.Equal(t, "管理环境变量", tr.T(EnvShort))
}

func TestTranslator_T_EN(t *testing.T) {
	tr := New(EN)
	assert.Equal(t, "Manage environment variables", tr.T(EnvShort))
}

func TestTranslator_T_FallbackKey(t *testing.T) {
	tr := New(EN)
	assert.Equal(t, "nope", tr.T("nope")) // unknown key falls back to the key itself
}

func TestTranslator_T_WithArgs(t *testing.T) {
	tr := New(ZH)
	assert.Equal(t, "已写入 default.env: FOO=bar", tr.T(MsgWroteEnv, "default.env", "FOO=bar"))
}
