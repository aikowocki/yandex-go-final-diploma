package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

const shutdownTimeout = 10 * time.Second

// Run запускает приложение и ждёт сигнала остановки.
// SIGINT/SIGTERM/SIGQUIT запускают graceful shutdown.
func (c *Container) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	g, groupCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		slog.Info("starting gRPC server", "addr", c.Config.GRPCAddr)
		return c.GRPC.Run(c.Config.GRPCAddr)
	})

	// pprof — отдельный HTTP-listener, поднимается только если явно задан адрес в конфиге.
	pprofSrv := newPprofServer(c.Config.PprofAddress)
	if pprofSrv != nil {
		g.Go(func() error {
			slog.Info("starting pprof server", "addr", c.Config.PprofAddress)
			if err := pprofSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Warn("pprof server stopped", "err", err)
			}
			return nil
		})
	}

	// Ждём либо ошибки запуска (groupCtx отменится раньше остановки gRPC/pprof), либо сигнала ОС
	// (тот же groupCtx унаследует отмену от ctx).
	<-groupCtx.Done()
	slog.Info("shutdown signal received")

	// Graceful shutdown с таймаутом
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	c.shutdown(shutdownCtx, pprofSrv)

	return g.Wait()
}

func (c *Container) shutdown(ctx context.Context, pprofSrv *http.Server) {
	c.GRPC.Stop()
	if pprofSrv != nil {
		_ = pprofSrv.Shutdown(ctx)
	}
	c.DB.Close()
	slog.Info("shutdown complete")
}

// newPprofServer собирает *http.Server с эндпоинтами net/http/pprof на отдельном ServeMux
func newPprofServer(addr string) *http.Server {
	if addr == "" {
		return nil
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
