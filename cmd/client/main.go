package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/app"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/cli"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/config"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/tui"
)

// shutdownFlushTimeout — сколько ждём завершения outbox-flush (ReplayOutbox) после получения
// сигнала остановки, прежде чем выйти.
const shutdownFlushTimeout = 5 * time.Second

var (
	version   = "dev"
	buildDate = "unknown"
)

func main() {
	// Если аргументы отсутствуют (или единственный аргумент — «tui») — запускаем TUI.
	if shouldLaunchTUI(os.Args[1:]) {
		runTUI()
		return
	}

	dataDir, err := config.DefaultDataDir()
	if err != nil {
		fatal(err.Error())
	}

	// SIGINT/SIGTERM/SIGQUIT по умолчанию (без явного signal.Notify) убивают процесс НЕМЕДЛЕННО
	// (default disposition ОС), что может прервать сетевой вызов на середине. CLI-команды —
	// одноразовые и короткие; регистрация здесь просто отключает default-kill, позволяя текущей
	// команде (единственная активная операция в one-shot CLI) штатно завершить свой RPC и выйти
	// через обычный `return` из kctx.Run(), а не быть убитой посреди отправки данных на сервер.
	// Из канала не читаем — сигнал не отменяет операцию, только не даёт runtime мгновенно убить
	// процесс до того, как текущая команда допишет данные и дойдёт до defer container.Close().
	signal.Ignore(syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	var root cli.CLI
	kctx := kong.Parse(&root,
		kong.Name("gophkeeper-client"),
		kong.Description("GophKeeper client"),
		kong.Vars{"default_data_dir": dataDir},
		kong.Configuration(kong.JSON, filepath.Join(dataDir, "config.json")),
		kong.UsageOnError(),
	)

	container, err := app.New(&root.Config)
	if err != nil {
		fatal(err.Error())
	}
	defer func() { _ = container.Close() }()

	kctx.Bind(
		container.Auth,
		container.Vault,
		container.Secret,
		container.Sync,
		container.Local,
		container.GRPC,
		container.Localizer,
		container.Config,
		&cli.BuildInfo{Version: version, Date: formatBuildDate(buildDate)},
	)

	if err := kctx.Run(); err != nil {
		fatal(cli.RenderError(container.Localizer, err))
	}
}

// shouldLaunchTUI определяет, нужно ли запускать TUI вместо CLI.
func shouldLaunchTUI(args []string) bool {
	if len(args) == 0 {
		return true
	}
	if len(args) == 1 && args[0] == "tui" {
		return true
	}
	return false
}

// runTUI запускает интерактивный Bubble Tea TUI. Конфиг тот же, что у CLI.
func runTUI() {
	// Убираем "tui" из os.Args, чтобы config.LoadClientConfig (парсит os.Args[1:]) не споткнулся.
	cleanArgs := make([]string, 0, len(os.Args))
	cleanArgs = append(cleanArgs, os.Args[0])
	for _, a := range os.Args[1:] {
		if a != "tui" {
			cleanArgs = append(cleanArgs, a)
		}
	}
	os.Args = cleanArgs

	// Первый запуск: если config.json ещё нет — показываем onboarding wizard (server addr,
	// data dir, persist).
	dataDir, err := config.DefaultDataDir()
	if err != nil {
		fatal(err.Error())
	}
	if !fileExists(filepath.Join(dataDir, "config.json")) {
		if oerr := tui.RunOnboarding("localhost:9090", dataDir); oerr != nil {
			fatal(oerr.Error())
		}
	}

	cfg, err := config.LoadClientConfig()
	if err != nil {
		fatal(err.Error())
	}
	cfg.Version = version

	container, err := app.New(cfg)
	if err != nil {
		fatal(err.Error())
	}
	defer func() { _ = container.Close() }()

	// pprof — отдельный HTTP-listener, поднимается только если явно задан адрес в конфиге.
	if pprofSrv := newPprofServer(cfg.PprofAddress); pprofSrv != nil {
		go func() {
			slog.Info("starting pprof server", "addr", cfg.PprofAddress)
			if err := pprofSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Warn("pprof server stopped", "err", err)
			}
		}()
	}

	// Пробуем загрузить кешированные параметры KDF для офлайн-разблокировки.
	_ = container.Auth.LoadCachedEncryption(context.Background())

	startScreen := tui.StartScreen(container)
	model := tui.New(context.Background(), container, startScreen)

	zone.NewGlobal()
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())

	// SIGINT/SIGTERM/SIGQUIT: TUI — long-running процесс (в отличие от one-shot CLI), поэтому
	// здесь нужен настоящий graceful shutdown.
	sigCtx, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stopSignals()
	go func() {
		<-sigCtx.Done()
		slog.Info("shutdown signal received, flushing outbox")
		p.Quit()
	}()

	_, runErr := p.Run()

	flushCtx, cancelFlush := context.WithTimeout(context.Background(), shutdownFlushTimeout)
	if err := container.Sync.ReplayOutbox(flushCtx); err != nil {
		slog.Warn("shutdown: replay outbox failed", "err", err)
	}
	cancelFlush()

	if runErr != nil {
		fatal(runErr.Error())
	}
}

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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func fatal(msg string) {
	_, _ = fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

func formatBuildDate(raw string) string {
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	return t.Format("2006-01-02")
}
