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

// Localizer инкапсулирует goi18n Localizer. Хранит bundle и текущий язык, чтобы поддерживать
// смену языка в рантайме (SetLang) без пересоздания объекта — все держатели *Localizer
// (view-модели TUI) продолжают использовать тот же указатель и сразу видят новый язык.
type Localizer struct {
	bundle    *goi18n.Bundle
	lang      string
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

// Lang возвращает текущий язык локализатора.
func (l *Localizer) Lang() string {
	return l.lang
}

// SetLang переключает язык локализатора. Пустой lang игнорируется.
func (l *Localizer) SetLang(lang string) {
	if lang == "" || lang == l.lang {
		return
	}
	l.lang = lang
	l.localizer = goi18n.NewLocalizer(l.bundle, lang)
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
	return &Localizer{bundle: bundle, lang: lang, localizer: goi18n.NewLocalizer(bundle, lang)}
}
