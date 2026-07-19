package sync

import (
	"context"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
)

// Sync подтягивает изменения с сервера в локальный кеш. Использует лёгкий CheckFreshness,
// чтобы не дёргать ListRow для папок, чья версия не изменилась с последнего синка.
func (u *UseCase) Sync(ctx context.Context) error {
	token, err := u.accessToken()
	if err != nil {
		return err
	}

	freshness, err := u.server.CheckFreshness(ctx, token)
	if err != nil {
		return err
	}

	local, err := u.vaultMap(ctx)
	if err != nil {
		return err
	}

	// Если сервер знает о папке, которой нет в локальном кеше — подтягиваем метаданные (Tier 1).
	if u.needsVaultMeta(freshness, local) {
		if err := u.refreshVaultMeta(ctx, token); err != nil {
			return err
		}
		if local, err = u.vaultMap(ctx); err != nil {
			return err
		}
	}

	for _, fv := range freshness {
		lv, ok := local[fv.ID]
		if ok && !lv.SyncEnabled {
			continue // пользователь явно отключил синхронизацию этого vault
		}
		if ok && lv.SyncedVersion >= fv.Version {
			continue // версия не изменилась — ListRow не вызываем
		}
		if err := u.pullVaultRows(ctx, token, fv.ID, fv.Version); err != nil {
			return err
		}
	}
	return nil
}

// SetVaultSyncEnabled переключает флаг «синхронизировать этот vault» в локальном кеше.
func (u *UseCase) SetVaultSyncEnabled(ctx context.Context, vaultID string, enabled bool) error {
	return u.local.SetVaultSyncEnabled(ctx, vaultID, enabled)
}

// kvSyncScopeChosen — kv-ключ, отмечающий, что пользователь уже прошёл экран выбора папок
// для синхронизации (показывается один раз при первом входе, когда на сервере есть папки).
const kvSyncScopeChosen = "sync.scope_chosen"

// SyncScopeChosen сообщает, показывался ли уже пользователю экран выбора папок для синка.
func (u *UseCase) SyncScopeChosen(ctx context.Context) bool {
	v, ok, err := u.local.KVGet(ctx, kvSyncScopeChosen)
	return err == nil && ok && len(v) > 0
}

// MarkSyncScopeChosen отмечает, что выбор сделан — экран больше не будет показываться
// повторно на этом устройстве/аккаунте (сбрасывается вместе с остальным кешом при смене
// аккаунта через WipeAccountData).
func (u *UseCase) MarkSyncScopeChosen(ctx context.Context) error {
	return u.local.KVSet(ctx, kvSyncScopeChosen, []byte("1"))
}

func (u *UseCase) vaultMap(ctx context.Context) (map[string]contracts.LocalVault, error) {
	vaults, err := u.local.ListVaults(ctx)
	if err != nil {
		return nil, err
	}
	m := make(map[string]contracts.LocalVault, len(vaults))
	for _, v := range vaults {
		m[v.ID] = v
	}
	return m, nil
}

func (u *UseCase) needsVaultMeta(freshness []contracts.VaultVersion, local map[string]contracts.LocalVault) bool {
	for _, fv := range freshness {
		if _, ok := local[fv.ID]; !ok {
			return true
		}
	}
	return false
}

// refreshVaultMeta тянет Tier 1 (список папок) и обновляет метаданные в кеше,
// сохраняя synced_version (UpsertVault его не трогает).
func (u *UseCase) refreshVaultMeta(ctx context.Context, token string) error {
	items, err := u.server.ListVaults(ctx, token)
	if err != nil {
		return err
	}
	for _, it := range items {
		if err := u.local.UpsertVault(ctx, contracts.LocalVault{
			ID:              it.ID,
			WrappedVaultKey: it.WrappedVaultKey,
			EncName:         it.EncName,
			Version:         it.Version,
		}); err != nil {
			return err
		}
	}
	return nil
}

// pullVaultRows тянет Tier 2a (enc_row) папку и кеширует строки, затем помечает synced_version.
func (u *UseCase) pullVaultRows(ctx context.Context, token, vaultID string, version int64) error {
	rows, err := u.server.ListSecretRows(ctx, token, vaultID)
	if err != nil {
		return err
	}

	pendingIDs, err := u.outstandingSecretIDs(ctx)
	if err != nil {
		return err
	}

	for _, r := range rows {
		if pendingIDs[r.ID] {
			continue
		}
		if err := u.local.UpsertSecretRow(ctx, contracts.LocalSecret{
			ID:      r.ID,
			VaultID: vaultID,
			Type:    r.Type,
			EncRow:  r.EncRow,
			Version: r.Version,
		}); err != nil {
			return fmt.Errorf("cache secret row %s: %w", r.ID, err)
		}
	}
	return u.local.SetVaultSyncedVersion(ctx, vaultID, version)
}

// outstandingSecretIDs возвращает множество secret_id, у которых есть outbox-запись со
// статусом pending или conflict — их локальная версия не должна затираться серверной до
// того, как запись будет проиграна (ReplayOutbox) или конфликт разрешён явно пользователем.
func (u *UseCase) outstandingSecretIDs(ctx context.Context) (map[string]bool, error) {
	pending, err := u.local.ListPendingOutbox(ctx)
	if err != nil {
		return nil, err
	}
	conflicts, err := u.local.ListOutboxByStatus(ctx, contracts.OutboxStatusConflict)
	if err != nil {
		return nil, err
	}
	ids := make(map[string]bool, len(pending)+len(conflicts))
	for _, e := range pending {
		if e.Entity == "secret" {
			ids[e.EntityID] = true
		}
	}
	for _, e := range conflicts {
		if e.Entity == "secret" {
			ids[e.EntityID] = true
		}
	}
	return ids, nil
}
