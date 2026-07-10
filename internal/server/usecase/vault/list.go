package vault

import "context"

// ListVaults возвращает список папок пользователя (для разворачивания VaultKey и показа имени).
func (u *UseCase) ListVaults(ctx context.Context, userID string) ([]Tier1, error) {
	if userID == "" {
		return nil, ErrEmptyUserID
	}

	vaults, err := u.vaults.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]Tier1, 0, len(vaults))
	for _, v := range vaults {
		result = append(result, Tier1{
			ID:              v.ID,
			WrappedVaultKey: v.WrappedVaultKey,
			EncName:         v.EncName,
			Version:         v.Version,
		})
	}
	return result, nil
}
