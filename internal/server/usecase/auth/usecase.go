package auth

import (
	"context"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
)

// Repository — хранение пользователей.
type Repository interface {
	Create(ctx context.Context, u domain.User) (domain.User, error)
	GetByLogin(ctx context.Context, login string) (domain.User, error)
	GetByID(ctx context.Context, id string) (domain.User, error)
	// UpdateEncKDF сохраняет enc_kdf_salt/enc_kdf_params/enc_master_key после SetupEncryption.
	UpdateEncKDF(ctx context.Context, userID string, salt, params, encMasterKey []byte) error
}

// RecoveryRepository — хранение recovery codes.
type RecoveryRepository interface {
	StoreCode(ctx context.Context, userID, codeID string, encMasterKey []byte) error
	GetBlob(ctx context.Context, userID, codeID string) ([]byte, error)
	MarkUsed(ctx context.Context, userID, codeID string) error
	DeleteAll(ctx context.Context, userID string) error
}

// TokenIssuer — выпуск/проверка/обновление JWT.
type TokenIssuer interface {
	Issue(userID string) (access, refresh string, err error)
	Verify(token string) (userID string, err error)
	VerifyRefresh(refreshToken string) (userID string, err error)
}

// TxManager выполняет fn как одну атомарную единицу работы в хранилище.
// Порт намеренно не знает про конкретную БД: usecase лишь решает, какие
// шаги должны быть атомарны, и прокидывает полученный ctx в репозитории.
type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

// UseCase реализует аутентификацию пользователей и настройку шифрования на сервере.
type UseCase struct {
	users    Repository
	recovery RecoveryRepository
	tokens   TokenIssuer
	tx       TxManager
}

// New создает UseCase аутентификации.
func New(users Repository, recovery RecoveryRepository, tokens TokenIssuer, tx TxManager) *UseCase {
	return &UseCase{users: users, recovery: recovery, tokens: tokens, tx: tx}
}
