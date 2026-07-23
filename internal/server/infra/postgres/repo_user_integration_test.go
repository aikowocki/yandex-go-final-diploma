package postgres_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/infra/postgres"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth"
)

func TestUserRepo_CreateAndGetByLogin(t *testing.T) {
	ctx := context.Background()
	repo := postgres.NewUserRepo(newTestDB(t))

	created, err := repo.Create(ctx, domain.User{Login: "alice", AuthHash: "hash1"})
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)

	got, err := repo.GetByLogin(ctx, "alice")
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "alice", got.Login)
	assert.Equal(t, "hash1", got.AuthHash)
}

func TestUserRepo_CreateDuplicateLogin(t *testing.T) {
	ctx := context.Background()
	repo := postgres.NewUserRepo(newTestDB(t))

	_, err := repo.Create(ctx, domain.User{Login: "bob", AuthHash: "hash1"})
	require.NoError(t, err)

	_, err = repo.Create(ctx, domain.User{Login: "bob", AuthHash: "hash2"})
	require.ErrorIs(t, err, auth.ErrLoginTaken)
}

func TestUserRepo_GetByLogin_NotFound(t *testing.T) {
	repo := postgres.NewUserRepo(newTestDB(t))
	_, err := repo.GetByLogin(context.Background(), "unknown")
	require.ErrorIs(t, err, auth.ErrUserNotFound)
}

func TestUserRepo_GetByID(t *testing.T) {
	ctx := context.Background()
	repo := postgres.NewUserRepo(newTestDB(t))

	created, err := repo.Create(ctx, domain.User{Login: "alice", AuthHash: "hash1"})
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "alice", got.Login)
}

func TestUserRepo_GetByID_InvalidUUID(t *testing.T) {
	repo := postgres.NewUserRepo(newTestDB(t))
	_, err := repo.GetByID(context.Background(), "not-a-uuid")
	require.ErrorIs(t, err, auth.ErrUserNotFound)
}

func TestUserRepo_GetByID_NotFound(t *testing.T) {
	repo := postgres.NewUserRepo(newTestDB(t))
	_, err := repo.GetByID(context.Background(), "00000000-0000-0000-0000-000000000000")
	require.ErrorIs(t, err, auth.ErrUserNotFound)
}

func TestUserRepo_UpdateEncKDF(t *testing.T) {
	ctx := context.Background()
	repo := postgres.NewUserRepo(newTestDB(t))

	created, err := repo.Create(ctx, domain.User{Login: "alice", AuthHash: "hash1"})
	require.NoError(t, err)

	params := []byte(`{"version":1,"memory":8192,"iterations":1,"parallelism":1,"keyLen":32}`)
	require.NoError(t, repo.UpdateEncKDF(ctx, created.ID, []byte("salt"), params, []byte("encmk")))

	got, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, []byte("salt"), got.EncKDFSalt)
	// jsonb в Postgres переупорядочивает поля при хранении — сравниваем семантически.
	assert.JSONEq(t, string(params), string(got.EncKDFParams))
	assert.Equal(t, []byte("encmk"), got.EncMasterKey)
}

func TestUserRepo_UpdateEncKDF_InvalidUUID(t *testing.T) {
	repo := postgres.NewUserRepo(newTestDB(t))
	err := repo.UpdateEncKDF(context.Background(), "not-a-uuid", nil, nil, nil)
	require.ErrorIs(t, err, auth.ErrUserNotFound)
}
