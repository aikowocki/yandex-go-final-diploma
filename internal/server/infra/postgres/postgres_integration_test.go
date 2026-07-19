package postgres_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	pgmodule "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/infra/postgres"
)

// Контейнер Postgres и подключение к нему поднимаются один раз на весь пакет тестов
// (а не на каждый тест), потому что старт контейнера — самая дорогая часть теста
// (секунды на docker run + миграции). Изоляция между тестами обеспечивается не
// отдельными БД, а TRUNCATE всех таблиц перед каждым вызовом newTestDB.
var (
	sharedContainer *pgmodule.PostgresContainer
	sharedDB        *postgres.DB
	sharedInitOnce  sync.Once
	sharedInitErr   error
)

// tablesToTruncate — все таблицы схемы, в порядке, безопасном для TRUNCATE ... CASCADE.
var tablesToTruncate = []string{"recovery_codes", "secrets", "vaults", "users"}

func TestMain(m *testing.M) {
	code := m.Run()

	if sharedContainer != nil {
		_ = sharedContainer.Terminate(context.Background())
	}
	if sharedDB != nil {
		sharedDB.Close()
	}

	os.Exit(code)
}

// newTestDB возвращает подключение к общему для всех тестов пакета Postgres-контейнеру,
// поднятому лениво при первом обращении. Перед возвратом все таблицы очищаются, поэтому
// тесты остаются изолированными друг от друга (в том числе при повторяющихся логинах типа
// "alice") несмотря на общий контейнер и пул соединений.
func newTestDB(t *testing.T) *postgres.DB {
	t.Helper()

	sharedInitOnce.Do(func() {
		sharedContainer, sharedDB, sharedInitErr = startSharedPostgres()
	})
	require.NoError(t, sharedInitErr, "не удалось поднять общий тестовый Postgres")

	truncateAllTables(t, sharedDB)

	return sharedDB
}

func startSharedPostgres() (*pgmodule.PostgresContainer, *postgres.DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	const dbName, user, pass = "gophkeeper_test", "gophkeeper", "gophkeeper"
	container, err := pgmodule.Run(ctx, "postgres:16-alpine",
		pgmodule.WithDatabase(dbName),
		pgmodule.WithUsername(user),
		pgmodule.WithPassword(pass),
		pgmodule.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("start postgres container: %w", err)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return terminateAndReturn(container, fmt.Errorf("get connection string: %w", err))
	}

	if err := postgres.RunMigrations(dsn); err != nil {
		return terminateAndReturn(container, fmt.Errorf("run migrations: %w", err))
	}

	db, err := postgres.NewPool(context.Background(), dsn)
	if err != nil {
		return terminateAndReturn(container, fmt.Errorf("connect pool: %w", err))
	}

	return container, db, nil
}

func terminateAndReturn(container *pgmodule.PostgresContainer, err error) (*pgmodule.PostgresContainer, *postgres.DB, error) {
	_ = container.Terminate(context.Background())
	return nil, nil, err
}

// truncateAllTables очищает все таблицы перед каждым тестом, использующим общий
// контейнер, чтобы данные одного теста не просачивались в другой.
func truncateAllTables(t *testing.T, db *postgres.DB) {
	t.Helper()

	query := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", joinTables(tablesToTruncate))
	_, err := db.Exec(context.Background(), query)
	require.NoError(t, err, "не удалось очистить таблицы перед тестом")
}

func joinTables(tables []string) string {
	out := ""
	for i, tbl := range tables {
		if i > 0 {
			out += ", "
		}
		out += tbl
	}
	return out
}
