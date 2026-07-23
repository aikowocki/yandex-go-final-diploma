package secret

import (
	"errors"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
)

// shouldFallbackOffline сообщает, что ошибка серверного вызова должна привести к офлайн-
// сохранению (outbox) вместо потери данных. Покрывает:
//   - ErrUnavailable — сеть/сервер физически недоступен (connection refused, timeout)
//   - ErrInvalidCredentials — токен протух (401 Unauthenticated) — данные не должны теряться,
//     пользователь перелогинится позже, а outbox доиграется при следующем sync.
func shouldFallbackOffline(err error) bool {
	return errors.Is(err, grpcclient.ErrUnavailable) ||
		errors.Is(err, grpcclient.ErrInvalidCredentials)
}
