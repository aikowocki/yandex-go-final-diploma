package auth_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth/mocks"
)

// Бенчмарки Register/Login — намеренно самое дорогое место в usecase-слое сервера: оба пути
// хэшируют пароль через Argon2id (memory-hard KDF, специально требовательный к CPU/памяти,
// чтобы затруднить offline brute-force). Обнаружено через cmd/loadclient: при конкурентной
// нагрузке (30+ параллельных виртуальных пользователей) именно register/login первыми
// показывают деградацию latency (секунды вместо десятков миллисекунд) — Argon2id-параметры
// (argon2id.DefaultParams) конкурируют за CPU между горутинами. Бенчмарки ниже дают точную
// цифру "сколько стоит один Register/Login" изолированно от сети/БД, без искажения от
// параллельной нагрузки на реальный процесс.
//
// Запуск:
//
//	go test ./internal/server/usecase/auth/... -bench=Register -benchmem -run=^$
//	go test ./internal/server/usecase/auth/... -bench=Login -benchmem -run=^$ -cpuprofile=cpu.out
//	go tool pprof cpu.out
//
// -benchtime, -cpu N позволяют сравнить однопоточный и параллельный (RunParallel) throughput,
// чтобы увидеть, во сколько раз конкурентный Argon2id медленнее одного вызова * N.

func BenchmarkRegister(b *testing.B) {
	users := mocks.NewMockRepository(b)
	users.EXPECT().Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, u domain.User) (domain.User, error) {
			return domain.User{ID: "user-1", Login: u.Login, AuthHash: u.AuthHash}, nil
		}).Maybe()

	tokens := mocks.NewMockTokenIssuer(b)
	tokens.EXPECT().Issue(mock.Anything).Return("access", "refresh", nil).Maybe()

	uc := auth.New(users, nil, tokens, benchTx{})
	ctx := context.Background()

	b.ReportAllocs()
	for b.Loop() {
		if _, err := uc.Register(ctx, auth.RegisterParams{
			Login:           "bench-user",
			LoginCredential: []byte("bench-password-123"),
		}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRegister_Parallel показывает, как масштабируется Argon2id-хэширование при
// конкурентных Register (аналог нескольких пользователей, регистрирующихся одновременно —
// ровно сценарий, который cmd/loadclient воспроизводит по сети). GOMAXPROCS горутин конкурируют
// за CPU внутри одного процесса Argon2id, поэтому throughput здесь растёт НЕ линейно с числом
// горутин.
func BenchmarkRegister_Parallel(b *testing.B) {
	users := mocks.NewMockRepository(b)
	users.EXPECT().Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, u domain.User) (domain.User, error) {
			return domain.User{ID: "user-1", Login: u.Login, AuthHash: u.AuthHash}, nil
		}).Maybe()

	tokens := mocks.NewMockTokenIssuer(b)
	tokens.EXPECT().Issue(mock.Anything).Return("access", "refresh", nil).Maybe()

	uc := auth.New(users, nil, tokens, benchTx{})
	ctx := context.Background()

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := uc.Register(ctx, auth.RegisterParams{
				Login:           "bench-user",
				LoginCredential: []byte("bench-password-123"),
			}); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkLogin_Success измеряет argon2id.ComparePasswordAndHash — тот же порядок стоимости,
// что и CreateHash в Register, вызывается на КАЖДЫЙ Login (в т.ч. Refresh-флоу на клиенте при
// каждом новом запуске CLI-процесса.
func BenchmarkLogin_Success(b *testing.B) {
	// Хэш считаем один раз до цикла — измеряем именно стоимость ПРОВЕРКИ пароля, не создания.
	precomputedHash := precomputeHashForBench(b, "bench-password-123")

	users := mocks.NewMockRepository(b)
	users.EXPECT().GetByLogin(mock.Anything, "bench-user").
		Return(domain.User{ID: "user-1", Login: "bench-user", AuthHash: precomputedHash}, nil).Maybe()

	tokens := mocks.NewMockTokenIssuer(b)
	tokens.EXPECT().Issue("user-1").Return("access", "refresh", nil).Maybe()

	uc := auth.New(users, nil, tokens, benchTx{})
	ctx := context.Background()

	b.ReportAllocs()
	for b.Loop() {
		if _, err := uc.Login(ctx, auth.LoginParams{
			Login:           "bench-user",
			LoginCredential: []byte("bench-password-123"),
		}); err != nil {
			b.Fatal(err)
		}
	}
}

// precomputeHashForBench получает валидный Argon2id-хэш через один настоящий Register —
// не создаём отдельную зависимость на internal-детали пакета auth (хэш непрозрачен снаружи).
func precomputeHashForBench(b *testing.B, password string) string {
	b.Helper()
	var captured string
	users := mocks.NewMockRepository(b)
	users.EXPECT().Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, u domain.User) (domain.User, error) {
			captured = u.AuthHash
			return domain.User{ID: "user-1", Login: u.Login, AuthHash: u.AuthHash}, nil
		})
	tokens := mocks.NewMockTokenIssuer(b)
	tokens.EXPECT().Issue(mock.Anything).Return("a", "r", nil)

	uc := auth.New(users, nil, tokens, benchTx{})
	if _, err := uc.Register(context.Background(), auth.RegisterParams{Login: "bench-user", LoginCredential: []byte(password)}); err != nil {
		b.Fatal(err)
	}
	return captured
}

// benchTx — TxManager, выполняющий fn немедленно, без мок-фреймворка (в горячем цикле
// бенчмарка избегаем накладных расходов testify.mock на матчинг аргументов).
type benchTx struct{}

func (benchTx) Do(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }
