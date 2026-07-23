package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
)

// ClientConfig — конфигурация клиента gophkeeper, парсится из флагов/env/config.json (kong).
type ClientConfig struct {
	ServerAddr      string `help:"gRPC server address" short:"s" default:"localhost:9090" env:"SERVER_ADDR"`
	DataDir         string `help:"local data directory" short:"d" default:"${default_data_dir}" env:"DATA_DIR"`
	NoPersist       bool   `help:"disable local persistence" short:"n" default:"false" env:"NO_PERSIST"`
	LogLevel        string `help:"log level" default:"info" env:"LOG_LEVEL"`
	Lang            string `help:"UI language (en, ru)" default:"en" env:"GOPHKEEPER_LANG"`
	AutolockMinutes int    `help:"auto-lock timeout in minutes (0 = never)" default:"5" env:"AUTOLOCK_MINUTES"`
	TOTPRevealMode  string `help:"TOTP reveal mode: focused|all" default:"focused" env:"TOTP_REVEAL_MODE"`
	SyncDelayMs     int    `help:"debug: artificial sync delay per vault (ms)" default:"0" env:"SYNC_DELAY_MS"`
	PprofAddress    string `help:"pprof HTTP listen address for TUI mode (empty = disabled)" env:"PPROF_ADDRESS"`
	// Version — версия сборки (заполняется из main, не сохраняется в config.json).
	Version string `kong:"-" json:"-"`
}

// persistedConfig — подмножество ClientConfig, сохраняемое в config.json.
type persistedConfig struct {
	ServerAddr      string `json:"server_addr"`
	DataDir         string `json:"data_dir"`
	NoPersist       bool   `json:"no_persist"`
	LogLevel        string `json:"log_level"`
	Lang            string `json:"lang"`
	AutolockMinutes int    `json:"autolock_minutes"`
	TOTPRevealMode  string `json:"totp_reveal_mode"`
}

// Save сохраняет конфиг клиента в config.json (в defaultDataDir).
func Save(cfg *ClientConfig) error {
	dir, err := defaultDataDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	// Определяем, какой data_dir писать в config.json: если текущий DataDir совпадает с
	// defaultDataDir — пишем его (onboarding wizard записал осознанный выбор пользователя).
	// Если отличается (задан через env/flag) — сохраняем дефолт, чтобы не перезатирать
	// config.json эфемерной настройкой текущего запуска.
	persistDataDir := cfg.DataDir
	if cfg.DataDir != dir {
		// Попытка прочитать предыдущее значение из существующего config.json.
		if existing, err := os.ReadFile(filepath.Join(dir, "config.json")); err == nil {
			var prev persistedConfig
			if json.Unmarshal(existing, &prev) == nil && prev.DataDir != "" {
				persistDataDir = prev.DataDir
			}
		} else {
			persistDataDir = dir
		}
	}

	pc := persistedConfig{
		ServerAddr:      cfg.ServerAddr,
		DataDir:         persistDataDir,
		NoPersist:       cfg.NoPersist,
		LogLevel:        cfg.LogLevel,
		Lang:            cfg.Lang,
		AutolockMinutes: cfg.AutolockMinutes,
		TOTPRevealMode:  cfg.TOTPRevealMode,
	}
	raw, err := json.MarshalIndent(pc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "config.json"), raw, 0o600)
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
// config.json всегда ищется в глобальной директории приложения (defaultDataDir).
func parseClientConfig(args []string, defaultDataDir string) (*ClientConfig, error) {
	configPath := filepath.Join(defaultDataDir, "config.json")

	var cfg ClientConfig
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

	if !hasDataDirFlag(args) {
		if envDataDir, ok := os.LookupEnv("DATA_DIR"); ok && envDataDir != "" {
			cfg.DataDir = envDataDir
		}
	}
	return &cfg, nil
}

// hasDataDirFlag сообщает, передан ли --data-dir/-d явно среди аргументов — в этом случае
// флаг должен выигрывать у DATA_DIR env (обычный приоритет flag > env).
func hasDataDirFlag(args []string) bool {
	for _, a := range args {
		if a == "--data-dir" || a == "-d" || strings.HasPrefix(a, "--data-dir=") {
			return true
		}
	}
	return false
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
