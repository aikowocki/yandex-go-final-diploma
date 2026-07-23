package grpcclient_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	authusecase "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth"
	secretusecase "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret"
)

func TestGRPCClient_Ping(t *testing.T) {
	b := newTestGRPCClient(t)
	msg, err := b.Client.Ping(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "pong", msg)
}

func TestGRPCClient_Register_Success(t *testing.T) {
	b := newTestGRPCClient(t)
	b.Users.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.User{ID: "u1", Login: "alice"}, nil)

	tokens, err := b.Client.Register(context.Background(), "alice", []byte("pw"))
	require.NoError(t, err)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.Equal(t, "u1", tokens.UserID)
}

func TestGRPCClient_Register_LoginTaken(t *testing.T) {
	b := newTestGRPCClient(t)
	b.Users.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.User{}, authusecase.ErrLoginTaken)

	_, err := b.Client.Register(context.Background(), "alice", []byte("pw"))
	require.ErrorIs(t, err, grpcclient.ErrLoginTaken)
}

func TestGRPCClient_Login_InvalidCredentials(t *testing.T) {
	b := newTestGRPCClient(t)
	b.Users.EXPECT().GetByLogin(mock.Anything, "alice").Return(domain.User{}, authusecase.ErrUserNotFound)

	_, err := b.Client.Login(context.Background(), "alice", []byte("pw"))
	require.ErrorIs(t, err, grpcclient.ErrInvalidCredentials)
}

func TestGRPCClient_RefreshToken_InvalidToken(t *testing.T) {
	b := newTestGRPCClient(t)
	_, err := b.Client.RefreshToken(context.Background(), "not-a-real-token")
	require.Error(t, err)
	assert.ErrorIs(t, err, grpcclient.ErrInvalidCredentials)
}

func TestGRPCClient_SetupEncryption_NoUser(t *testing.T) {
	b := newTestGRPCClient(t)
	err := b.Client.SetupEncryption(context.Background(), "", []byte("s"), []byte("p"), []byte("m"))
	require.Error(t, err)
}

func TestGRPCClient_CreateVault_NoUser(t *testing.T) {
	b := newTestGRPCClient(t)
	_, err := b.Client.CreateVault(context.Background(), "", []byte("wvk"), []byte("name"))
	require.Error(t, err)
}

func TestGRPCClient_CreateVault_Success(t *testing.T) {
	b := newTestGRPCClient(t)
	tok := mustAccessToken(t, b)
	b.Vaults.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.Vault{ID: "v1"}, nil)

	id, err := b.Client.CreateVault(context.Background(), tok, []byte("wvk"), []byte("name"))
	require.NoError(t, err)
	assert.Equal(t, "v1", id)
}

func TestGRPCClient_ListVaults_Success(t *testing.T) {
	b := newTestGRPCClient(t)
	tok := mustAccessToken(t, b)
	b.Vaults.EXPECT().ListByUser(mock.Anything, mock.Anything).Return(nil, nil)

	items, err := b.Client.ListVaults(context.Background(), tok)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestGRPCClient_CheckFreshness_Success(t *testing.T) {
	b := newTestGRPCClient(t)
	tok := mustAccessToken(t, b)
	b.Vaults.EXPECT().CheckFreshness(mock.Anything, mock.Anything).Return(nil, nil)

	items, err := b.Client.CheckFreshness(context.Background(), tok)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestGRPCClient_CreateSecret_Success(t *testing.T) {
	b := newTestGRPCClient(t)
	tok := mustAccessToken(t, b)
	b.Owner.EXPECT().IsOwner(mock.Anything, "v1", mock.Anything).Return(true, nil)
	b.Secret.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, s domain.Secret) (domain.Secret, error) {
		return s, nil
	})
	b.Secret.EXPECT().BumpVaultVersion(mock.Anything, "v1").Return(nil)

	err := b.Client.CreateSecret(context.Background(), tok, "s1", "v1", 1, []byte("r"), []byte("i"), []byte("p"))
	require.NoError(t, err)
}

func TestGRPCClient_UpdateSecret_Conflict(t *testing.T) {
	b := newTestGRPCClient(t)
	tok := mustAccessToken(t, b)
	b.Secret.EXPECT().GetForUpdate(mock.Anything, "s1", mock.Anything).Return(domain.Secret{
		ID: "s1", VaultID: "v1", Version: 5, EncRow: []byte("srv-row"), EncIndex: []byte("srv-idx"), EncPayload: []byte("srv-payload"),
	}, nil)

	_, err := b.Client.UpdateSecret(context.Background(), tok, "s1", 3, []byte("r"), []byte("i"), []byte("p"))
	require.Error(t, err)
	var conflict *grpcclient.ConflictError
	require.True(t, errors.As(err, &conflict))
	assert.Equal(t, int64(5), conflict.Server.Version)
	assert.Equal(t, []byte("srv-row"), conflict.Server.EncRow)
}

func TestGRPCClient_UpdateSecret_Success(t *testing.T) {
	b := newTestGRPCClient(t)
	tok := mustAccessToken(t, b)
	b.Secret.EXPECT().GetForUpdate(mock.Anything, "s1", mock.Anything).Return(domain.Secret{
		ID: "s1", VaultID: "v1", Version: 3,
	}, nil)
	b.Secret.EXPECT().UpdateFields(mock.Anything, "s1", mock.Anything, mock.Anything, mock.Anything).Return(int64(4), nil)
	b.Secret.EXPECT().BumpVaultVersion(mock.Anything, "v1").Return(nil)

	version, err := b.Client.UpdateSecret(context.Background(), tok, "s1", 3, []byte("r"), []byte("i"), []byte("p"))
	require.NoError(t, err)
	assert.Equal(t, int64(4), version)
}

func TestGRPCClient_DeleteSecret_Success(t *testing.T) {
	b := newTestGRPCClient(t)
	tok := mustAccessToken(t, b)
	b.Secret.EXPECT().GetForUpdate(mock.Anything, "s1", mock.Anything).Return(domain.Secret{
		ID: "s1", VaultID: "v1", Version: 2,
	}, nil)
	b.Secret.EXPECT().SoftDelete(mock.Anything, "s1").Return(int64(3), nil)
	b.Secret.EXPECT().BumpVaultVersion(mock.Anything, "v1").Return(nil)

	require.NoError(t, b.Client.DeleteSecret(context.Background(), tok, "s1", 2))
}

func TestGRPCClient_DeleteSecret_Conflict(t *testing.T) {
	b := newTestGRPCClient(t)
	tok := mustAccessToken(t, b)
	b.Secret.EXPECT().GetForUpdate(mock.Anything, "s1", mock.Anything).Return(domain.Secret{
		ID: "s1", VaultID: "v1", Version: 9,
	}, nil)

	err := b.Client.DeleteSecret(context.Background(), tok, "s1", 2)
	require.Error(t, err)
	var conflict *grpcclient.ConflictError
	require.True(t, errors.As(err, &conflict))
	assert.Equal(t, int64(9), conflict.Server.Version)
}

func TestGRPCClient_ListSecretRows_Success(t *testing.T) {
	b := newTestGRPCClient(t)
	tok := mustAccessToken(t, b)
	b.Secret.EXPECT().ListRow(mock.Anything, "v1", mock.Anything).Return([]domain.Secret{
		{ID: "s1", Type: domain.SecretTypeText, Version: 1, EncRow: []byte("r1")},
	}, nil)

	items, err := b.Client.ListSecretRows(context.Background(), tok, "v1")
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "s1", items[0].ID)
}

func TestGRPCClient_ListSecretIndex_Success(t *testing.T) {
	b := newTestGRPCClient(t)
	tok := mustAccessToken(t, b)
	b.Secret.EXPECT().ListIndex(mock.Anything, "v1", mock.Anything).Return(nil, nil)

	items, err := b.Client.ListSecretIndex(context.Background(), tok, "v1")
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestGRPCClient_GetSecretPayload_NotFound(t *testing.T) {
	b := newTestGRPCClient(t)
	tok := mustAccessToken(t, b)
	b.Secret.EXPECT().GetPayload(mock.Anything, "s1", mock.Anything).Return(domain.Secret{}, secretusecase.ErrSecretNotFound)

	_, err := b.Client.GetSecretPayload(context.Background(), tok, "s1")
	require.ErrorIs(t, err, grpcclient.ErrNotFound)
}

func TestGRPCClient_GetSecretPayload_Success(t *testing.T) {
	b := newTestGRPCClient(t)
	tok := mustAccessToken(t, b)
	b.Secret.EXPECT().GetPayload(mock.Anything, "s1", mock.Anything).Return(domain.Secret{
		ID: "s1", Type: domain.SecretTypeText, Version: 1, EncPayload: []byte("p"),
	}, nil)

	item, err := b.Client.GetSecretPayload(context.Background(), tok, "s1")
	require.NoError(t, err)
	assert.Equal(t, []byte("p"), item.EncPayload)
}

func TestGRPCClient_AttachBlob_Success(t *testing.T) {
	b := newTestGRPCClient(t)
	tok := mustAccessToken(t, b)
	b.Secret.EXPECT().GetForUpdate(mock.Anything, "s1", mock.Anything).Return(domain.Secret{
		ID: "s1", VaultID: "v1", Version: 1, Type: domain.SecretTypeBinary,
	}, nil)
	b.Secret.EXPECT().AttachBlob(mock.Anything, "s1", "blob-ref", int64(2048)).Return(int64(1), nil)
	b.Secret.EXPECT().BumpVaultVersion(mock.Anything, "v1").Return(nil)

	version, err := b.Client.AttachBlob(context.Background(), tok, "s1", 1, "blob-ref", 2048)
	require.NoError(t, err)
	assert.Equal(t, int64(1), version)
}

func TestGRPCClient_StoreRecoveryCodes_Success(t *testing.T) {
	b := newTestGRPCClient(t)
	tok := mustAccessToken(t, b)
	b.Recovery.EXPECT().DeleteAll(mock.Anything, mock.Anything).Return(nil)
	b.Recovery.EXPECT().StoreCode(mock.Anything, mock.Anything, "code-a", []byte("enc-a")).Return(nil)

	err := b.Client.StoreRecoveryCodes(context.Background(), tok, []contracts.RecoveryCodeEntry{
		{CodeID: "code-a", EncMasterKey: []byte("enc-a")},
	})
	require.NoError(t, err)
}

func TestGRPCClient_GetRecoveryBlob_Success(t *testing.T) {
	b := newTestGRPCClient(t)
	tok := mustAccessToken(t, b)
	b.Recovery.EXPECT().GetBlob(mock.Anything, mock.Anything, "code-a").Return([]byte("blob"), nil)

	blob, err := b.Client.GetRecoveryBlob(context.Background(), tok, "code-a")
	require.NoError(t, err)
	assert.Equal(t, []byte("blob"), blob)
}

func TestGRPCClient_MarkRecoveryCodeUsed_Success(t *testing.T) {
	b := newTestGRPCClient(t)
	tok := mustAccessToken(t, b)
	b.Recovery.EXPECT().MarkUsed(mock.Anything, mock.Anything, "code-a").Return(nil)

	require.NoError(t, b.Client.MarkRecoveryCodeUsed(context.Background(), tok, "code-a"))
}

func mustAccessToken(t *testing.T, b *testServerBundle) string {
	t.Helper()
	access, _, err := b.Tokens.Issue("user-1")
	require.NoError(t, err)
	return access
}
