package app

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"
	"time"
)

const shutdownTimeout = 10 * time.Second

// Run запускает приложение и ждёт сигнала остановки.
func (c *Container) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Запускаем gRPC в отдельной горутине
	errCh := make(chan error, 1)
	go func() {
		slog.Info("starting gRPC server", "addr", c.Config.GRPCAddr)
		errCh <- c.GRPC.Run(c.Config.GRPCAddr)
	}()

	// Ждём либо ошибки запуска, либо сигнала
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	}

	// Graceful shutdown с таймаутом
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	c.shutdown(shutdownCtx)
	return nil
}

func (c *Container) shutdown(ctx context.Context) {
	c.GRPC.Stop()
	c.DB.Close()
	slog.Info("shutdown complete")
}
