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
// Ошибки Value() намеренно игнорируются: pgtype.UUID, прочитанный из БД,
// всегда валиден, поэтому Value() для него не может завершиться ошибкой.
func uuidToString(id pgtype.UUID) string {
	s, _ := id.Value()
	str, _ := s.(string)
	return str
}
