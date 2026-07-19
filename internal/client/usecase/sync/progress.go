package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
)

// Progress описывает текущий этап синхронизации для отображения в UI.
type Progress struct {
	Stage string // "vaults", "rows", "index"
	Done  int    // сколько обработано
	Total int    // сколько всего
}

func (p Progress) String() string {
	switch p.Stage {
	case "vaults":
		return "↻ Загрузка папок..."
	case "rows":
		return fmt.Sprintf("↻ Синхронизация папок (%d/%d)", p.Done, p.Total)
	case "index":
		return fmt.Sprintf("↻ Расширенная синхронизация (%d/%d)", p.Done, p.Total)
	default:
		return "↻ Синхронизация..."
	}
}

// SyncWithProgress работает как Sync(), но отправляет прогресс в канал.
// Канал НЕ закрывается вызывающей стороной — закрывается здесь по завершении.
func (u *UseCase) SyncWithProgress(ctx context.Context, progress chan<- Progress) error {
	defer close(progress)

	token, err := u.accessToken()
	if err != nil {
		return err
	}

	// Stage 1: CheckFreshness
	progress <- Progress{Stage: "vaults"}
	freshness, err := u.server.CheckFreshness(ctx, token)
	if err != nil {
		return err
	}

	local, err := u.vaultMap(ctx)
	if err != nil {
		return err
	}

	if u.needsVaultMeta(freshness, local) {
		if err := u.refreshVaultMeta(ctx, token); err != nil {
			return err
		}
		if local, err = u.vaultMap(ctx); err != nil {
			return err
		}
	}

	// Определяем какие vault'ы нужно синкать.
	var toSync []contracts.VaultVersion
	for _, fv := range freshness {
		lv, ok := local[fv.ID]
		if ok && !lv.SyncEnabled {
			continue
		}
		if ok && lv.SyncedVersion >= fv.Version {
			continue
		}
		toSync = append(toSync, fv)
	}

	// Stage 2: Pull rows (Tier 2a)
	for i, fv := range toSync {
		progress <- Progress{Stage: "rows", Done: i + 1, Total: len(toSync)}
		if u.syncDelayMs > 0 {
			time.Sleep(time.Duration(u.syncDelayMs) * time.Millisecond)
		}
		if err := u.pullVaultRows(ctx, token, fv.ID, fv.Version); err != nil {
			return err
		}
	}

	// Stage 3: Load indexes (Tier 2b) — для всех синхронизируемых vault'ов.
	if u.indexLoader != nil {
		syncedVaultIDs := make([]string, 0, len(toSync))
		for _, fv := range toSync {
			syncedVaultIDs = append(syncedVaultIDs, fv.ID)
		}
		for i, vid := range syncedVaultIDs {
			progress <- Progress{Stage: "index", Done: i + 1, Total: len(syncedVaultIDs)}
			if u.syncDelayMs > 0 {
				time.Sleep(time.Duration(u.syncDelayMs) * time.Millisecond)
			}
			if err := u.indexLoader.LoadIndexes(ctx, vid); err != nil {
				// Не фейлим весь sync из-за ошибки загрузки индекса (Tier 2b optional).
				continue
			}
		}
	}

	return nil
}
