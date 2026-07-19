package grpcserver_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/alexedwards/argon2id"

	pb "github.com/aikowocki/yandex-go-final-diploma/api/proto/gen/gophkeeper/v1"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	authusecase "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth"
	authmocks "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth/mocks"
	vaultmocks "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/vault/mocks"
)

// TestErrors_AuthErrorMapping прогоняет Register/Login через реальный auth usecase с
// репозиторием-моком, форсируя каждую ветку mapAuthErr через реальные сценарии ошибок.
func TestServer_Ping(t *testing.T) {
	srv := newAuthTestServer(t, authmocks.NewMockRepository(t), authmocks.NewMockRecoveryRepository(t), authmocks.NewMockTokenIssuer(t))
	resp, err := srv.Ping(context.Background(), &pb.PingRequest{})
	require.NoError(t, err)
	assert.Equal(t, "pong", resp.GetMessage())
}

func TestServer_SetupEncryption_Success(t *testing.T) {
	users := authmocks.NewMockRepository(t)
	users.EXPECT().GetByID(mock.Anything, "user-1").Return(domain.User{ID: "user-1"}, nil)
	users.EXPECT().UpdateEncKDF(mock.Anything, "user-1", []byte("salt"), []byte("params"), []byte("enc")).Return(nil)

	srv := newAuthTestServer(t, users, authmocks.NewMockRecoveryRepository(t), authmocks.NewMockTokenIssuer(t))
	_, err := srv.SetupEncryption(ctxWithUser("user-1"), &pb.SetupEncryptionRequest{
		EncKdfSalt: []byte("salt"), EncKdfParams: []byte("params"), EncMasterKey: []byte("enc"),
	})
	require.NoError(t, err)
}

func TestServer_SetupEncryption_NoUser(t *testing.T) {
	srv := newAuthTestServer(t, authmocks.NewMockRepository(t), authmocks.NewMockRecoveryRepository(t), authmocks.NewMockTokenIssuer(t))
	_, err := srv.SetupEncryption(context.Background(), &pb.SetupEncryptionRequest{})
	requireStatusCode(t, err, codes.NotFound)
}

func TestErrors_Register_LoginTaken(t *testing.T) {
	users := authmocks.NewMockRepository(t)
	users.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.User{}, authusecase.ErrLoginTaken)

	srv := newAuthTestServer(t, users, authmocks.NewMockRecoveryRepository(t), authmocks.NewMockTokenIssuer(t))
	_, err := srv.Register(context.Background(), &pb.RegisterRequest{Login: "alice", LoginCredential: []byte("pw")})
	requireStatusCode(t, err, codes.AlreadyExists)
}

func TestErrors_Register_EmptyLogin(t *testing.T) {
	srv := newAuthTestServer(t, authmocks.NewMockRepository(t), authmocks.NewMockRecoveryRepository(t), authmocks.NewMockTokenIssuer(t))
	_, err := srv.Register(context.Background(), &pb.RegisterRequest{LoginCredential: []byte("pw")})
	requireStatusCode(t, err, codes.InvalidArgument)
}

func TestErrors_Login_InvalidCredentials(t *testing.T) {
	hash, err := argon2id.CreateHash("correct-pw", argon2id.DefaultParams)
	require.NoError(t, err)

	users := authmocks.NewMockRepository(t)
	users.EXPECT().GetByLogin(mock.Anything, "alice").Return(domain.User{ID: "u1", Login: "alice", AuthHash: hash}, nil)

	srv := newAuthTestServer(t, users, authmocks.NewMockRecoveryRepository(t), authmocks.NewMockTokenIssuer(t))
	_, err = srv.Login(context.Background(), &pb.LoginRequest{Login: "alice", LoginCredential: []byte("wrong-pw")})
	requireStatusCode(t, err, codes.Unauthenticated)
}

func TestErrors_Login_UserNotFound(t *testing.T) {
	users := authmocks.NewMockRepository(t)
	users.EXPECT().GetByLogin(mock.Anything, "unknown").Return(domain.User{}, authusecase.ErrUserNotFound)

	srv := newAuthTestServer(t, users, authmocks.NewMockRecoveryRepository(t), authmocks.NewMockTokenIssuer(t))
	_, err := srv.Login(context.Background(), &pb.LoginRequest{Login: "unknown", LoginCredential: []byte("pw")})
	requireStatusCode(t, err, codes.Unauthenticated)
}

func TestErrors_RefreshToken_InvalidToken(t *testing.T) {
	tokens := authmocks.NewMockTokenIssuer(t)
	tokens.EXPECT().VerifyRefresh("bad-token").Return("", authusecase.ErrInvalidRefreshToken)

	srv := newAuthTestServer(t, authmocks.NewMockRepository(t), authmocks.NewMockRecoveryRepository(t), tokens)
	_, err := srv.RefreshToken(context.Background(), &pb.RefreshTokenRequest{RefreshToken: "bad-token"})
	requireStatusCode(t, err, codes.Unauthenticated)
}

// TestErrors_VaultInternalError покрывает default-ветку mapVaultErr (не одна из известных
// валидационных ошибок -> codes.Internal).
func TestErrors_Vault_InternalError(t *testing.T) {
	vaults := vaultmocks.NewMockRepository(t)
	vaults.EXPECT().ListByUser(mock.Anything, "user-1").Return(nil, assert.AnError)

	srv := newVaultTestServer(t, vaults)
	_, err := srv.ListVaults(ctxWithUser("user-1"), &pb.ListVaultsRequest{})
	requireStatusCode(t, err, codes.Internal)
}
