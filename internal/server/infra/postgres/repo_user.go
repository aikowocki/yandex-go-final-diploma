package postgres

import (
	"context"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/infra/postgres/gen"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth"
)

// constraintUsersLogin — имя unique-ограничения на users.login
const constraintUsersLogin = "users_login_key"

// UserRepo реализует auth.Repository поверх sqlc-сгенерированных запросов.
// Транслирует ошибки pgx/pgconn в доменные sentinel-ошибки usecase-слоя — наружу
type UserRepo struct {
	db *DB
}

// NewUserRepo создаёт UserRepo поверх переданного пула соединений.
func NewUserRepo(db *DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) q(ctx context.Context) *gen.Queries {
	return gen.New(r.db.querier(ctx))
}

// Create создаёт нового пользователя, транслируя конфликт логина в auth.ErrLoginTaken.
func (r *UserRepo) Create(ctx context.Context, u domain.User) (domain.User, error) {
	row, err := r.q(ctx).CreateUser(ctx, gen.CreateUserParams{
		Login:    u.Login,
		AuthHash: u.AuthHash,
	})
	if err != nil {
		if constraint, ok := uniqueViolation(err); ok && constraint == constraintUsersLogin {
			return domain.User{}, auth.ErrLoginTaken
		}
		return domain.User{}, fmt.Errorf("create user: %w", err)
	}
	return mapUser(row), nil
}

// GetByLogin находит пользователя по логину.
func (r *UserRepo) GetByLogin(ctx context.Context, login string) (domain.User, error) {
	row, err := r.q(ctx).GetUserByLogin(ctx, login)
	if err != nil {
		if isNoRows(err) {
			return domain.User{}, auth.ErrUserNotFound
		}
		return domain.User{}, fmt.Errorf("get user by login: %w", err)
	}
	return mapUser(row), nil
}

// GetByID находит пользователя по идентификатору.
func (r *UserRepo) GetByID(ctx context.Context, id string) (domain.User, error) {
	pgID, err := parseUUID(id)
	if err != nil {
		return domain.User{}, auth.ErrUserNotFound
	}

	row, err := r.q(ctx).GetUserByID(ctx, pgID)
	if err != nil {
		if isNoRows(err) {
			return domain.User{}, auth.ErrUserNotFound
		}
		return domain.User{}, fmt.Errorf("get user by id: %w", err)
	}
	return mapUser(row), nil
}

// UpdateEncKDF обновляет параметры KDF и зашифрованный master key пользователя.
func (r *UserRepo) UpdateEncKDF(ctx context.Context, userID string, salt, params, encMasterKey []byte) error {
	pgID, err := parseUUID(userID)
	if err != nil {
		return auth.ErrUserNotFound
	}

	if err := r.q(ctx).UpdateUserEncKDF(ctx, gen.UpdateUserEncKDFParams{
		ID:           pgID,
		EncKdfSalt:   salt,
		EncKdfParams: params,
		EncMasterKey: encMasterKey,
	}); err != nil {
		return fmt.Errorf("update user enc kdf: %w", err)
	}
	return nil
}

// mapUser конвертирует sqlc-строку users в доменную модель domain.User.
func mapUser(row gen.User) domain.User {
	return domain.User{
		ID:           uuidToString(row.ID),
		Login:        row.Login,
		AuthHash:     row.AuthHash,
		EncKDFSalt:   row.EncKdfSalt,
		EncKDFParams: row.EncKdfParams,
		EncMasterKey: row.EncMasterKey,
		TOTPSecret:   row.TotpSecret,
		TOTPEnabled:  row.TotpEnabled,
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}
}
