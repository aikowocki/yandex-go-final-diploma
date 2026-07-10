package secret

import "context"

// ListRow возвращает Tier 2a-проекцию секретов ваулта (строки списка, без payload).
func (u *UseCase) ListRow(ctx context.Context, userID, vaultID string) ([]Row, error) {
	if userID == "" {
		return nil, ErrEmptyUserID
	}
	if vaultID == "" {
		return nil, ErrEmptyVaultID
	}

	secrets, err := u.secrets.ListRow(ctx, vaultID, userID)
	if err != nil {
		return nil, err
	}

	result := make([]Row, 0, len(secrets))
	for _, s := range secrets {
		result = append(result, Row{
			ID:      s.ID,
			Type:    s.Type,
			Version: s.Version,
			EncRow:  s.EncRow,
		})
	}
	return result, nil
}
