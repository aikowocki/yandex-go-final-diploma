package cli

import (
	"context"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	clienti18n "github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/localstore"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
	secretuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
)

// OutboxCmd — группа команд для инспекции оффлайн-очереди.
type OutboxCmd struct {
	List    OutboxListCmd    `cmd:"" help:"List pending offline operations and conflicts."`
	Resolve OutboxResolveCmd `cmd:"" help:"Resolve an outbox conflict (interactively pick mine/server)."`
}

// OutboxListCmd показывает содержимое очереди (для проверки оффлайн-сценария).
type OutboxListCmd struct{}

// Run выводит список отложенных офлайн-операций и конфликтов очереди outbox.
func (c *OutboxListCmd) Run(local *localstore.Store, l *clienti18n.Localizer) error {
	ctx := context.Background()

	pending, err := local.ListPendingOutbox(ctx)
	if err != nil {
		return err
	}
	conflicts, err := local.ListOutboxByStatus(ctx, contracts.OutboxStatusConflict)
	if err != nil {
		return err
	}

	if len(pending) == 0 && len(conflicts) == 0 {
		fmt.Println(l.T("outbox_empty"))
		return nil
	}

	for _, e := range pending {
		fmt.Printf("#%d\t%s\t%s\t%s\t%s\tpending\n", e.ID, e.Op, e.Entity, e.EntityID, e.CreatedAt)
	}
	if len(conflicts) > 0 {
		fmt.Println(l.T("outbox_conflict_note"))
		for _, e := range conflicts {
			fmt.Printf("#%d\t%s\t%s\t%s\t%s\tconflict\n", e.ID, e.Op, e.Entity, e.EntityID, e.CreatedAt)
		}
	}
	return nil
}

// OutboxResolveCmd разрешает outbox-запись со статусом conflict.
type OutboxResolveCmd struct {
	ID int64 `arg:"" help:"Outbox entry id (from 'outbox list')."`
}

// Run интерактивно разрешает конфликтную запись outbox по её id.
func (c *OutboxResolveCmd) Run(auth *authuc.UseCase, secret *secretuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}

	conflict, err := secret.ConflictFromOutbox(ctx, c.ID)
	if err != nil {
		return err
	}
	if conflict == nil {
		// Гонка разрешилась сама — запись уже убрана из очереди.
		fmt.Println(l.T("outbox_conflict_autoresolved"))
		return nil
	}
	return resolveGenericConflictInteractive(ctx, secret, l, conflict)
}
