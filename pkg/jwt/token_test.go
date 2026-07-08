package jwt_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/pkg/jwt"
)

func newIssuer() *jwt.TokenIssuer {
	return jwt.New([]byte("test-secret"), time.Minute, time.Hour)
}

func TestIssueAndVerify(t *testing.T) {
	t.Parallel()

	issuer := newIssuer()

	access, refresh, err := issuer.Issue("user-123")
	require.NoError(t, err)
	assert.NotEmpty(t, access)
	assert.NotEmpty(t, refresh)
	assert.NotEqual(t, access, refresh)

	userID, err := issuer.Verify(access)
	require.NoError(t, err)
	assert.Equal(t, "user-123", userID)
}

func TestVerify_RejectsRefreshTokenAsAccess(t *testing.T) {
	t.Parallel()

	issuer := newIssuer()

	_, refresh, err := issuer.Issue("user-123")
	require.NoError(t, err)

	_, err = issuer.Verify(refresh)
	assert.ErrorIs(t, err, jwt.ErrWrongType)
}

func TestRefresh_RejectsAccessTokenAsRefresh(t *testing.T) {
	t.Parallel()

	issuer := newIssuer()

	access, _, err := issuer.Issue("user-123")
	require.NoError(t, err)

	_, _, err = issuer.Refresh(access)
	assert.ErrorIs(t, err, jwt.ErrWrongType)
}

func TestRefresh_IssuesNewValidPair(t *testing.T) {
	t.Parallel()

	issuer := newIssuer()

	_, refresh, err := issuer.Issue("user-123")
	require.NoError(t, err)

	newAccess, newRefresh, err := issuer.Refresh(refresh)
	require.NoError(t, err)
	assert.NotEmpty(t, newAccess)
	assert.NotEmpty(t, newRefresh)

	userID, err := issuer.Verify(newAccess)
	require.NoError(t, err)
	assert.Equal(t, "user-123", userID)
}

func TestVerify_RejectsExpiredToken(t *testing.T) {
	t.Parallel()

	issuer := jwt.New([]byte("test-secret"), -time.Minute, time.Hour) // access уже истёк в момент выпуска

	access, _, err := issuer.Issue("user-123")
	require.NoError(t, err)

	_, err = issuer.Verify(access)
	assert.ErrorIs(t, err, jwt.ErrInvalidToken)
}

func TestVerify_RejectsTokenFromDifferentSecret(t *testing.T) {
	t.Parallel()

	issuerA := jwt.New([]byte("secret-a"), time.Minute, time.Hour)
	issuerB := jwt.New([]byte("secret-b"), time.Minute, time.Hour)

	access, _, err := issuerA.Issue("user-123")
	require.NoError(t, err)

	_, err = issuerB.Verify(access)
	assert.ErrorIs(t, err, jwt.ErrInvalidToken)
}

func TestVerify_RejectsGarbageToken(t *testing.T) {
	t.Parallel()

	issuer := newIssuer()

	_, err := issuer.Verify("not-a-jwt-token")
	assert.ErrorIs(t, err, jwt.ErrInvalidToken)
}

func TestIssue_DifferentUsersGetDifferentSubjects(t *testing.T) {
	t.Parallel()

	issuer := newIssuer()

	accessA, _, err := issuer.Issue("user-a")
	require.NoError(t, err)

	accessB, _, err := issuer.Issue("user-b")
	require.NoError(t, err)

	userA, err := issuer.Verify(accessA)
	require.NoError(t, err)
	userB, err := issuer.Verify(accessB)
	require.NoError(t, err)

	assert.Equal(t, "user-a", userA)
	assert.Equal(t, "user-b", userB)
}
