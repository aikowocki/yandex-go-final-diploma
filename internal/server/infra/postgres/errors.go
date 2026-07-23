package postgres

import (
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// isNoRows проверяет, что запрос не вернул строк.
func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

// uniqueViolation возвращает имя нарушенного ограничения и true если нарушение unique constraint.
func uniqueViolation(err error) (constraint string, ok bool) {
	if pgErr, ok2 := errors.AsType[*pgconn.PgError](err); ok2 && pgErr.Code == pgerrcode.UniqueViolation {
		return pgErr.ConstraintName, true
	}
	return "", false
}

// isConnRetryable сообщает, безопасно ли повторить запрос из-за проблемы
// соединения.
func isConnRetryable(err error) bool {
	return pgconn.SafeToRetry(err)
}

// isTxRetryable сообщает, стоит ли повторить транзакцию целиком.
func isTxRetryable(err error) bool {
	if isConnRetryable(err) {
		return true
	}
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		switch pgErr.Code {
		case pgerrcode.SerializationFailure, pgerrcode.DeadlockDetected:
			return true
		}
	}
	return false
}
