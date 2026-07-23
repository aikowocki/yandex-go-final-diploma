package auth_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth/mocks"
)

func newUseCaseWithRecovery(t *testing.T, recovery *mocks.MockRecoveryRepository) *auth.UseCase {
	t.Helper()
	return auth.New(mocks.NewMockRepository(t), recovery, mocks.NewMockTokenIssuer(t), passthroughTx{})
}

func TestStoreRecoveryCodes_Success(t *testing.T) {
	t.Parallel()

	recovery := mocks.NewMockRecoveryRepository(t)
	recovery.EXPECT().DeleteAll(mock.Anything, "user-1").Return(nil)
	recovery.EXPECT().StoreCode(mock.Anything, "user-1", "code-a", []byte("enc-a")).Return(nil)
	recovery.EXPECT().StoreCode(mock.Anything, "user-1", "code-b", []byte("enc-b")).Return(nil)

	uc := newUseCaseWithRecovery(t, recovery)
	err := uc.StoreRecoveryCodes(context.Background(), "user-1", []auth.RecoveryCodeEntry{
		{CodeID: "code-a", EncMasterKey: []byte("enc-a")},
		{CodeID: "code-b", EncMasterKey: []byte("enc-b")},
	})
	require.NoError(t, err)
}

func TestStoreRecoveryCodes_EmptyUserID(t *testing.T) {
	t.Parallel()

	uc := newUseCaseWithRecovery(t, mocks.NewMockRecoveryRepository(t))
	err := uc.StoreRecoveryCodes(context.Background(), "", nil)
	require.ErrorIs(t, err, auth.ErrEmptyUserID)
}

func TestStoreRecoveryCodes_DeleteAllError(t *testing.T) {
	t.Parallel()

	recovery := mocks.NewMockRecoveryRepository(t)
	recovery.EXPECT().DeleteAll(mock.Anything, "user-1").Return(assert.AnError)

	uc := newUseCaseWithRecovery(t, recovery)
	err := uc.StoreRecoveryCodes(context.Background(), "user-1", []auth.RecoveryCodeEntry{{CodeID: "c"}})
	require.ErrorIs(t, err, assert.AnError)
}

func TestStoreRecoveryCodes_StoreCodeError(t *testing.T) {
	t.Parallel()

	recovery := mocks.NewMockRecoveryRepository(t)
	recovery.EXPECT().DeleteAll(mock.Anything, "user-1").Return(nil)
	recovery.EXPECT().StoreCode(mock.Anything, "user-1", "code-a", mock.Anything).Return(assert.AnError)

	uc := newUseCaseWithRecovery(t, recovery)
	err := uc.StoreRecoveryCodes(context.Background(), "user-1", []auth.RecoveryCodeEntry{{CodeID: "code-a"}})
	require.ErrorIs(t, err, assert.AnError)
}

func TestGetRecoveryBlob_Success(t *testing.T) {
	t.Parallel()

	recovery := mocks.NewMockRecoveryRepository(t)
	recovery.EXPECT().GetBlob(mock.Anything, "user-1", "code-a").Return([]byte("blob"), nil)

	blob, err := newUseCaseWithRecovery(t, recovery).GetRecoveryBlob(context.Background(), "user-1", "code-a")
	require.NoError(t, err)
	assert.Equal(t, []byte("blob"), blob)
}

func TestGetRecoveryBlob_EmptyUserID(t *testing.T) {
	t.Parallel()

	_, err := newUseCaseWithRecovery(t, mocks.NewMockRecoveryRepository(t)).GetRecoveryBlob(context.Background(), "", "code-a")
	require.ErrorIs(t, err, auth.ErrEmptyUserID)
}

func TestMarkRecoveryCodeUsed_Success(t *testing.T) {
	t.Parallel()

	recovery := mocks.NewMockRecoveryRepository(t)
	recovery.EXPECT().MarkUsed(mock.Anything, "user-1", "code-a").Return(nil)

	err := newUseCaseWithRecovery(t, recovery).MarkRecoveryCodeUsed(context.Background(), "user-1", "code-a")
	require.NoError(t, err)
}

func TestMarkRecoveryCodeUsed_EmptyUserID(t *testing.T) {
	t.Parallel()

	err := newUseCaseWithRecovery(t, mocks.NewMockRecoveryRepository(t)).MarkRecoveryCodeUsed(context.Background(), "", "code-a")
	require.ErrorIs(t, err, auth.ErrEmptyUserID)
}
