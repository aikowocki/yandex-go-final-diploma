package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/infra/postgres/gen"
)

// txKey — приватный ключ для хранения активной транзакции в context.
type txKey struct{}

// TxManager объединяет несколько операций репозиториев в одну транзакцию БД.
// usecase запускает атомарный блок через Do, не зная про pgx/sqlc — он лишь
// прокидывает полученный ctx в методы репозиториев. Репозитории через querier
// автоматически подхватывают транзакцию из ctx (или работают на пуле, если
// транзакции нет).
type TxManager struct {
	db *DB
}

// NewTxManager создаёт TxManager поверх переданного пула соединений.
func NewTxManager(db *DB) *TxManager {
	return &TxManager{db: db}
}

// Do выполняет fn внутри одной транзакции: коммитит при успехе, откатывает при
// ошибке или панике.
func (m *TxManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := txFromContext(ctx); ok {
		return fn(ctx)
	}

	return withRetry(ctx, isTxRetryable, func() (err error) {
		tx, err := m.db.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}
		defer func() {
			// Rollback безопасен после Commit (вернёт ErrTxClosed)
			// ошибку игнорируем осознанно.
			_ = tx.Rollback(ctx)
		}()

		if err := fn(withTx(ctx, tx)); err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit tx: %w", err)
		}
		return nil
	})
}

func withTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

func txFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	return tx, ok
}

// querier возвращает исполнителя sqlc-запросов для текущего контекста:
// активную транзакцию, если она есть, иначе — пул с ретраями одиночных
// запросов. Репозитории строят gen.Queries поверх результата на каждый вызов.
func (db *DB) querier(ctx context.Context) gen.DBTX {
	if tx, ok := txFromContext(ctx); ok {
		return tx
	}
	return retryingDB{pool: db.Pool}
}
