package secret

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
)

// SummaryRow — унифицированная проекция Tier 2a для отображения в таблице TUI независимо от
// типа секрета (используется на вкладке «Все» и как общий вид для остальных типовых вкладок).
// Subtitle — краткая type-specific подсказка второй колонки (Username/URI для login, Issuer
// для TOTP, Bank/Brand для карты, Filename для файла).
type SummaryRow struct {
	ID         string
	Version    int64
	Type       int32
	Title      string
	Tags       []string
	Subtitle   string
	URI        string // только для login/password (Tier 2a хранит URI)
	Expiry     string // только для bank_card (из Tier 2b Index, если загружен)
	Bank       string // только для bank_card
	Cardholder string // только для bank_card
	Brand      string // только для bank_card
	Size       int64  // только для binary (из Tier 2b Index)
}

// ListAllRows возвращает Tier 2a-строки ВСЕХ типов секретов папки (вкладка «Все» в TUI),
// расшифровывая каждую строку по её реальному типу. Не делает сетевых вызовов (тот же
// локальный кеш, что и типовые List*Rows).
func (u *UseCase) ListAllRows(ctx context.Context, vaultID string) ([]SummaryRow, error) {
	return u.listRowsByType(ctx, vaultID, 0)
}

// ListRowsByType — то же, что ListAllRows, но с фильтром по конкретному domain.SecretType.
// secretType == 0 означает «без фильтра» (все типы).
func (u *UseCase) ListRowsByType(ctx context.Context, vaultID string, secretType domain.SecretType) ([]SummaryRow, error) {
	return u.listRowsByType(ctx, vaultID, int32(secretType))
}

func (u *UseCase) listRowsByType(ctx context.Context, vaultID string, filterType int32) ([]SummaryRow, error) {
	if vaultID == "" {
		return nil, ErrEmptyVaultID
	}
	vaultKey, err := u.vaultKey(vaultID)
	if err != nil {
		return nil, err
	}

	items, err := u.local.ListSecretsByVault(ctx, vaultID)
	if err != nil {
		return nil, err
	}

	result := make([]SummaryRow, 0, len(items))
	for _, it := range items {
		if filterType != 0 && it.Type != filterType {
			continue
		}
		row, err := decryptSummaryRow(u, vaultKey, vaultID, it.ID, it.Type, it.Version, it.EncRow)
		if err != nil {
			slog.Warn("secret: decrypt row failed, skipping", "secret_id", it.ID, "err", err)
			continue
		}
		enrichSummaryRow(u, vaultKey, vaultID, &row, it)
		result = append(result, row)
	}
	return result, nil
}

// enrichSummaryRow дополняет SummaryRow Tier 2b-полями (Expiry/Bank/Cardholder/Brand для карт,
// Size для файлов), если Tier 2b (enc_index) уже догружен. Общая логика для обычного списка
// (listRowsByType) и поиска (SearchSummary) — раньше поиск не вызывал этот блок, из-за чего
// найденные карточки показывали пустые эти поля, хотя обычный список их подтягивал нормально.
func enrichSummaryRow(u *UseCase, vaultKey []byte, vaultID string, row *SummaryRow, it contracts.LocalSecret) {
	if !it.IndexLoaded || len(it.EncIndex) == 0 {
		return
	}
	idxAD := secretAAD(vaultID, it.ID, it.Version, tierIndex)
	switch domain.SecretType(it.Type) {
	case domain.SecretTypeBankCard:
		var idx secretcontent.BankCardIndex
		if err := u.cipher.DecryptStruct(vaultKey, idxAD, it.EncIndex, &idx); err == nil {
			row.Expiry = idx.Expiry
			row.Bank = idx.Bank
			row.Cardholder = idx.Cardholder
			row.Brand = idx.Brand
		}
	case domain.SecretTypeBinary:
		var idx secretcontent.BinaryIndex
		if err := u.cipher.DecryptStruct(vaultKey, idxAD, it.EncIndex, &idx); err == nil {
			row.Size = idx.Size
		}
	}
}

// decryptSummaryRow расшифровывает Tier 2a под правильную типизированную структуру (по
// secretType) и приводит её к единому SummaryRow для отображения в таблице.
func decryptSummaryRow(u *UseCase, vaultKey []byte, vaultID, secretID string, secretType int32, version int64, encRow []byte) (SummaryRow, error) {
	ad := secretAAD(vaultID, secretID, version, tierRow)
	base := SummaryRow{ID: secretID, Version: version, Type: secretType}

	switch domain.SecretType(secretType) {
	case domain.SecretTypeLoginPassword:
		var row secretcontent.LoginPasswordRow
		if err := u.cipher.DecryptStruct(vaultKey, ad, encRow, &row); err != nil {
			return SummaryRow{}, err
		}
		base.Title, base.Tags, base.Subtitle = row.Title, row.Tags, row.Username
		base.URI = row.URI
	case domain.SecretTypeText:
		var row secretcontent.TextRow
		if err := u.cipher.DecryptStruct(vaultKey, ad, encRow, &row); err != nil {
			return SummaryRow{}, err
		}
		base.Title, base.Tags = row.Title, row.Tags
	case domain.SecretTypeBinary:
		var row secretcontent.BinaryRow
		if err := u.cipher.DecryptStruct(vaultKey, ad, encRow, &row); err != nil {
			return SummaryRow{}, err
		}
		base.Title, base.Tags, base.Subtitle = row.Title, row.Tags, row.Filename
	case domain.SecretTypeBankCard:
		var row secretcontent.BankCardRow
		if err := u.cipher.DecryptStruct(vaultKey, ad, encRow, &row); err != nil {
			return SummaryRow{}, err
		}
		base.Title, base.Tags = row.Title, row.Tags
		if row.Last4 != "" {
			base.Subtitle = "•• " + row.Last4
		}
	case domain.SecretTypeTOTP:
		var row secretcontent.TOTPRow
		if err := u.cipher.DecryptStruct(vaultKey, ad, encRow, &row); err != nil {
			return SummaryRow{}, err
		}
		base.Title, base.Tags, base.Subtitle = row.Title, row.Tags, row.Issuer
	default:
		return SummaryRow{}, fmt.Errorf("unknown secret type %d", secretType)
	}
	return base, nil
}
