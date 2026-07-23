package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsNoRows(t *testing.T) {
	assert.True(t, isNoRows(pgx.ErrNoRows))
	assert.False(t, isNoRows(errors.New("other")))
	assert.False(t, isNoRows(nil))
}

func TestUniqueViolation(t *testing.T) {
	pgErr := &pgconn.PgError{Code: pgerrcode.UniqueViolation, ConstraintName: "users_login_key"}
	constraint, ok := uniqueViolation(pgErr)
	assert.True(t, ok)
	assert.Equal(t, "users_login_key", constraint)

	_, ok = uniqueViolation(errors.New("other"))
	assert.False(t, ok)

	otherPgErr := &pgconn.PgError{Code: pgerrcode.ForeignKeyViolation}
	_, ok = uniqueViolation(otherPgErr)
	assert.False(t, ok)
}

func TestIsTxRetryable(t *testing.T) {
	assert.True(t, isTxRetryable(&pgconn.PgError{Code: pgerrcode.SerializationFailure}))
	assert.True(t, isTxRetryable(&pgconn.PgError{Code: pgerrcode.DeadlockDetected}))
	assert.False(t, isTxRetryable(&pgconn.PgError{Code: pgerrcode.UniqueViolation}))
	assert.False(t, isTxRetryable(errors.New("other")))
}

func TestWithRetry_SucceedsFirstTry(t *testing.T) {
	calls := 0
	err := withRetry(context.Background(), func(error) bool { return true }, func() error {
		calls++
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestWithRetry_NonRetryableErrorReturnsImmediately(t *testing.T) {
	calls := 0
	boom := errors.New("boom")
	err := withRetry(context.Background(), func(error) bool { return false }, func() error {
		calls++
		return boom
	})
	require.ErrorIs(t, err, boom)
	assert.Equal(t, 1, calls)
}

func TestWithRetry_RetriesUpToMaxAttempts(t *testing.T) {
	calls := 0
	boom := errors.New("boom")
	err := withRetry(context.Background(), func(error) bool { return true }, func() error {
		calls++
		return boom
	})
	require.ErrorIs(t, err, boom)
	assert.Equal(t, retryMaxAttempts, calls)
}

func TestWithRetry_SucceedsAfterRetries(t *testing.T) {
	calls := 0
	err := withRetry(context.Background(), func(error) bool { return true }, func() error {
		calls++
		if calls < 2 {
			return errors.New("transient")
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 2, calls)
}

func TestWithRetry_ContextCancelledDuringBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()
	err := withRetry(ctx, func(error) bool { return true }, func() error {
		calls++
		return errors.New("transient")
	})
	require.ErrorIs(t, err, context.Canceled)
}
