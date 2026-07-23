package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/infra/postgres/gen"
)

// baseRepo инкапсулирует общую для всех репозиториев инфраструктуру: доступ
// к пулу соединений и построение querier, привязанного к текущей tx/conn.
type baseRepo struct {
	db *DB
}

// q возвращает sqlc-Queries, использующий активную транзакцию из ctx (если
// она есть) или пул соединений с ретраями иначе.
func (r baseRepo) q(ctx context.Context) *gen.Queries {
	return gen.New(r.db.querier(ctx))
}

// parseUUIDOr парсит строковый id в pgtype.UUID, транслируя невалидный формат
// в переданный sentinel notFound — так репозитории не раскрывают вызывающей
// стороне детали парсинга UUID.
func parseUUIDOr(id string, notFound error) (pgtype.UUID, error) {
	pgID, err := parseUUID(id)
	if err != nil {
		return pgtype.UUID{}, notFound
	}
	return pgID, nil
}

// wrapNotFound транслирует pgx.ErrNoRows в переданный sentinel notFound,
// остальные ошибки оборачивает с указанным контекстом операции.
func wrapNotFound(err error, notFound error, opCtx string) error {
	if isNoRows(err) {
		return notFound
	}
	return fmt.Errorf("%s: %w", opCtx, err)
}
