package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth/mocks"
)

func TestSetupEncryption_Success(t *testing.T) {
	t.Parallel()

	users := mocks.NewMockRepository(t)
	users.EXPECT().GetByID(mock.Anything, "user-1").Return(domain.User{ID: "user-1"}, nil)
	users.EXPECT().
		UpdateEncKDF(mock.Anything, "user-1", []byte("salt"), []byte(`{"version":1}`), []byte("encmk")).
		Return(nil)

	err := newUseCase(users, mocks.NewMockTokenIssuer(t)).SetupEncryption(context.Background(), auth.SetupEncryptionParams{
		UserID:       "user-1",
		EncKDFSalt:   []byte("salt"),
		EncKDFParams: []byte(`{"version":1}`),
		EncMasterKey: []byte("encmk"),
	})

	require.NoError(t, err)
}

func TestSetupEncryption_UserNotFound(t *testing.T) {
	t.Parallel()

	users := mocks.NewMockRepository(t)
	users.EXPECT().GetByID(mock.Anything, "ghost").Return(domain.User{}, auth.ErrUserNotFound)

	err := newUseCase(users, mocks.NewMockTokenIssuer(t)).SetupEncryption(context.Background(), auth.SetupEncryptionParams{
		UserID: "ghost",
	})

	assert.ErrorIs(t, err, auth.ErrUserNotFound)
}

func TestSetupEncryption_UpdateError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("connection refused")
	users := mocks.NewMockRepository(t)
	users.EXPECT().GetByID(mock.Anything, "user-1").Return(domain.User{ID: "user-1"}, nil)
	users.EXPECT().UpdateEncKDF(mock.Anything, "user-1", mock.Anything, mock.Anything, mock.Anything).Return(wantErr)

	err := newUseCase(users, mocks.NewMockTokenIssuer(t)).SetupEncryption(context.Background(), auth.SetupEncryptionParams{
		UserID: "user-1",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
}
