package i18n

import (
	"embed"
	"io/fs"
	"log/slog"

	"github.com/BurntSushi/toml"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed locales/*/*.toml
var localeFS embed.FS

// SupportedLangs список доступных языков.
var SupportedLangs = []string{"en", "ru"}

// Localizer инкапсулирует goi18n Localizer.
type Localizer struct {
	localizer *goi18n.Localizer
}

// T переводит сообщение по ключу id.
func (l *Localizer) T(id string) string {
	msg, err := l.localizer.Localize(&goi18n.LocalizeConfig{MessageID: id})
	if err != nil {
		slog.Warn("i18n: missing translation", "key", id)
		return id
	}
	return msg
}

// NewBundle создаёт и наполняет i18n-бандл переводами, встроенными в бинарник.
// Все файлы locales/*/*.toml подхватываются автоматически.
func NewBundle() *goi18n.Bundle {
	bundle := goi18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	paths, err := fs.Glob(localeFS, "locales/*/*.toml")
	if err != nil {
		panic("i18n: failed to glob locale files: " + err.Error())
	}

	for _, path := range paths {
		if _, err := bundle.LoadMessageFileFS(localeFS, path); err != nil {
			panic("i18n: failed to load locale file " + path + ": " + err.Error())
		}
	}

	return bundle
}

// NewLocalizer возвращает Localizer для переданного языка.
func NewLocalizer(bundle *goi18n.Bundle, lang string) *Localizer {
	return &Localizer{localizer: goi18n.NewLocalizer(bundle, lang)}
}
