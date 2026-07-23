package secret_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret/mocks"
)

// Бенчмарки usecase-слоя секретов — без сети/БД (мок-репозиторий), чтобы измерять только
// накладные расходы самого usecase (валидация, сборка domain-объектов, работа с TxManager),
// изолированно от Postgres. Запуск:
//
//	go test ./internal/server/usecase/secret/... -bench=. -benchmem -run=^$
//	go test ./internal/server/usecase/secret/... -bench=BenchmarkUpdateSecret_Conflict -cpuprofile=cpu.out -run=^$
//	go tool pprof cpu.out

// BenchmarkCreateSecret_Success измеряет happy-path создания секрета: проверка владения
// папкой + запись + bump версии папки (все шаги — no-op на моках, значит бенчмарк отражает
// чистые накладные расходы usecase, не инфраструктуры).
func BenchmarkCreateSecret_Success(b *testing.B) {
	vaults := mocks.NewMockVaultOwnership(b)
	vaults.EXPECT().IsOwner(mock.Anything, "vault-1", "user-1").Return(true, nil)

	secrets := mocks.NewMockRepository(b)
	secrets.EXPECT().Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, s domain.Secret) (domain.Secret, error) { return s, nil })
	secrets.EXPECT().BumpVaultVersion(mock.Anything, "vault-1").Return(nil)

	uc := secret.New(secrets, vaults, benchTx{})
	ctx := context.Background()
	params := validBenchCreateParams()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := uc.CreateSecret(ctx, params); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUpdateSecret_Success измеряет happy-path обновления с оптимистичной блокировкой
// (GetForUpdate + UpdateFields + BumpVaultVersion) — самый частый путь при активной
// синхронизации нескольких устройств одного пользователя.
func BenchmarkUpdateSecret_Success(b *testing.B) {
	secrets := mocks.NewMockRepository(b)
	secrets.EXPECT().GetForUpdate(mock.Anything, "secret-1", "user-1").
		Return(domain.Secret{ID: "secret-1", VaultID: "vault-1", Version: 3}, nil)
	secrets.EXPECT().UpdateFields(mock.Anything, "secret-1", []byte("r"), []byte("i"), []byte("p")).
		Return(int64(4), nil)
	secrets.EXPECT().BumpVaultVersion(mock.Anything, "vault-1").Return(nil)

	uc := secret.New(secrets, mocks.NewMockVaultOwnership(b), benchTx{})
	ctx := context.Background()
	params := secret.UpdateParams{
		UserID: "user-1", SecretID: "secret-1", BaseVersion: 3,
		EncRow: []byte("r"), EncIndex: []byte("i"), EncPayload: []byte("p"),
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := uc.UpdateSecret(ctx, params); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUpdateSecret_Conflict измеряет путь гонки версий : GetForUpdate обнаруживает
// version mismatch, UpdateFields не вызывается, строится *ErrConflict с полной серверной версией.
func BenchmarkUpdateSecret_Conflict(b *testing.B) {
	secrets := mocks.NewMockRepository(b)
	secrets.EXPECT().GetForUpdate(mock.Anything, "secret-1", "user-1").
		Return(domain.Secret{
			ID: "secret-1", VaultID: "vault-1", Version: 9,
			EncRow: []byte("server-row"), EncIndex: []byte("server-index"), EncPayload: []byte("server-payload"),
		}, nil)

	uc := secret.New(secrets, mocks.NewMockVaultOwnership(b), benchTx{})
	ctx := context.Background()
	params := secret.UpdateParams{
		UserID: "user-1", SecretID: "secret-1", BaseVersion: 3,
		EncRow: []byte("r"), EncIndex: []byte("i"), EncPayload: []byte("p"),
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := uc.UpdateSecret(ctx, params); err == nil {
			b.Fatal("expected conflict error")
		}
	}
}

// BenchmarkListRow_ManySecrets измеряет отдачу Tier 2a-списка при разном размере папки —
// показывает, как масштабируется маппинг domain.Secret -> secret.Row в зависимости от
// количества секретов (актуально для крупных vault'ов при полном pull на новом устройстве).
func BenchmarkListRow_ManySecrets(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		b.Run(benchName(n), func(b *testing.B) {
			rows := make([]domain.Secret, n)
			for i := range rows {
				rows[i] = domain.Secret{ID: benchID(i), Type: domain.SecretTypeLoginPassword, Version: 1, EncRow: []byte("row-data")}
			}

			secrets := mocks.NewMockRepository(b)
			secrets.EXPECT().ListRow(mock.Anything, "vault-1", "user-1").Return(rows, nil)

			uc := secret.New(secrets, mocks.NewMockVaultOwnership(b), benchTx{})
			ctx := context.Background()

			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := uc.ListRow(ctx, "user-1", "vault-1"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func validBenchCreateParams() secret.CreateParams {
	return secret.CreateParams{
		UserID:     "user-1",
		VaultID:    "vault-1",
		SecretID:   "secret-1",
		Type:       domain.SecretTypeLoginPassword,
		EncRow:     []byte("enc-row"),
		EncIndex:   []byte("enc-index"),
		EncPayload: []byte("enc-payload"),
	}
}

// benchTx — TxManager, выполняющий fn немедленно, без мок-фреймворка (избегаем накладных
// расходов mockery/testify.mock на матчинг аргументов внутри горячего цикла бенчмарка).
type benchTx struct{}

func (benchTx) Do(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

func benchName(n int) string {
	switch n {
	case 10:
		return "10_secrets"
	case 100:
		return "100_secrets"
	case 1000:
		return "1000_secrets"
	default:
		return "n_secrets"
	}
}

func benchID(i int) string {
	const alphabet = "0123456789abcdef"
	b := []byte("secret-00000000")
	n := i
	for pos := len(b) - 1; pos >= len(b)-8 && n > 0; pos-- {
		b[pos] = alphabet[n%16]
		n /= 16
	}
	return string(b)
}
