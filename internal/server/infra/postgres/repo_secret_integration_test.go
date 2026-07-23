package postgres_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/infra/postgres"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret"
)

// createTestVault — вспомогательная функция: создаёт пользователя и его vault для тестов SecretRepo.
func createTestVault(t *testing.T, db *postgres.DB, login string) (user domain.User, vault domain.Vault) {
	t.Helper()
	user = createTestUser(t, db, login)
	vault, err := postgres.NewVaultRepo(db).Create(context.Background(), domain.Vault{
		UserID: user.ID, WrappedVaultKey: []byte("wvk"), EncName: []byte("name"),
	})
	require.NoError(t, err)
	return user, vault
}

func TestSecretRepo_CreateAndListRow(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	user, vault := createTestVault(t, db, "alice")
	repo := postgres.NewSecretRepo(db)

	created, err := repo.Create(ctx, domain.Secret{
		ID: "11111111-1111-1111-1111-111111111111", VaultID: vault.ID,
		Type: domain.SecretTypeLoginPassword, EncRow: []byte("row"), EncIndex: []byte("idx"), EncPayload: []byte("payload"),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), created.Version)

	rows, err := repo.ListRow(ctx, vault.ID, user.ID)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, created.ID, rows[0].ID)
	assert.Equal(t, []byte("row"), rows[0].EncRow)
}

func TestSecretRepo_GetForUpdate_AndUpdateFields(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	user, vault := createTestVault(t, db, "alice")
	repo := postgres.NewSecretRepo(db)

	created, err := repo.Create(ctx, domain.Secret{
		ID: "22222222-2222-2222-2222-222222222222", VaultID: vault.ID,
		Type: domain.SecretTypeText, EncRow: []byte("row"), EncIndex: []byte("idx"), EncPayload: []byte("payload"),
	})
	require.NoError(t, err)

	got, err := repo.GetForUpdate(ctx, created.ID, user.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), got.Version)

	newVersion, err := repo.UpdateFields(ctx, created.ID, []byte("row2"), []byte("idx2"), []byte("payload2"))
	require.NoError(t, err)
	assert.Equal(t, int64(2), newVersion)

	got2, err := repo.GetForUpdate(ctx, created.ID, user.ID)
	require.NoError(t, err)
	assert.Equal(t, []byte("row2"), got2.EncRow)
}

func TestSecretRepo_GetForUpdate_NotFound(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "alice")
	repo := postgres.NewSecretRepo(db)

	_, err := repo.GetForUpdate(context.Background(), "00000000-0000-0000-0000-000000000000", user.ID)
	require.ErrorIs(t, err, secret.ErrSecretNotFound)
}

func TestSecretRepo_SoftDelete(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	_, vault := createTestVault(t, db, "alice")
	repo := postgres.NewSecretRepo(db)

	created, err := repo.Create(ctx, domain.Secret{
		ID: "33333333-3333-3333-3333-333333333333", VaultID: vault.ID,
		Type: domain.SecretTypeText, EncRow: []byte("row"), EncIndex: []byte("idx"), EncPayload: []byte("payload"),
	})
	require.NoError(t, err)

	newVersion, err := repo.SoftDelete(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), newVersion)
}

func TestSecretRepo_AttachBlob(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	user, vault := createTestVault(t, db, "alice")
	repo := postgres.NewSecretRepo(db)

	created, err := repo.Create(ctx, domain.Secret{
		ID: "44444444-4444-4444-4444-444444444444", VaultID: vault.ID,
		Type: domain.SecretTypeBinary, EncRow: []byte("row"), EncIndex: []byte("idx"), EncPayload: []byte("payload"),
	})
	require.NoError(t, err)

	newVersion, err := repo.AttachBlob(ctx, created.ID, "blob-ref-1", 1024)
	require.NoError(t, err)
	// AttachBlob не инкрементирует version секрета (только blob_ref/blob_size).
	assert.Equal(t, int64(1), newVersion)

	got, err := repo.GetForUpdate(ctx, created.ID, user.ID)
	require.NoError(t, err)
	require.NotNil(t, got.BlobRef)
	assert.Equal(t, "blob-ref-1", *got.BlobRef)
	require.NotNil(t, got.BlobSize)
	assert.Equal(t, int64(1024), *got.BlobSize)
}

func TestSecretRepo_BumpVaultVersion(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	user, vault := createTestVault(t, db, "alice")
	secretRepo := postgres.NewSecretRepo(db)
	vaultRepo := postgres.NewVaultRepo(db)

	require.NoError(t, secretRepo.BumpVaultVersion(ctx, vault.ID))

	versions, err := vaultRepo.CheckFreshness(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	assert.Equal(t, int64(2), versions[0].Version)
}

func TestSecretRepo_ListIndex(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	user, vault := createTestVault(t, db, "alice")
	repo := postgres.NewSecretRepo(db)

	_, err := repo.Create(ctx, domain.Secret{
		ID: "55555555-5555-5555-5555-555555555555", VaultID: vault.ID,
		Type: domain.SecretTypeText, EncRow: []byte("row"), EncIndex: []byte("idx"), EncPayload: []byte("payload"),
	})
	require.NoError(t, err)

	items, err := repo.ListIndex(ctx, vault.ID, user.ID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, []byte("idx"), items[0].EncIndex)
}

func TestSecretRepo_GetPayload(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	user, vault := createTestVault(t, db, "alice")
	repo := postgres.NewSecretRepo(db)

	created, err := repo.Create(ctx, domain.Secret{
		ID: "66666666-6666-6666-6666-666666666666", VaultID: vault.ID,
		Type: domain.SecretTypeTOTP, EncRow: []byte("row"), EncIndex: []byte("idx"), EncPayload: []byte("payload"),
	})
	require.NoError(t, err)

	got, err := repo.GetPayload(ctx, created.ID, user.ID)
	require.NoError(t, err)
	assert.Equal(t, []byte("payload"), got.EncPayload)
	assert.Equal(t, domain.SecretTypeTOTP, got.Type)
}

func TestSecretRepo_GetPayload_NotFound(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "alice")
	repo := postgres.NewSecretRepo(db)

	_, err := repo.GetPayload(context.Background(), "00000000-0000-0000-0000-000000000000", user.ID)
	require.ErrorIs(t, err, secret.ErrSecretNotFound)
}

func TestSecretRepo_Create_InvalidVaultID(t *testing.T) {
	repo := postgres.NewSecretRepo(newTestDB(t))
	_, err := repo.Create(context.Background(), domain.Secret{
		ID: "77777777-7777-7777-7777-777777777777", VaultID: "not-a-uuid",
		Type: domain.SecretTypeText, EncRow: []byte("r"), EncIndex: []byte("i"), EncPayload: []byte("p"),
	})
	require.Error(t, err)
}
