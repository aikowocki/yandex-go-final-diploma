package cli

import (
	"context"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	clienti18n "github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/localstore"
)

// OutboxCmd — группа команд для инспекции оффлайн-очереди.
type OutboxCmd struct {
	List OutboxListCmd `cmd:"" help:"List pending offline operations and conflicts."`
}

// OutboxListCmd показывает содержимое очереди (для проверки оффлайн-сценария).
type OutboxListCmd struct{}

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
