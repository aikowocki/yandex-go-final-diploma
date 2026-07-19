package tui

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/app"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	secretuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

// seedLoginSecretsBench создаёт n login/password секретов в vaultID через настоящий
// UseCase.CreateLoginPassword (реальное шифрование + запись в SQLite), с мок-сервером,
// который на CreateSecret отвечает успехом без сети. Возвращает container для дальнейшего
// использования в бенчмарке.
func seedLoginSecretsBench(b *testing.B, n int) (*app.Container, string) {
	b.Helper()

	server := mocks.NewMockServerClient(b)
	server.EXPECT().CreateSecret(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Maybe()

	c := newTestContainer(b, server)
	ctx := context.Background()

	vaultID := "vault-bench"
	vaultKey, err := crypto.GenerateKey()
	require.NoError(b, err)
	c.Session.OpenVault(vaultID, vaultKey)

	for i := 0; i < n; i++ {
		_, err := c.Secret.CreateLoginPassword(ctx, vaultID, secretuc.CreateLoginPasswordInput{
			Title:    fmt.Sprintf("Secret %d", i),
			Username: fmt.Sprintf("user%d@example.com", i),
			Password: "correct-horse-battery-staple",
			URI:      fmt.Sprintf("https://example-%d.com", i),
			Tags:     []string{"bench", fmt.Sprintf("group%d", i%10)},
		})
		require.NoError(b, err)
	}
	return c, vaultID
}

// BenchmarkDashboardTable_LoadAndFilter измеряет полный цикл «открыть вкладку» (reload —
// расшифровка ВСЕХ Tier 2a-строк папки) + применение локального фильтра (applyLocalFilter,
// тот самый путь, где раньше был баг с пропадающими Tier 2b-полями при поиске) — на разном
// количестве секретов. Показывает, где TUI начинает "тормозить" при росте папок.
func BenchmarkDashboardTable_LoadAndFilter(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		b.Run(fmt.Sprintf("%d_secrets", n), func(b *testing.B) {
			c, vaultID := seedLoginSecretsBench(b, n)
			ctx := context.Background()

			table := newDashboardTableModel(ctx, c)
			table.vaultID = vaultID

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cmd := table.reload()
				msg := cmd()
				loaded, ok := msg.(rowsLoadedMsg)
				require.True(b, ok)
				table.allRows = loaded.rows
				table = table.applyLocalFilter()

				table.searchTerm = "group3"
				table = table.applyLocalFilter()
			}
		})
	}
}

// BenchmarkDashboardTable_View измеряет рендер таблицы (view()) — построение строк, padding,
// zone.Mark для каждой ячейки — на разном количестве строк В ОКНЕ ПРОКРУТКИ (maxRows не растёт
// с общим количеством секретов, т.к. scrollWindow всегда рисует constant-size окно; бенчмарк
// подтверждает, что рендер НЕ деградирует с ростом папки, только объём данных в allRows/rows).
func BenchmarkDashboardTable_View(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		b.Run(fmt.Sprintf("%d_secrets", n), func(b *testing.B) {
			c, vaultID := seedLoginSecretsBench(b, n)
			ctx := context.Background()

			table := newDashboardTableModel(ctx, c)
			table.vaultID = vaultID
			cmd := table.reload()
			msg := cmd()
			loaded := msg.(rowsLoadedMsg)
			table.allRows = loaded.rows
			table = table.applyLocalFilter()
			table.loading = false

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = table.view(120, 20)
			}
		})
	}
}
