package cli

import (
	"context"
	"fmt"

	clienti18n "github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/localstore"
)

// OutboxCmd — группа команд для инспекции оффлайн-очереди.
type OutboxCmd struct {
	List OutboxListCmd `cmd:"" help:"List pending offline operations."`
}

// OutboxListCmd показывает содержимое очереди (для проверки оффлайн-сценария).
type OutboxListCmd struct{}

func (c *OutboxListCmd) Run(local *localstore.Store, l *clienti18n.Localizer) error {
	entries, err := local.ListPendingOutbox(context.Background())
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Println(l.T("outbox_empty"))
		return nil
	}
	for _, e := range entries {
		fmt.Printf("#%d\t%s\t%s\t%s\t%s\n", e.ID, e.Op, e.Entity, e.EntityID, e.CreatedAt)
	}
	return nil
}
