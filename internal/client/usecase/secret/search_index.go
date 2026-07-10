package secret

import (
	"context"
	"strings"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
)

// SearchResult — результат поиска по секретам папки.
type SearchResult struct {
	Rows []DecryptedRow
	// Incomplete=true, если Tier 2b (note/custom_fields) ещё не догружен для части секретов:
	// поиск по этим полям пока не полон (используется для индикатора «поиск неполный»).
	Incomplete bool
}

// Search ищет секреты папки в локальном кеше.
func (u *UseCase) Search(ctx context.Context, vaultID, query string) (SearchResult, error) {
	if vaultID == "" {
		return SearchResult{}, ErrEmptyVaultID
	}
	vaultKey, err := u.vaultKey(vaultID)
	if err != nil {
		return SearchResult{}, err
	}

	items, err := u.local.ListSecretsByVault(ctx, vaultID)
	if err != nil {
		return SearchResult{}, err
	}

	q := strings.ToLower(strings.TrimSpace(query))
	var res SearchResult

	for _, it := range items {
		if !it.IndexLoaded {
			res.Incomplete = true
		}

		rowMap, row, err := u.decryptRowGeneric(vaultKey, vaultID, it.ID, it.Version, it.EncRow)
		if err != nil {
			return SearchResult{}, err
		}

		match := q == "" || matchesQuery(rowMap, q)
		if !match && it.IndexLoaded {
			idxMap, err := u.decryptIndexGeneric(vaultKey, vaultID, it.ID, it.Version, it.EncIndex)
			if err != nil {
				return SearchResult{}, err
			}
			match = matchesQuery(idxMap, q)
		}

		if match {
			res.Rows = append(res.Rows, DecryptedRow{ID: it.ID, Version: it.Version, Row: row})
		}
	}
	return res, nil
}

// decryptRowGeneric расшифровывает Tier 2a в map[string]any.
func (u *UseCase) decryptRowGeneric(vaultKey []byte, vaultID, secretID string, version int64, encRow []byte) (map[string]any, secretcontent.LoginPasswordRow, error) {
	var m map[string]any
	ad := secretAAD(vaultID, secretID, version, tierRow)
	if err := u.cipher.DecryptStruct(vaultKey, ad, encRow, &m); err != nil {
		return nil, secretcontent.LoginPasswordRow{}, err
	}
	var row secretcontent.LoginPasswordRow
	if err := remarshal(m, &row); err != nil {
		return nil, secretcontent.LoginPasswordRow{}, err
	}
	return m, row, nil
}

// decryptIndexGeneric расшифровывает Tier 2b в map[string]any для типонезависимого поиска.
func (u *UseCase) decryptIndexGeneric(vaultKey []byte, vaultID, secretID string, version int64, encIndex []byte) (map[string]any, error) {
	if len(encIndex) == 0 {
		return nil, nil
	}
	var m map[string]any
	ad := secretAAD(vaultID, secretID, version, tierIndex)
	if err := u.cipher.DecryptStruct(vaultKey, ad, encIndex, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// matchesQuery ищет подстроку q (уже в нижнем регистре) среди ЛЮБЫХ строковых значений
// произвольно вложенной JSON-структуры (map/slice/string) — работает одинаково для всех типов
// секрета без type-specific кода: tags/custom_fields — массивы, остальные поля — top-level строки.
func matchesQuery(v any, qLower string) bool {
	switch val := v.(type) {
	case string:
		return containsFold(val, qLower)
	case map[string]any:
		for _, vv := range val {
			if matchesQuery(vv, qLower) {
				return true
			}
		}
	case []any:
		for _, vv := range val {
			if matchesQuery(vv, qLower) {
				return true
			}
		}
	}
	return false
}

func containsFold(s, qLower string) bool {
	return strings.Contains(strings.ToLower(s), qLower)
}
