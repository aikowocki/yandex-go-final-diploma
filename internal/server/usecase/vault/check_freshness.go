package vault

import "context"

// CheckFreshness возвращает версии всех (не удалённых) папок пользователя одним запросом.
// Клиент сравнивает их со своими synced_version и решает, что подтягивать (Tier 2a/2b/3).
func (u *UseCase) CheckFreshness(ctx context.Context, userID string) ([]Version, error) {
	if userID == "" {
		return nil, ErrEmptyUserID
	}
	return u.vaults.CheckFreshness(ctx, userID)
}
