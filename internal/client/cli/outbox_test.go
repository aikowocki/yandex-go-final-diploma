package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/localstore"
)

func newOutboxTestLocal(t *testing.T) *localstore.Store {
	t.Helper()
	local, err := localstore.Open("", false)
	require.NoError(t, err)
	t.Cleanup(func() { _ = local.Close() })
	return local
}

func TestOutboxListCmd_Empty(t *testing.T) {
	local := newOutboxTestLocal(t)
	l := i18n.NewLocalizer(i18n.NewBundle(), "en")

	cmd := &OutboxListCmd{}
	require.NoError(t, cmd.Run(local, l))
}

func TestOutboxListCmd_WithPendingEntry(t *testing.T) {
	local := newOutboxTestLocal(t)
	l := i18n.NewLocalizer(i18n.NewBundle(), "en")

	_, err := local.EnqueueOutbox(context.Background(), contracts.OutboxEntry{
		Op: contracts.OutboxOpCreate, Entity: "secret", EntityID: "s1",
	})
	require.NoError(t, err)

	cmd := &OutboxListCmd{}
	require.NoError(t, cmd.Run(local, l))
}

func TestOutboxListCmd_WithConflictEntry(t *testing.T) {
	local := newOutboxTestLocal(t)
	l := i18n.NewLocalizer(i18n.NewBundle(), "en")

	id, err := local.EnqueueOutbox(context.Background(), contracts.OutboxEntry{
		Op: contracts.OutboxOpUpdate, Entity: "secret", EntityID: "s1",
	})
	require.NoError(t, err)
	require.NoError(t, local.SetOutboxStatus(context.Background(), id, contracts.OutboxStatusConflict))

	cmd := &OutboxListCmd{}
	require.NoError(t, cmd.Run(local, l))
}
