package secret

import "context"

// UpdateSecret применяет новые шифротексты секрета с оптимистичной блокировкой по версии.
// В одной транзакции: читает текущую версию под блокировкой (FOR UPDATE); если она совпадает
// с base_version — обновляет (version+1) и бампит версию папки; иначе не трогает данные и
// возвращает *ErrConflict с актуальной серверной версией.
func (u *UseCase) UpdateSecret(ctx context.Context, params UpdateParams) (int64, error) {
	if params.UserID == "" {
		return 0, ErrEmptyUserID
	}
	if params.SecretID == "" {
		return 0, ErrEmptySecretID
	}
	if len(params.EncRow) == 0 {
		return 0, ErrEmptyEncRow
	}
	if len(params.EncIndex) == 0 {
		return 0, ErrEmptyEncIndex
	}

	var newVersion int64
	err := u.tx.Do(ctx, func(ctx context.Context) error {
		current, err := u.secrets.GetForUpdate(ctx, params.SecretID, params.UserID)
		if err != nil {
			return err
		}
		if current.Deleted {
			return ErrSecretNotFound
		}
		if current.Version != params.BaseVersion {
			return &ErrConflict{Current: current}
		}

		v, err := u.secrets.UpdateFields(ctx, params.SecretID, params.EncRow, params.EncIndex, params.EncPayload)
		if err != nil {
			return err
		}
		if err := u.secrets.BumpVaultVersion(ctx, current.VaultID); err != nil {
			return err
		}
		newVersion = v
		return nil
	})
	if err != nil {
		return 0, err
	}
	return newVersion, nil
}
