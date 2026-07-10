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
		if ok && lv.SyncedVersion >= fv.Version {
			continue // версия не изменилась — ListRow не вызываем
		}
		if err := u.pullVaultRows(ctx, token, fv.ID, fv.Version); err != nil {
			return err
		}
	}
	return nil
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
	for _, r := range rows {
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
