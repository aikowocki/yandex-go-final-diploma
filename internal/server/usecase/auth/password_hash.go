package auth

import (
	"context"
	"fmt"
	"runtime"

	"github.com/alexedwards/argon2id"
)

// passwordHashParams — фиксированные параметры Argon2id для хеширования login credential.
//
// НЕ используем argon2id.DefaultParams: там Parallelism = runtime.NumCPU(), т.е. КАЖДЫЙ
// отдельный вызов CreateHash/ComparePasswordAndHash пытается занять все ядра машины сразу.
// Под конкурентной нагрузкой (N параллельных Register/Login) это оборачивается тем, что N
// вызовов одновременно конкурируют за одни и те же ядра, а также совокупно требуют N * Memory
// памяти (64 МиБ на вызов по дефолту) — именно это было обнаружено профилированием под
// нагрузкой: argon2.processBlockGeneric и обнуление буфера съедали ~94% CPU-времени сервера
// при 30 параллельных виртуальных пользователях, а latency Register улетала на секунды.
//
// Parallelism=1 убирает гонку за несколько ядер (сам вызов чуть медленнее
// в однопоточном режиме, но зато не конкурирует сам с собой), Memory снижена до 19 МиБ —
// рекомендация OWASP для t=2. Совокупно с семафором ниже это держит суммарное потребление памяти под
// контролем независимо от числа одновременных запросов.
var passwordHashParams = &argon2id.Params{
	Memory:      19 * 1024, // 19 MiB
	Iterations:  2,
	Parallelism: 1,
	SaltLength:  16,
	KeyLength:   32,
}

// maxConcurrentPasswordHashes ограничает число одновременно выполняющихся Argon2id-операций
// (CreateHash/ComparePasswordAndHash) в рамках одного процесса сервера. Без этого лимита сервер
// пытался бы прогнать все конкурентные Register/Login параллельно, даже если реальных ядер
// меньше, чем запросов — то есть просто ставил бы горутины в очередь на уровне ОС-планировщика
// вместо явной, предсказуемой очереди на уровне приложения. GOMAXPROCS — разумный дефолт:
// больше физического параллелизма всё равно не даст выигрыша, т.к. Argon2id — CPU-bound.
var passwordHashSem = make(chan struct{}, max(2, runtime.GOMAXPROCS(0)))

// hashPassword — обёртка над argon2id.CreateHash с фиксированными параметрами и ограничением
// конкурентности через семафор. Уважает ctx.Done(), чтобы клиент, отменивший запрос (таймаут gRPC),
// не заставлял сервер впустую ждать в очереди.
func hashPassword(ctx context.Context, password string) (string, error) {
	if err := acquireHashSlot(ctx); err != nil {
		return "", err
	}
	defer releaseHashSlot()

	return argon2id.CreateHash(password, passwordHashParams)
}

// comparePassword — обёртка над argon2id.ComparePasswordAndHash с тем же лимитом
// конкурентности. Параметры для сравнения берутся из самого хеша —
// смена passwordHashParams влияет только на НОВЫЕ хеши, старые продолжают проверяться со
// своими исходными параметрами (миграция не требуется).
func comparePassword(ctx context.Context, password, hash string) (bool, error) {
	if err := acquireHashSlot(ctx); err != nil {
		return false, err
	}
	defer releaseHashSlot()

	return argon2id.ComparePasswordAndHash(password, hash)
}

func acquireHashSlot(ctx context.Context) error {
	select {
	case passwordHashSem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for password hash slot: %w", ctx.Err())
	}
}

func releaseHashSlot() {
	<-passwordHashSem
}
