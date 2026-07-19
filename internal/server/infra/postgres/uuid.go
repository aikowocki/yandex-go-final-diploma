package postgres

import "github.com/jackc/pgx/v5/pgtype"

// parseUUID парсит строковый UUID в pgtype.UUID.
// Используется всеми репозиториями пакета postgres при обращении по ID.
func parseUUID(s string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(s); err != nil {
		return pgtype.UUID{}, err
	}
	return id, nil
}

// uuidToString конвертирует pgtype.UUID в каноническую строку.
//
// Используем id.String() напрямую, а не id.Value() (database/sql/driver.Valuer):
// Value() оборачивает результат в interface{} (driver.Value), что на горячем пути
// (ListRow/ListIndex — по одному вызову на КАЖДУЮ строку списка секретов) даёт лишнюю
// heap-аллокацию на боксинг строки в интерфейс плюс никому не нужную здесь ошибку.
// String() возвращает готовую строку без промежуточного interface{} и без error.
// Найдено через pprof (profiles/base.pprof -> profiles/result.pprof, alloc_space diff).
func uuidToString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return id.String()
}
