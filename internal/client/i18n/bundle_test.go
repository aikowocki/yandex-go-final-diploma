package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBundle_LoadsAllLocales(t *testing.T) {
	bundle := NewBundle()
	assert.NotNil(t, bundle)
}

func TestNewLocalizer_TranslatesKnownKey(t *testing.T) {
	bundle := NewBundle()
	l := NewLocalizer(bundle, "en")

	assert.Equal(t, "en", l.Lang())
	assert.Equal(t, "i18n test to GophKeeper client", l.T("test"))
}

func TestLocalizer_T_UnknownKeyReturnsID(t *testing.T) {
	bundle := NewBundle()
	l := NewLocalizer(bundle, "en")

	assert.Equal(t, "does_not_exist_key", l.T("does_not_exist_key"))
}

func TestLocalizer_SetLang_SwitchesTranslation(t *testing.T) {
	bundle := NewBundle()
	l := NewLocalizer(bundle, "en")

	l.SetLang("ru")
	assert.Equal(t, "ru", l.Lang())
}

func TestLocalizer_SetLang_EmptyIsNoop(t *testing.T) {
	bundle := NewBundle()
	l := NewLocalizer(bundle, "en")

	l.SetLang("")
	assert.Equal(t, "en", l.Lang())
}

func TestLocalizer_SetLang_SameLangIsNoop(t *testing.T) {
	bundle := NewBundle()
	l := NewLocalizer(bundle, "en")

	l.SetLang("en")
	assert.Equal(t, "en", l.Lang())
}

func TestSupportedLangs_ContainsEnAndRu(t *testing.T) {
	assert.Contains(t, SupportedLangs, "en")
	assert.Contains(t, SupportedLangs, "ru")
}
