package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/infra/postgres"
)

func TestTxManager_Do_CommitsOnSuccess(t *testing.T) {
	db := newTestDB(t)
	tx := postgres.NewTxManager(db)
	userRepo := postgres.NewUserRepo(db)

	err := tx.Do(context.Background(), func(ctx context.Context) error {
		_, err := userRepo.Create(ctx, domain.User{Login: "alice", AuthHash: "hash"})
		return err
	})
	require.NoError(t, err)

	_, err = userRepo.GetByLogin(context.Background(), "alice")
	require.NoError(t, err, "запись должна быть закоммичена")
}

func TestTxManager_Do_RollsBackOnError(t *testing.T) {
	db := newTestDB(t)
	tx := postgres.NewTxManager(db)
	userRepo := postgres.NewUserRepo(db)

	boom := errors.New("boom")
	err := tx.Do(context.Background(), func(ctx context.Context) error {
		if _, cerr := userRepo.Create(ctx, domain.User{Login: "bob", AuthHash: "hash"}); cerr != nil {
			return cerr
		}
		return boom
	})
	require.ErrorIs(t, err, boom)

	_, err = userRepo.GetByLogin(context.Background(), "bob")
	require.Error(t, err, "запись не должна быть закоммичена")
}

func TestTxManager_Do_NestedReusesSameTx(t *testing.T) {
	db := newTestDB(t)
	tx := postgres.NewTxManager(db)
	userRepo := postgres.NewUserRepo(db)

	err := tx.Do(context.Background(), func(ctx context.Context) error {
		// Вложенный Do в том же ctx должен использовать ту же транзакцию (не открывать новую).
		return tx.Do(ctx, func(ctx context.Context) error {
			_, err := userRepo.Create(ctx, domain.User{Login: "alice", AuthHash: "hash"})
			return err
		})
	})
	require.NoError(t, err)

	_, err = userRepo.GetByLogin(context.Background(), "alice")
	require.NoError(t, err)
}
