package postgres_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/infra/postgres"
)

// createTestUser — вспомогательная функция для создания владельца vault'а в тестах.
func createTestUser(t *testing.T, db *postgres.DB, login string) domain.User {
	t.Helper()
	u, err := postgres.NewUserRepo(db).Create(context.Background(), domain.User{Login: login, AuthHash: "hash"})
	require.NoError(t, err)
	return u
}

func TestVaultRepo_CreateAndIsOwner(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	user := createTestUser(t, db, "alice")
	repo := postgres.NewVaultRepo(db)

	created, err := repo.Create(ctx, domain.Vault{
		UserID: user.ID, WrappedVaultKey: []byte("wvk"), EncName: []byte("name"),
	})
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, int64(1), created.Version)

	owns, err := repo.IsOwner(ctx, created.ID, user.ID)
	require.NoError(t, err)
	assert.True(t, owns)
}

func TestVaultRepo_IsOwner_WrongUser(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	owner := createTestUser(t, db, "owner")
	other := createTestUser(t, db, "other")
	repo := postgres.NewVaultRepo(db)

	created, err := repo.Create(ctx, domain.Vault{UserID: owner.ID, WrappedVaultKey: []byte("k"), EncName: []byte("n")})
	require.NoError(t, err)

	owns, err := repo.IsOwner(ctx, created.ID, other.ID)
	require.NoError(t, err)
	assert.False(t, owns)
}

func TestVaultRepo_IsOwner_InvalidVaultID(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "alice")
	repo := postgres.NewVaultRepo(db)

	owns, err := repo.IsOwner(context.Background(), "not-a-uuid", user.ID)
	require.NoError(t, err)
	assert.False(t, owns)
}

func TestVaultRepo_ListByUser(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	user := createTestUser(t, db, "alice")
	repo := postgres.NewVaultRepo(db)

	_, err := repo.Create(ctx, domain.Vault{UserID: user.ID, WrappedVaultKey: []byte("k1"), EncName: []byte("n1")})
	require.NoError(t, err)
	_, err = repo.Create(ctx, domain.Vault{UserID: user.ID, WrappedVaultKey: []byte("k2"), EncName: []byte("n2")})
	require.NoError(t, err)

	vaults, err := repo.ListByUser(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, vaults, 2)
}

func TestVaultRepo_ListByUser_Empty(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "alice")
	repo := postgres.NewVaultRepo(db)

	vaults, err := repo.ListByUser(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Empty(t, vaults)
}

func TestVaultRepo_CheckFreshness(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	user := createTestUser(t, db, "alice")
	repo := postgres.NewVaultRepo(db)

	created, err := repo.Create(ctx, domain.Vault{UserID: user.ID, WrappedVaultKey: []byte("k"), EncName: []byte("n")})
	require.NoError(t, err)

	versions, err := repo.CheckFreshness(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	assert.Equal(t, created.ID, versions[0].ID)
	assert.Equal(t, int64(1), versions[0].Version)
}
