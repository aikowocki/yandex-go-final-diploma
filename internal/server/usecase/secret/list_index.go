package secret

import "context"

// ListIndex возвращает Tier 2b-проекцию (индексные блобы для поиска).
func (u *UseCase) ListIndex(ctx context.Context, userID, vaultID string) ([]IndexItem, error) {
	if userID == "" {
		return nil, ErrEmptyUserID
	}
	if vaultID == "" {
		return nil, ErrEmptyVaultID
	}

	secrets, err := u.secrets.ListIndex(ctx, vaultID, userID)
	if err != nil {
		return nil, err
	}

	result := make([]IndexItem, 0, len(secrets))
	for _, s := range secrets {
		result = append(result, IndexItem{
			ID:       s.ID,
			Version:  s.Version,
			EncIndex: s.EncIndex,
		})
	}
	return result, nil
}
