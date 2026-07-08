package config

import (
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"
)

type ClientConfig struct {
	ServerAddr string `help:"gRPC server address" short:"s" default:"localhost:9090" env:"SERVER_ADDR"`
	DataDir    string `help:"local data directory" short:"d" default:"${default_data_dir}" env:"DATA_DIR"`
	NoPersist  bool   `help:"disable local persistence" short:"n" default:"false" env:"NO_PERSIST"`
	LogLevel   string `help:"log level" default:"info" env:"LOG_LEVEL"`
	Lang       string `help:"UI language (en, ru)" default:"en" env:"GOPHKEEPER_LANG"`
}

// LoadClientConfig загружает конфиг из os.Args.
func LoadClientConfig() (*ClientConfig, error) {
	dataDir, err := defaultDataDir()
	if err != nil {
		return nil, err
	}
	return parseClientConfig(os.Args[1:], dataDir)
}

// parseClientConfig парсит конфиг из переданных аргументов.
func parseClientConfig(args []string, defaultDataDir string) (*ClientConfig, error) {
	var cfg ClientConfig
	configPath := filepath.Join(defaultDataDir, "config.json")

	parser, err := kong.New(&cfg,
		kong.Name("gophkeeper-client"),
		kong.Description("GophKeeper client"),
		kong.Vars{"default_data_dir": defaultDataDir},
		kong.Configuration(kong.JSON, configPath),
		kong.Exit(func(int) {}),
	)
	if err != nil {
		return nil, err
	}
	_, err = parser.Parse(args)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// DefaultDataDir возвращает каталог данных клиента по умолчанию (<UserConfigDir>/gophkeeper).
func DefaultDataDir() (string, error) {
	return defaultDataDir()
}

func defaultDataDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "gophkeeper"), nil
}
