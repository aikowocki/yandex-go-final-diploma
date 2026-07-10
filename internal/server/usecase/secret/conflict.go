package secret

import (
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
)

type ErrConflict struct {
	Current domain.Secret
}

func (e *ErrConflict) Error() string {
	return fmt.Sprintf("secret: version conflict (server version %d)", e.Current.Version)
}
