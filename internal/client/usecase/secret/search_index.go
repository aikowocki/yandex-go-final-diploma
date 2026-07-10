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

// Search ищет секреты папки в локальном кеше. Всегда ищет по Tier 2a (title/username/uri/tags);
// по Tier 2b (note/custom_fields) — только для секретов, у которых индекс уже догружен (LoadIndexes).
// Пустой запрос возвращает все строки. Если хотя бы у одного секрета Tier 2b не загружен,
// результат помечается Incomplete.
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

		var row secretcontent.LoginPasswordRow
		ad := secretAAD(vaultID, it.ID, it.Version, tierRow)
		if err := u.cipher.DecryptStruct(vaultKey, ad, it.EncRow, &row); err != nil {
			return SearchResult{}, err
		}

		match := q == "" || matchRow(row, q)
		if !match && it.IndexLoaded {
			idx, err := u.decryptIndex(vaultKey, vaultID, localSecretView{ID: it.ID, Version: it.Version, EncIndex: it.EncIndex})
			if err != nil {
				return SearchResult{}, err
			}
			match = matchIndex(idx, q)
		}

		if match {
			res.Rows = append(res.Rows, DecryptedRow{ID: it.ID, Version: it.Version, Row: row})
		}
	}
	return res, nil
}

// matchRow ищет подстроку q (в нижнем регистре) по Tier 2a-полям.
func matchRow(row secretcontent.LoginPasswordRow, q string) bool {
	if containsFold(row.Title, q) || containsFold(row.Username, q) || containsFold(row.URI, q) {
		return true
	}
	for _, t := range row.Tags {
		if containsFold(t, q) {
			return true
		}
	}
	return false
}

// matchIndex ищет подстроку q по Tier 2b-полям (note/custom_fields).
func matchIndex(idx secretcontent.LoginPasswordIndex, q string) bool {
	if containsFold(idx.Note, q) {
		return true
	}
	for _, kv := range idx.CustomFields {
		if containsFold(kv.Key, q) || containsFold(kv.Value, q) {
			return true
		}
	}
	return false
}

func containsFold(s, qLower string) bool {
	return strings.Contains(strings.ToLower(s), qLower)
}
