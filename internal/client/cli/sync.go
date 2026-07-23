package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
	clienti18n "github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
	syncuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/sync"
)

// SyncCmd синхронизирует локальный кеш с сервером и проигрывает оффлайн-очередь.
type SyncCmd struct{}

// Run синхронизирует локальный кеш с сервером и проигрывает оффлайн-очередь.
func (c *SyncCmd) Run(auth *authuc.UseCase, sync *syncuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}
	return runSync(ctx, sync, l)
}

// runSync прогоняет полный цикл синхронизации: pull изменений + flush outbox.
// Если сети нет — это не ошибка: сообщаем пользователю и выходим (синк произойдёт позже).
func runSync(ctx context.Context, sync *syncuc.UseCase, l *clienti18n.Localizer) error {
	if err := sync.Sync(ctx); err != nil {
		if errors.Is(err, grpcclient.ErrUnavailable) {
			fmt.Println(l.T("sync_offline"))
			return nil
		}
		return err
	}
	if err := sync.ReplayOutbox(ctx); err != nil {
		if errors.Is(err, grpcclient.ErrUnavailable) {
			fmt.Println(l.T("sync_offline"))
			return nil
		}
		return err
	}
	fmt.Println(l.T("sync_done"))
	return nil
}
