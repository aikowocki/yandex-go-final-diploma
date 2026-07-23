package secret

import "context"

// DeleteSecret выполняет soft-delete секрета с оптимистичной блокировкой по версии.
// Удаление — это тоже update: если кто-то успел изменить секрет (version != base_version),
// возвращается *ErrConflict, а не тихий успех. Уже удалённый секрет → ErrSecretNotFound.
func (u *UseCase) DeleteSecret(ctx context.Context, params DeleteParams) (int64, error) {
	if params.UserID == "" {
		return 0, ErrEmptyUserID
	}
	if params.SecretID == "" {
		return 0, ErrEmptySecretID
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

		v, err := u.secrets.SoftDelete(ctx, params.SecretID)
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
