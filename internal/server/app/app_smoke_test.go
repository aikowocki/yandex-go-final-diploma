package app_test

import (
	"context"
	"net"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	pgmodule "github.com/testcontainers/testcontainers-go/modules/postgres"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/aikowocki/yandex-go-final-diploma/api/proto/gen/gophkeeper/v1"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/app"
)

// TestApp_SmokeBootAndRegister — смок-тест: поднимает реальный Postgres (testcontainers),
// собирает весь Container через app.New (config → DB+migrations → objectstore(nil) → grpc),
// запускает Run() и делает один живой RPC-вызов через сгенерированный клиент. Не проверяет
// бизнес-логику деталей (это делают юнит/интеграционные тесты слоёв) — только то, что
// весь стек поднимается и отвечает на реальный запрос по сети, без падений на любом из
// этапов сборки/старта/остановки процесса.
func TestApp_SmokeBootAndRegister(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	const dbName, dbUser, dbPass = "gophkeeper_smoke", "gophkeeper", "gophkeeper"
	pgContainer, err := pgmodule.Run(ctx, "postgres:16-alpine",
		pgmodule.WithDatabase(dbName),
		pgmodule.WithUsername(dbUser),
		pgmodule.WithPassword(dbPass),
		pgmodule.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pgContainer.Terminate(context.Background()) })

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	addr := "127.0.0.1:0"
	lis, err := net.Listen("tcp", addr)
	require.NoError(t, err)
	grpcAddr := lis.Addr().String()
	require.NoError(t, lis.Close()) // отдаём порт приложению, само слушать не будем

	// app.New/config читает флаги/env через kong, а os.Args в тестовом бинарнике содержит
	// флаги go test (-test.run и т.п.), которые kong не распознаёт. Подменяем os.Args на
	// пустой набор (только имя программы) и передаём всё через env, как в реальном запуске.
	t.Setenv("DATABASE_DSN", dsn)
	t.Setenv("JWT_SECRET", "smoke-test-secret")
	t.Setenv("GRPC_ADDR", grpcAddr)
	t.Setenv("LOG_LEVEL", "error")
	// MINIO_ENDPOINT не задан — objectstore.New не вызывается (сервер без blob-хранилища),
	// это штатный режим согласно providers.NewObjectStore.
	origArgs := os.Args
	os.Args = []string{origArgs[0]}
	t.Cleanup(func() { os.Args = origArgs })

	container, err := app.New(ctx)
	require.NoError(t, err, "весь DI-стек (config, db+migrations, objectstore, grpc) должен собраться без ошибок")

	runErrCh := make(chan error, 1)
	go func() { runErrCh <- container.Run() }()

	// Ждём, пока gRPC-сервер реально начнёт слушать порт.
	require.Eventually(t, func() bool {
		conn, dialErr := net.DialTimeout("tcp", grpcAddr, 200*time.Millisecond)
		if dialErr != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, 10*time.Second, 100*time.Millisecond, "gRPC-сервер должен начать слушать адрес")

	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	client := pb.NewAuthServiceClient(conn)

	pingRes, err := client.Ping(ctx, &pb.PingRequest{})
	require.NoError(t, err)
	require.Equal(t, "pong", pingRes.GetMessage())

	regRes, err := client.Register(ctx, &pb.RegisterRequest{
		Login:           "smoke-user",
		LoginCredential: []byte("smoke-password"),
	})
	require.NoError(t, err, "полный happy-path Register должен пройти через весь стек: grpc -> usecase -> postgres")
	require.NotEmpty(t, regRes.GetAccessToken())
	require.NotEmpty(t, regRes.GetUserId())

	// т.к. тест выполняется в том же процессе, что и container.Run() (внутри), можно послать
	// сигнал самому себе через syscall.Kill.
	require.NoError(t, syscall.Kill(os.Getpid(), syscall.SIGQUIT))

	select {
	case runErr := <-runErrCh:
		require.NoError(t, runErr, "Run() должен завершиться без ошибки после graceful shutdown по SIGQUIT")
	case <-time.After(15 * time.Second):
		t.Fatal("Run() не завершился в течение 15s после SIGQUIT — graceful shutdown завис")
	}
}
