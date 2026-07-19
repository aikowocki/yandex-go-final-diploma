package blob

import (
	"context"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
)

// SecretLookup — доступ к секретам, нужный blob usecase для проверки прав/владения.
type SecretLookup interface {
	GetForUpdate(ctx context.Context, secretID, userID string) (domain.Secret, error)
}

// UseCase реализует операции хранения двоичных секретов на сервере.
type UseCase struct {
	storage contracts.BlobStorage
	secrets SecretLookup
}

// New создает UseCase для работы с бинарными секретами.
func New(storage contracts.BlobStorage, secrets SecretLookup) *UseCase {
	return &UseCase{storage: storage, secrets: secrets}
}

// objectKey — детерминированный ключ объекта в MinIO: изолирует блобы по папке и секрету.
func objectKey(vaultID, secretID string) string {
	return vaultID + "/" + secretID
}
