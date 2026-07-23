package postgres_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/infra/postgres"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth"
)

func TestRecoveryRepo_StoreAndGetBlob(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	user := createTestUser(t, db, "alice")
	repo := postgres.NewRecoveryRepo(db)

	require.NoError(t, repo.StoreCode(ctx, user.ID, "code-1", []byte("enc-master-key")))

	blob, err := repo.GetBlob(ctx, user.ID, "code-1")
	require.NoError(t, err)
	assert.Equal(t, []byte("enc-master-key"), blob)
}

func TestRecoveryRepo_GetBlob_NotFound(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "alice")
	repo := postgres.NewRecoveryRepo(db)

	_, err := repo.GetBlob(context.Background(), user.ID, "unknown-code")
	require.Error(t, err)
}

func TestRecoveryRepo_GetBlob_InvalidUserID(t *testing.T) {
	repo := postgres.NewRecoveryRepo(newTestDB(t))
	_, err := repo.GetBlob(context.Background(), "not-a-uuid", "code-1")
	require.ErrorIs(t, err, auth.ErrUserNotFound)
}

func TestRecoveryRepo_MarkUsed_ExcludesFromGetBlob(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	user := createTestUser(t, db, "alice")
	repo := postgres.NewRecoveryRepo(db)

	require.NoError(t, repo.StoreCode(ctx, user.ID, "code-1", []byte("enc")))
	require.NoError(t, repo.MarkUsed(ctx, user.ID, "code-1"))

	_, err := repo.GetBlob(ctx, user.ID, "code-1")
	require.Error(t, err, "использованный код не должен возвращаться GetBlob")
}

func TestRecoveryRepo_MarkUsed_InvalidUserID(t *testing.T) {
	repo := postgres.NewRecoveryRepo(newTestDB(t))
	err := repo.MarkUsed(context.Background(), "not-a-uuid", "code-1")
	require.ErrorIs(t, err, auth.ErrUserNotFound)
}

func TestRecoveryRepo_DeleteAll(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	user := createTestUser(t, db, "alice")
	repo := postgres.NewRecoveryRepo(db)

	require.NoError(t, repo.StoreCode(ctx, user.ID, "code-1", []byte("enc1")))
	require.NoError(t, repo.StoreCode(ctx, user.ID, "code-2", []byte("enc2")))

	require.NoError(t, repo.DeleteAll(ctx, user.ID))

	_, err := repo.GetBlob(ctx, user.ID, "code-1")
	require.Error(t, err)
	_, err = repo.GetBlob(ctx, user.ID, "code-2")
	require.Error(t, err)
}

func TestRecoveryRepo_DeleteAll_InvalidUserID(t *testing.T) {
	repo := postgres.NewRecoveryRepo(newTestDB(t))
	err := repo.DeleteAll(context.Background(), "not-a-uuid")
	require.ErrorIs(t, err, auth.ErrUserNotFound)
}

func TestRecoveryRepo_StoreCode_InvalidUserID(t *testing.T) {
	repo := postgres.NewRecoveryRepo(newTestDB(t))
	err := repo.StoreCode(context.Background(), "not-a-uuid", "code-1", []byte("enc"))
	require.ErrorIs(t, err, auth.ErrUserNotFound)
}
