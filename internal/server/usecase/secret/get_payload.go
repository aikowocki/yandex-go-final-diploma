package secret

import "context"

// GetPayload возвращает Tier 3 (чувствительное тело) одного секрета.
func (u *UseCase) GetPayload(ctx context.Context, userID, secretID string) (Payload, error) {
	if userID == "" {
		return Payload{}, ErrEmptyUserID
	}
	if secretID == "" {
		return Payload{}, ErrEmptySecretID
	}

	s, err := u.secrets.GetPayload(ctx, secretID, userID)
	if err != nil {
		return Payload{}, err
	}

	return Payload{
		ID:         s.ID,
		Type:       s.Type,
		Version:    s.Version,
		EncPayload: s.EncPayload,
	}, nil
}
