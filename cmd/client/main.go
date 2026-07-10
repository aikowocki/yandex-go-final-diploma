package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/alecthomas/kong"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/app"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/cli"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/config"
)

var (
	version   = "dev"
	buildDate = "unknown"
)

func main() {
	dataDir, err := config.DefaultDataDir()
	if err != nil {
		fatal(err.Error())
	}

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
	defer container.Close()

	kctx.Bind(
		container.Auth,
		container.Vault,
		container.Secret,
		container.Sync,
		container.Local,
		container.GRPC,
		container.Localizer,
		&cli.BuildInfo{Version: version, Date: formatBuildDate(buildDate)},
	)

	if err := kctx.Run(); err != nil {
		fatal(cli.RenderError(container.Localizer, err))
	}
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
