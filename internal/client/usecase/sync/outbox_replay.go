package sync

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
)

// ReplayOutbox проигрывает очередь оффлайн-изменений: для каждой записи вызывает
// соответствующий RPC и при успехе удаляет её из очереди. Если сеть снова недоступна —
// останавливается без ошибки, оставляя очередь для следующей попытки.
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
		if e.Op != contracts.OutboxOpCreate || e.Entity != "secret" {
			continue // TODO: update/delete + разрешение конфликтов
		}

		var p contracts.OutboxSecretCreate
		if err := json.Unmarshal(e.Payload, &p); err != nil {
			return fmt.Errorf("decode outbox %d: %w", e.ID, err)
		}

		serverID, err := u.server.CreateSecret(ctx, token, p.VaultID, p.Type, p.EncRow, p.EncIndex, p.EncPayload)
		if err != nil {
			if isOffline(err) {
				return nil // сеть снова недоступна — оставляем очередь как есть
			}
			return err
		}

		if err := u.reconcileCreated(ctx, p, serverID); err != nil {
			return err
		}
		if err := u.local.RemoveOutbox(ctx, e.ID); err != nil {
			return err
		}
	}
	return nil
}

// reconcileCreated заменяет временную локальную запись секрета на запись с серверным id
// и снимает флаг dirty (секрет успешно синхронизирован).
func (u *UseCase) reconcileCreated(ctx context.Context, p contracts.OutboxSecretCreate, serverID string) error {
	if err := u.local.UpsertSecretRow(ctx, contracts.LocalSecret{
		ID:            serverID,
		VaultID:       p.VaultID,
		Type:          p.Type,
		EncRow:        p.EncRow,
		EncIndex:      p.EncIndex,
		EncPayload:    p.EncPayload,
		Version:       1,
		IndexLoaded:   len(p.EncIndex) > 0,
		PayloadLoaded: len(p.EncPayload) > 0,
	}); err != nil {
		return err
	}
	if p.TempID != "" && p.TempID != serverID {
		return u.local.DeleteSecret(ctx, p.TempID)
	}
	return nil
}
