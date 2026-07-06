package providers

import (
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/config"
	clienti18n "github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
)

// NewLocalizer создаёт Localizer на основе языка из конфигурации клиента.
func NewLocalizer(cfg *config.ClientConfig) *clienti18n.Localizer {
	bundle := clienti18n.NewBundle()
	return clienti18n.NewLocalizer(bundle, cfg.Lang)
}
