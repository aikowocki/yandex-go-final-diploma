// cmd/loadclient — нагрузочный генератор клиентской нагрузки на реальный gRPC-сервер
// GophKeeper. Эмулирует N параллельных пользователей ("виртуальных клиентов"), каждый со своим
// изолированным in-memory Container (own account, own localstore, own session), выполняющих
// одинаковый сценарий в цикле заданное время, и печатает агрегированную статистику латентности
// и throughput по каждой операции — полезно для профилирования сервера под конкурентной
// нагрузкой (в связке с net/http/pprof, см. PPROF_ADDRESS в конфиге сервера) и для проверки,
// что оптимистичная блокировка/синхронизация не деградируют при параллельных клиентах.
//
// Запуск (сервер должен быть поднят, docker compose up):
//
//	go run ./cmd/loadclient -addr=localhost:9090 -users=20 -duration=30s
//
// Каждый виртуальный пользователь:
//  1. Register (уникальный login vu-<runID>-<n>) + SetupEncryption + Unlock
//  2. Create vault
//  3. В цикле до истечения duration: CreateLoginPassword -> UpdateLoginPassword ->
//     ListRow -> GetPayload -> Sync, с случайными паузами (think time) между итерациями.
package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/app"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/config"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
)

func main() {
	var (
		addr      = flag.String("addr", "localhost:9090", "gRPC server address")
		users     = flag.Int("users", 10, "number of concurrent virtual users")
		duration  = flag.Duration("duration", 30*time.Second, "total load duration")
		thinkTime = flag.Duration("think", 50*time.Millisecond, "pause between iterations per user")
	)
	flag.Parse()

	runID := time.Now().UnixNano()
	stats := newStatsCollector()

	fmt.Printf("GophKeeper load client: addr=%s users=%d duration=%s think=%s\n", *addr, *users, *duration, *thinkTime)

	ctx, cancel := context.WithTimeout(context.Background(), *duration+30*time.Second)
	defer cancel()

	deadline := time.Now().Add(*duration)

	var wg sync.WaitGroup
	for i := 0; i < *users; i++ {
		wg.Add(1)
		go func(userIdx int) {
			defer wg.Done()
			runVirtualUser(ctx, virtualUserParams{
				addr:      *addr,
				login:     fmt.Sprintf("vu-%d-%d", runID, userIdx),
				deadline:  deadline,
				thinkTime: *thinkTime,
				stats:     stats,
			})
		}(i)
	}
	wg.Wait()

	stats.printReport(*duration)
}

type virtualUserParams struct {
	addr      string
	login     string
	deadline  time.Time
	thinkTime time.Duration
	stats     *statsCollector
}

// runVirtualUser выполняет полный сценарий одного пользователя: регистрация/настройка один
// раз, затем цикл CRUD-операций до deadline. Ошибки соединения/сервера не останавливают
// остальных виртуальных пользователей — записываются в stats и логируются, горутина
// завершается досрочно только при фатальной ошибке на этапе настройки (register/unlock).
func runVirtualUser(ctx context.Context, p virtualUserParams) {
	dataDir, err := os.MkdirTemp("", "gk-loadclient-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[%s] tmp dir: %v\n", p.login, err)
		return
	}
	defer func() { _ = os.RemoveAll(dataDir) }()

	cfg := &config.ClientConfig{
		ServerAddr: p.addr,
		DataDir:    dataDir,
		NoPersist:  true, // in-memory localstore — изоляция между виртуальными пользователями
		LogLevel:   "error",
	}

	container, err := app.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[%s] app.New: %v\n", p.login, err)
		return
	}
	defer func() { _ = container.Close() }()

	const masterPassphrase = "load-test-master-pass"
	const loginCredential = "load-test-credential"

	if !p.stats.record("register", func() error {
		return container.Auth.Register(ctx, p.login, []byte(loginCredential))
	}) {
		return
	}
	if !p.stats.record("setup_encryption", func() error {
		return container.Auth.SetupEncryption(ctx, []byte(masterPassphrase))
	}) {
		return
	}

	var vaultID string
	if !p.stats.record("create_vault", func() error {
		id, cerr := container.Vault.Create(ctx, "load-vault")
		vaultID = id
		return cerr
	}) {
		return
	}

	var secretIDs []string
	for time.Now().Before(p.deadline) {
		secretID := ""
		p.stats.record("create_secret", func() error {
			id, cerr := container.Secret.CreateLoginPassword(ctx, vaultID, secret.CreateLoginPasswordInput{
				Title:    fmt.Sprintf("secret-%d", len(secretIDs)+1),
				Username: "load-user",
				Password: "load-password",
				URI:      "https://example.com",
			})
			secretID = id
			return cerr
		})
		if secretID != "" {
			secretIDs = append(secretIDs, secretID)
		}

		if len(secretIDs) > 0 {
			target := secretIDs[rand.Intn(len(secretIDs))]
			p.stats.record("update_secret", func() error {
				ver, ok, verr := container.Secret.LocalVersion(ctx, target)
				if verr != nil {
					return verr
				}
				if !ok {
					return nil // секрет мог ещё не осесть локально — пропускаем итерацию без ошибки
				}
				_, uerr := container.Secret.UpdateLoginPassword(ctx, vaultID, target, ver, secret.CreateLoginPasswordInput{
					Title:    "updated-title",
					Username: "load-user",
					Password: "updated-password",
					URI:      "https://example.com",
				})
				return uerr
			})

			p.stats.record("get_payload", func() error {
				_, perr := container.Secret.GetPayload(ctx, vaultID, target)
				return perr
			})
		}

		p.stats.record("list_row", func() error {
			_, lerr := container.Secret.ListRow(ctx, vaultID)
			return lerr
		})

		p.stats.record("sync", func() error {
			return container.Sync.Sync(ctx)
		})

		select {
		case <-ctx.Done():
			return
		case <-time.After(p.thinkTime):
		}
	}
}

// --- статистика ---

type opStats struct {
	mu        sync.Mutex
	durations []time.Duration
	errors    int64
	sampleErr string // текст первой встреченной ошибки — для диагностики в отчёте
}

type statsCollector struct {
	mu    sync.Mutex
	ops   map[string]*opStats
	total atomic.Int64
}

func newStatsCollector() *statsCollector {
	return &statsCollector{ops: make(map[string]*opStats)}
}

// record выполняет fn, замеряет длительность и записывает результат под именем op.
// Возвращает true, если fn не вернула ошибку (удобно для цепочек "остановиться при ошибке
// на этапе настройки").
func (s *statsCollector) record(op string, fn func() error) bool {
	start := time.Now()
	err := fn()
	elapsed := time.Since(start)

	s.mu.Lock()
	st, ok := s.ops[op]
	if !ok {
		st = &opStats{}
		s.ops[op] = st
	}
	s.mu.Unlock()

	st.mu.Lock()
	st.durations = append(st.durations, elapsed)
	if err != nil {
		st.errors++
		if st.sampleErr == "" {
			st.sampleErr = err.Error()
		}
	}
	st.mu.Unlock()

	s.total.Add(1)
	return err == nil
}

// printReport печатает по каждой операции: количество вызовов, ошибки, throughput (op/s) и
// перцентили латентности (p50/p90/p99/max).
func (s *statsCollector) printReport(duration time.Duration) {
	s.mu.Lock()
	names := make([]string, 0, len(s.ops))
	for name := range s.ops {
		names = append(names, name)
	}
	s.mu.Unlock()
	sort.Strings(names)

	fmt.Println()
	fmt.Printf("%-16s %8s %8s %10s %10s %10s %10s %10s\n", "OPERATION", "COUNT", "ERRORS", "RPS", "P50", "P90", "P99", "MAX")
	sampleErrs := make(map[string]string)
	for _, name := range names {
		st := s.ops[name]
		st.mu.Lock()
		durs := append([]time.Duration(nil), st.durations...)
		errs := st.errors
		sampleErr := st.sampleErr
		st.mu.Unlock()

		sort.Slice(durs, func(i, j int) bool { return durs[i] < durs[j] })
		count := len(durs)
		rps := float64(count) / duration.Seconds()

		fmt.Printf("%-16s %8d %8d %10.1f %10s %10s %10s %10s\n",
			name, count, errs, rps,
			percentile(durs, 0.50), percentile(durs, 0.90), percentile(durs, 0.99), maxDuration(durs))
		if sampleErr != "" {
			sampleErrs[name] = sampleErr
		}
	}
	fmt.Printf("\nTotal operations: %d\n", s.total.Load())

	if len(sampleErrs) > 0 {
		fmt.Println("\nSample errors (first occurrence per operation):")
		for _, name := range names {
			if e, ok := sampleErrs[name]; ok {
				fmt.Printf("  %-16s %s\n", name, e)
			}
		}
	}
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}

func maxDuration(sorted []time.Duration) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	return sorted[len(sorted)-1]
}
