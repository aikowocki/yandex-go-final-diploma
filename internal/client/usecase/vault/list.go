package vault

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
)

// List возвращает расшифрованные папки пользователя. Для каждой разворачивает VaultKey
// MasterKey'ом, расшифровывает имя и открывает папку в сессии (VaultKey понадобится для секретов).
func (u *UseCase) List(ctx context.Context) ([]DecryptedVault, error) {
	masterKey, ok := u.sess.MasterKey()
	if !ok {
		return nil, ErrLocked
	}

	token, err := u.accessToken()
	if err != nil {
		return nil, err
	}

	items, err := u.server.ListVaults(ctx, token)
	if err != nil {
		return nil, err
	}

	result := make([]DecryptedVault, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, it := range items {
		seen[it.ID] = struct{}{}

		vaultKey, err := u.cipher.UnwrapVaultKey(it.WrappedVaultKey, masterKey)
		if err != nil {
			// Не должно происходить для папок ТЕКУЩЕГО аккаунта (сервер прислал их только что),
			// но не валим всю операцию из-за одной записи — пропускаем и логируем
			slog.Warn("vault: unwrap vault key failed, skipping", "vault_id", it.ID, "err", err)
			continue
		}

		var name string
		if err := u.cipher.DecryptStruct(vaultKey, nil, it.EncName, &name); err != nil {
			slog.Warn("vault: decrypt vault name failed, skipping", "vault_id", it.ID, "err", err)
			continue
		}

		u.sess.OpenVault(it.ID, vaultKey)

		// Кешируем метаданные папки локально (шифротексты), чтобы secret-команды и sync
		// могли работать оффлайн. synced_version при этом не трогается (им управляет sync).
		if err := u.local.UpsertVault(ctx, contracts.LocalVault{
			ID:              it.ID,
			WrappedVaultKey: it.WrappedVaultKey,
			EncName:         it.EncName,
			Version:         it.Version,
		}); err != nil {
			return nil, fmt.Errorf("cache vault: %w", err)
		}

		// Сохранённый флаг sync_enabled читаем из локального кеша (UpsertVault выше его не
		// перетирает при ON CONFLICT) — иначе после ре-логина выбор пользователя потерялся бы.
		syncEnabled := true
		if lv, ok, gerr := u.local.GetVault(ctx, it.ID); gerr == nil && ok {
			syncEnabled = lv.SyncEnabled
		}
		result = append(result, DecryptedVault{ID: it.ID, Name: name, Version: it.Version, SyncEnabled: syncEnabled})
	}

	if err := u.pruneStaleVaults(ctx, seen); err != nil {
		return nil, err
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// pruneStaleVaults удаляет из локального кеша папки, которых сервер не вернул в List (seen) —
// сервер является источником правды для Tier 1. Покрывает как реально удалённые папки, так и
// «осиротевшие» записи от другого аккаунта/устаревшего кеша.
func (u *UseCase) pruneStaleVaults(ctx context.Context, seen map[string]struct{}) error {
	cached, err := u.local.ListVaults(ctx)
	if err != nil {
		return fmt.Errorf("list cached vaults: %w", err)
	}
	for _, cv := range cached {
		if _, ok := seen[cv.ID]; !ok {
			if err := u.local.DeleteVault(ctx, cv.ID); err != nil {
				return fmt.Errorf("prune stale vault %s: %w", cv.ID, err)
			}
		}
	}
	return nil
}

// ListLocal возвращает папки из локального кеша (без сети): разворачивает VaultKey MasterKey'ом,
// расшифровывает имена и открывает в сессии. Записи, которые не разворачиваются текущим
// MasterKey, пропускаются с предупреждением, а не обрушивают всю операцию: остальные папки
// пользователя должны остаться доступны.
func (u *UseCase) ListLocal(ctx context.Context) ([]DecryptedVault, error) {
	masterKey, ok := u.sess.MasterKey()
	if !ok {
		return nil, ErrLocked
	}

	items, err := u.local.ListVaults(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]DecryptedVault, 0, len(items))
	for _, it := range items {
		vaultKey, err := u.cipher.UnwrapVaultKey(it.WrappedVaultKey, masterKey)
		if err != nil {
			slog.Warn("vault: unwrap vault key failed for cached vault, skipping", "vault_id", it.ID, "err", err)
			continue
		}

		var name string
		if err := u.cipher.DecryptStruct(vaultKey, nil, it.EncName, &name); err != nil {
			slog.Warn("vault: decrypt cached vault name failed, skipping", "vault_id", it.ID, "err", err)
			continue
		}

		u.sess.OpenVault(it.ID, vaultKey)
		result = append(result, DecryptedVault{ID: it.ID, Name: name, Version: it.Version, SyncEnabled: it.SyncEnabled})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}
