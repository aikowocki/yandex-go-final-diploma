package sync

import (
	"context"
	"errors"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
)

// UseCase client-side синхронизация между локальным хранилищем и сервером.
type UseCase struct {
	server       contracts.ServerClient
	local        contracts.LocalStorage
	tokens       contracts.TokenStore
	blobUploader BlobUploader
	indexLoader  IndexLoader
	syncDelayMs  int
}

// BlobUploader — интерфейс для повторной загрузки blob'а из staging (реализуется secret.UseCase).
type BlobUploader interface {
	RetryBlobUpload(ctx context.Context, secretID, vaultID string) error
}

// IndexLoader — интерфейс для загрузки Tier 2b (Index) для vault'а (реализуется secret.UseCase).
type IndexLoader interface {
	LoadIndexes(ctx context.Context, vaultID string) error
}

// New созадает UseCase синхронизации.
func New(server contracts.ServerClient, local contracts.LocalStorage, tokens contracts.TokenStore) *UseCase {
	return &UseCase{server: server, local: local, tokens: tokens}
}

// SetBlobUploader устанавливает BlobUploader (вызывается после создания secret UseCase, чтобы
// избежать циклической зависимости при инициализации).
func (u *UseCase) SetBlobUploader(bu BlobUploader) {
	u.blobUploader = bu
}

// SetIndexLoader устанавливает IndexLoader для загрузки Tier 2b при sync.
func (u *UseCase) SetIndexLoader(il IndexLoader) {
	u.indexLoader = il
}

// SetSyncDelay устанавливает искусственную задержку между vault'ами (для демонстрации).
func (u *UseCase) SetSyncDelay(ms int) {
	u.syncDelayMs = ms
}

func (u *UseCase) accessToken() (string, error) {
	tokens, err := u.tokens.Load()
	if err != nil {
		return "", err
	}
	return tokens.AccessToken, nil
}

// isOffline сообщает, что ошибка вызвана недоступностью сети/сервера (Unavailable/DeadlineExceeded).
func isOffline(err error) bool {
	return errors.Is(err, grpcclient.ErrUnavailable)
}
