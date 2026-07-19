package secret

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
)

// ListOutboxConflicts возвращает outbox-записи типа secret со статусом conflict — их
// проигрывание (ReplayOutbox) остановилось из-за гонки версий с другим устройством/клиентом
// и требует явного разрешения пользователем.
func (u *UseCase) ListOutboxConflicts(ctx context.Context) ([]contracts.OutboxEntry, error) {
	entries, err := u.local.ListOutboxByStatus(ctx, contracts.OutboxStatusConflict)
	if err != nil {
		return nil, err
	}
	result := make([]contracts.OutboxEntry, 0, len(entries))
	for _, e := range entries {
		if e.Entity == "secret" {
			result = append(result, e)
		}
	}
	return result, nil
}

// ConflictFromOutbox восстанавливает *GenericConflict из outbox-записи со статусом conflict.
// Вместо отдельного RPC для получения серверной версии повторяет тот же update/delete запрос
// (тело у него уже есть, готовое к отправке) — сервер при повторной гонке версий вернёт тот же
// *grpcclient.ConflictError с актуальной серверной версией (все три тира), что и при обычном
// «живом» конфликте, так что дальше используется тот же путь построения карточки конфликта.
// Если конфликт уже разрешился сам — возвращает nil, nil.
func (u *UseCase) ConflictFromOutbox(ctx context.Context, entryID int64) (*GenericConflict, error) {
	entry, ok, err := u.local.GetOutbox(ctx, entryID)
	if err != nil {
		return nil, err
	}
	if !ok || entry.Entity != "secret" {
		return nil, ErrOutboxEntryNotFound
	}

	switch entry.Op {
	case contracts.OutboxOpUpdate:
		return u.conflictFromOutboxUpdate(ctx, entry)
	case contracts.OutboxOpDelete:
		return u.conflictFromOutboxDelete(ctx, entry)
	default:
		// create/blob_upload не участвуют в конфликтах версий.
		return nil, ErrOutboxEntryNotFound
	}
}

func (u *UseCase) conflictFromOutboxUpdate(ctx context.Context, e contracts.OutboxEntry) (*GenericConflict, error) {
	var p contracts.OutboxSecretUpdate
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return nil, fmt.Errorf("decode outbox update %d: %w", e.ID, err)
	}

	vaultKey, token, err := u.vaultContext(p.VaultID)
	if err != nil {
		return nil, err
	}

	mineVersion := p.BaseVersion + 1
	var mineRow, mineIndex, minePayload map[string]any
	if err := decryptTiers(u, vaultKey, p.VaultID, p.SecretID, mineVersion, p.EncRow, p.EncIndex, p.EncPayload,
		&mineRow, &mineIndex, &minePayload); err != nil {
		return nil, fmt.Errorf("decrypt mine version: %w", err)
	}

	_, err = u.server.UpdateSecret(ctx, token, p.SecretID, p.BaseVersion, p.EncRow, p.EncIndex, p.EncPayload)
	var conflict *grpcclient.ConflictError
	switch {
	case errors.As(err, &conflict):
		// ожидаемый путь — строим карточку конфликта ниже.
	case err == nil:
		// Сервер внезапно принял запись (например, кто-то уже разрешил конфликт руками
		// в этом же клиенте параллельно) — outbox-запись отработана, реального конфликта нет.
		if rerr := u.local.RemoveOutbox(ctx, e.ID); rerr != nil {
			return nil, rerr
		}
		return nil, u.cacheFullSecret(ctx, p.SecretID, p.VaultID, p.Type, p.EncRow, p.EncIndex, p.EncPayload, mineVersion, false)
	default:
		return nil, err
	}

	var srvRow, srvIndex, srvPayload map[string]any
	if err := decryptTiers(u, vaultKey, p.VaultID, p.SecretID, conflict.Server.Version,
		conflict.Server.EncRow, conflict.Server.EncIndex, conflict.Server.EncPayload,
		&srvRow, &srvIndex, &srvPayload); err != nil {
		return nil, fmt.Errorf("decrypt server version: %w", err)
	}

	server := conflict.Server
	secretType, secretID, vaultID := p.Type, p.SecretID, p.VaultID
	entryID := e.ID
	return &GenericConflict{
		SecretID:         secretID,
		VaultID:          vaultID,
		MineRow:          mineRow,
		MineIndex:        mineIndex,
		MinePayload:      minePayload,
		ServerRow:        srvRow,
		ServerIndex:      srvIndex,
		ServerPayload:    srvPayload,
		ServerVersion:    server.Version,
		ServerType:       server.Type,
		serverEncRow:     server.EncRow,
		serverEncIndex:   server.EncIndex,
		serverEncPayload: server.EncPayload,
		retryMine: func(ctx context.Context, baseVersion int64) (*GenericConflict, error) {
			return u.retryOutboxUpdate(ctx, entryID, p, secretType, baseVersion)
		},
	}, nil
}

func (u *UseCase) conflictFromOutboxDelete(ctx context.Context, e contracts.OutboxEntry) (*GenericConflict, error) {
	var p contracts.OutboxSecretDelete
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return nil, fmt.Errorf("decode outbox delete %d: %w", e.ID, err)
	}

	vaultKey, token, err := u.vaultContext(p.VaultID)
	if err != nil {
		return nil, err
	}

	err = u.server.DeleteSecret(ctx, token, p.SecretID, p.BaseVersion)
	var conflict *grpcclient.ConflictError
	switch {
	case errors.As(err, &conflict):
		// ожидаемый путь.
	case err == nil:
		if rerr := u.local.RemoveOutbox(ctx, e.ID); rerr != nil {
			return nil, rerr
		}
		return nil, u.local.DeleteSecret(ctx, p.SecretID)
	default:
		return nil, err
	}

	var srvRow, srvIndex, srvPayload map[string]any
	if err := decryptTiers(u, vaultKey, p.VaultID, p.SecretID, conflict.Server.Version,
		conflict.Server.EncRow, conflict.Server.EncIndex, conflict.Server.EncPayload,
		&srvRow, &srvIndex, &srvPayload); err != nil {
		return nil, fmt.Errorf("decrypt server version: %w", err)
	}

	server := conflict.Server
	secretID, vaultID, entryID := p.SecretID, p.VaultID, e.ID
	return &GenericConflict{
		SecretID:         secretID,
		VaultID:          vaultID,
		IsDelete:         true,
		ServerRow:        srvRow,
		ServerIndex:      srvIndex,
		ServerPayload:    srvPayload,
		ServerVersion:    server.Version,
		ServerType:       server.Type,
		serverEncRow:     server.EncRow,
		serverEncIndex:   server.EncIndex,
		serverEncPayload: server.EncPayload,
		retryMine: func(ctx context.Context, baseVersion int64) (*GenericConflict, error) {
			conflict, derr := u.DeleteSecret(ctx, vaultID, secretID, baseVersion)
			if derr != nil {
				return nil, derr
			}
			if conflict == nil {
				if rerr := u.local.RemoveOutbox(ctx, entryID); rerr != nil {
					return nil, rerr
				}
			}
			return conflict, nil
		},
	}, nil
}

// retryOutboxUpdate повторяет update outbox-записи с новым baseVersion (после ChoiceMine).
// Outbox хранит enc_row/enc_index/enc_payload зашифрованные под СТАРУЮ версию (p.BaseVersion+1)
// в AAD. При повторной отправке с новым baseVersion нужно перешифровать все тиры под новую
// версию (baseVersion+1) — иначе сервер сохранит enc-блобы под version=baseVersion+1, но
// фактический AAD внутри шифротекста будет от старой версии → decrypt при чтении сломается
func (u *UseCase) retryOutboxUpdate(ctx context.Context, entryID int64, p contracts.OutboxSecretUpdate, secretType int32, baseVersion int64) (*GenericConflict, error) {
	vaultKey, token, err := u.vaultContext(p.VaultID)
	if err != nil {
		return nil, err
	}

	// Расшифровываем тиры под исходную версию (ту, с которой они были зашифрованы в outbox).
	origVersion := p.BaseVersion + 1
	var row, index, payload map[string]any
	if err := decryptTiers(u, vaultKey, p.VaultID, p.SecretID, origVersion, p.EncRow, p.EncIndex, p.EncPayload, &row, &index, &payload); err != nil {
		return nil, fmt.Errorf("retryOutboxUpdate: decrypt original: %w", err)
	}

	// Перешифровываем под новую версию (baseVersion+1).
	newVersion := baseVersion + 1
	encRow, encIndex, encPayload, err := encryptTiers(u, vaultKey, p.VaultID, p.SecretID, newVersion, row, index, payload)
	if err != nil {
		return nil, fmt.Errorf("retryOutboxUpdate: re-encrypt: %w", err)
	}

	_, err = u.server.UpdateSecret(ctx, token, p.SecretID, baseVersion, encRow, encIndex, encPayload)
	if err != nil {
		if _, ok := errors.AsType[*grpcclient.ConflictError](err); ok {
			// Обновляем outbox-запись с новыми enc-блобами и baseVersion.
			p.BaseVersion = baseVersion
			p.EncRow = encRow
			p.EncIndex = encIndex
			p.EncPayload = encPayload
			body, merr := json.Marshal(p)
			if merr != nil {
				return nil, merr
			}
			if serr := u.local.SetOutboxStatus(ctx, entryID, contracts.OutboxStatusConflict); serr != nil {
				return nil, serr
			}
			// Обновляем payload записи (SetOutboxPayload не существует — пересоздаём).
			if rerr := u.local.RemoveOutbox(ctx, entryID); rerr != nil {
				return nil, rerr
			}
			newID, uerr := u.local.EnqueueOutbox(ctx, contracts.OutboxEntry{
				Op: contracts.OutboxOpUpdate, Entity: "secret", EntityID: p.SecretID,
				BaseVersion: baseVersion, Payload: body, Status: contracts.OutboxStatusConflict,
			})
			if uerr != nil {
				return nil, uerr
			}
			return u.ConflictFromOutbox(ctx, newID)
		}
		return nil, err
	}

	if err := u.local.RemoveOutbox(ctx, entryID); err != nil {
		return nil, err
	}
	return nil, u.cacheFullSecret(ctx, p.SecretID, p.VaultID, secretType, encRow, encIndex, encPayload, newVersion, false)
}
