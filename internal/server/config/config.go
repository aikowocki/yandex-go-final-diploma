package config

import (
	"os"

	"github.com/alecthomas/kong"
)

type ServerConfig struct {
	GRPCAddr      string          `help:"gRPC listen address" short:"a" default:":9090" env:"GRPC_ADDR"`
	DatabaseDSN   string          `help:"PostgreSQL DSN" short:"d" env:"DATABASE_DSN"`
	JWTSecret     string          `help:"JWT signing secret" short:"j" env:"JWT_SECRET"`
	LogLevel      string          `help:"log level" short:"l" default:"info" env:"LOG_LEVEL"`
	MinioEndpoint string          `help:"MinIO endpoint" env:"MINIO_ENDPOINT"`
	MinioAccess   string          `help:"MinIO access key" env:"MINIO_ACCESS_KEY"`
	MinioSecret   string          `help:"MinIO secret key" env:"MINIO_SECRET_KEY"`
	MinioBucket   string          `help:"MinIO bucket for binary secret blobs" default:"gophkeeper-blobs" env:"MINIO_BUCKET"`
	MinioUseSSL   bool            `help:"Use TLS when connecting to MinIO" env:"MINIO_USE_SSL"`
	ConfigFile    kong.ConfigFlag `help:"path to JSON config file" short:"c" env:"CONFIG" placeholder:"PATH"`
}

// LoadServerConfig загружает конфиг из os.Args.
func LoadServerConfig() (*ServerConfig, error) {
	return parseServerConfig(os.Args[1:])
}

// parseServerConfig парсит конфиг из переданных аргументов.
func parseServerConfig(args []string) (*ServerConfig, error) {
	var cfg ServerConfig
	parser, err := kong.New(&cfg,
		kong.Name("gophkeeper-server"),
		kong.Description("GophKeeper server"),
		kong.Configuration(kong.JSON, "/etc/gophkeeper/config.json"),
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
