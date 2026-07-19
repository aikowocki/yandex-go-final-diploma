package sync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
)

// ReplayOutbox проигрывает очередь оффлайн-изменений: для каждой pending-записи вызывает
// соответствующий RPC. При успехе удаляет запись из очереди. Если сеть снова недоступна —
// останавливается без ошибки, оставляя очередь для следующей попытки. При конфликте версий
// запись НЕ удаляется, а помечается статусом conflict — требуется явное разрешение пользователем.
func (u *UseCase) ReplayOutbox(ctx context.Context) error {
	token, err := u.accessToken()
	if err != nil {
		return err
	}

	entries, err := u.local.ListPendingOutbox(ctx)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if e.Entity != "secret" {
			continue // vault-операции появятся позже
		}

		done, err := u.replayEntry(ctx, token, e)
		if err != nil {
			if isOffline(err) {
				return nil // сеть снова недоступна — оставляем очередь как есть
			}
			return err
		}
		if !done {
			continue // конфликт — запись помечена, разрешит пользователь
		}
		if err := u.local.RemoveOutbox(ctx, e.ID); err != nil {
			return err
		}
	}
	return nil
}

// replayEntry проигрывает одну запись. Возвращает done=true, если запись отработана и её можно
// удалить; done=false, если возник конфликт (запись помечена conflict и остаётся в очереди).
func (u *UseCase) replayEntry(ctx context.Context, token string, e contracts.OutboxEntry) (bool, error) {
	switch e.Op {
	case contracts.OutboxOpCreate:
		return u.replayCreate(ctx, token, e)
	case contracts.OutboxOpUpdate:
		return u.replayUpdate(ctx, token, e)
	case contracts.OutboxOpDelete:
		return u.replayDelete(ctx, token, e)
	case contracts.OutboxOpBlobUpload:
		return u.replayBlobUpload(ctx, e)
	default:
		return false, fmt.Errorf("replay outbox %d: unknown op %q", e.ID, e.Op)
	}
}

func (u *UseCase) replayCreate(ctx context.Context, token string, e contracts.OutboxEntry) (bool, error) {
	var p contracts.OutboxSecretCreate
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return false, fmt.Errorf("decode outbox create %d: %w", e.ID, err)
	}
	if err := u.server.CreateSecret(ctx, token, p.SecretID, p.VaultID, p.Type, p.EncRow, p.EncIndex, p.EncPayload); err != nil {
		return false, err
	}
	// Снимаем dirty у синхронизированной строки (id стабилен — remap не нужен).
	return true, u.local.UpsertSecretRow(ctx, contracts.LocalSecret{
		ID:            p.SecretID,
		VaultID:       p.VaultID,
		Type:          p.Type,
		EncRow:        p.EncRow,
		EncIndex:      p.EncIndex,
		EncPayload:    p.EncPayload,
		Version:       1,
		IndexLoaded:   len(p.EncIndex) > 0,
		PayloadLoaded: len(p.EncPayload) > 0,
	})
}

func (u *UseCase) replayUpdate(ctx context.Context, token string, e contracts.OutboxEntry) (bool, error) {
	var p contracts.OutboxSecretUpdate
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return false, fmt.Errorf("decode outbox update %d: %w", e.ID, err)
	}
	_, err := u.server.UpdateSecret(ctx, token, p.SecretID, p.BaseVersion, p.EncRow, p.EncIndex, p.EncPayload)
	if err != nil {
		return u.handleReplayConflict(ctx, e, err)
	}
	return true, u.local.UpsertSecretRow(ctx, contracts.LocalSecret{
		ID:            p.SecretID,
		VaultID:       p.VaultID,
		Type:          p.Type,
		EncRow:        p.EncRow,
		EncIndex:      p.EncIndex,
		EncPayload:    p.EncPayload,
		Version:       p.BaseVersion + 1,
		IndexLoaded:   len(p.EncIndex) > 0,
		PayloadLoaded: len(p.EncPayload) > 0,
	})
}

func (u *UseCase) replayDelete(ctx context.Context, token string, e contracts.OutboxEntry) (bool, error) {
	var p contracts.OutboxSecretDelete
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return false, fmt.Errorf("decode outbox delete %d: %w", e.ID, err)
	}
	if err := u.server.DeleteSecret(ctx, token, p.SecretID, p.BaseVersion); err != nil {
		return u.handleReplayConflict(ctx, e, err)
	}
	return true, u.local.DeleteSecret(ctx, p.SecretID)
}

func (u *UseCase) replayBlobUpload(ctx context.Context, e contracts.OutboxEntry) (bool, error) {
	if u.blobUploader == nil {
		return false, fmt.Errorf("replay outbox %d: blob uploader not configured", e.ID)
	}
	var p contracts.OutboxBlobUpload
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return false, fmt.Errorf("decode outbox blob_upload %d: %w", e.ID, err)
	}
	if err := u.blobUploader.RetryBlobUpload(ctx, p.SecretID, p.VaultID); err != nil {
		if isOffline(err) {
			return false, nil // сеть недоступна — оставляем в очереди
		}
		return false, err
	}
	return true, nil
}

// handleReplayConflict помечает запись conflict, если ошибка — конфликт версий; иначе пробрасывает.
func (u *UseCase) handleReplayConflict(ctx context.Context, e contracts.OutboxEntry, err error) (bool, error) {
	if _, ok := errors.AsType[*grpcclient.ConflictError](err); ok {
		if serr := u.local.SetOutboxStatus(ctx, e.ID, contracts.OutboxStatusConflict); serr != nil {
			return false, serr
		}
		return false, nil // не удаляем запись, ждём явного разрешения
	}
	return false, err
}
