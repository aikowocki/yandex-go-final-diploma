package grpcserver_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pb "github.com/aikowocki/yandex-go-final-diploma/api/proto/gen/gophkeeper/v1"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/transport/grpcserver"
	authmocks "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth/mocks"
	secretmocks "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret/mocks"
	vaultmocks "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/vault/mocks"
)

func newAuthTestServer(t *testing.T, users *authmocks.MockRepository, recovery *authmocks.MockRecoveryRepository, tokens *authmocks.MockTokenIssuer) *grpcserver.Server {
	t.Helper()
	return newTestServer(t, vaultmocks.NewMockRepository(t), secretmocks.NewMockRepository(t), secretmocks.NewMockVaultOwnership(t), users, recovery, tokens)
}

func TestRecoveryHandler_StoreRecoveryCodes_Success(t *testing.T) {
	recovery := authmocks.NewMockRecoveryRepository(t)
	recovery.EXPECT().DeleteAll(mock.Anything, "user-1").Return(nil)
	recovery.EXPECT().StoreCode(mock.Anything, "user-1", "code-a", []byte("enc-a")).Return(nil)

	srv := newAuthTestServer(t, authmocks.NewMockRepository(t), recovery, authmocks.NewMockTokenIssuer(t))
	_, err := srv.StoreRecoveryCodes(ctxWithUser("user-1"), &pb.StoreRecoveryCodesRequest{
		Codes: []*pb.RecoveryCodeEntry{{CodeId: "code-a", EncMasterKey: []byte("enc-a")}},
	})
	require.NoError(t, err)
}

func TestRecoveryHandler_StoreRecoveryCodes_NoUser(t *testing.T) {
	srv := newAuthTestServer(t, authmocks.NewMockRepository(t), authmocks.NewMockRecoveryRepository(t), authmocks.NewMockTokenIssuer(t))
	_, err := srv.StoreRecoveryCodes(context.Background(), &pb.StoreRecoveryCodesRequest{})
	requireStatusCode(t, err, codes.NotFound)
}

func TestRecoveryHandler_GetRecoveryBlob_Success(t *testing.T) {
	recovery := authmocks.NewMockRecoveryRepository(t)
	recovery.EXPECT().GetBlob(mock.Anything, "user-1", "code-a").Return([]byte("blob"), nil)

	srv := newAuthTestServer(t, authmocks.NewMockRepository(t), recovery, authmocks.NewMockTokenIssuer(t))
	resp, err := srv.GetRecoveryBlob(ctxWithUser("user-1"), &pb.GetRecoveryBlobRequest{CodeId: "code-a"})
	require.NoError(t, err)
	assert.Equal(t, []byte("blob"), resp.GetEncMasterKey())
}

func TestRecoveryHandler_GetRecoveryBlob_NoUser(t *testing.T) {
	srv := newAuthTestServer(t, authmocks.NewMockRepository(t), authmocks.NewMockRecoveryRepository(t), authmocks.NewMockTokenIssuer(t))
	_, err := srv.GetRecoveryBlob(context.Background(), &pb.GetRecoveryBlobRequest{})
	requireStatusCode(t, err, codes.NotFound)
}

func TestRecoveryHandler_MarkRecoveryCodeUsed_Success(t *testing.T) {
	recovery := authmocks.NewMockRecoveryRepository(t)
	recovery.EXPECT().MarkUsed(mock.Anything, "user-1", "code-a").Return(nil)

	srv := newAuthTestServer(t, authmocks.NewMockRepository(t), recovery, authmocks.NewMockTokenIssuer(t))
	_, err := srv.MarkRecoveryCodeUsed(ctxWithUser("user-1"), &pb.MarkRecoveryCodeUsedRequest{CodeId: "code-a"})
	require.NoError(t, err)
}

func TestRecoveryHandler_MarkRecoveryCodeUsed_NoUser(t *testing.T) {
	srv := newAuthTestServer(t, authmocks.NewMockRepository(t), authmocks.NewMockRecoveryRepository(t), authmocks.NewMockTokenIssuer(t))
	_, err := srv.MarkRecoveryCodeUsed(context.Background(), &pb.MarkRecoveryCodeUsedRequest{})
	requireStatusCode(t, err, codes.NotFound)
}
