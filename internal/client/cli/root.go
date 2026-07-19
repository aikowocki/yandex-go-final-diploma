package cli

import "github.com/aikowocki/yandex-go-final-diploma/internal/client/config"

// CLI описывает корневую структуру команд клиента gophkeeper (парсится kong).
type CLI struct {
	Config config.ClientConfig `embed:""`

	Register        RegisterCmd        `cmd:"" help:"Register a new account and set up encryption."`
	Login           LoginCmd           `cmd:"" help:"Log in and unlock the local session."`
	SetupEncryption SetupEncryptionCmd `cmd:"" name:"setup-encryption" help:"Set up encryption for the current account."`
	Recover         RecoverCmd         `cmd:"" help:"Recover the master key using a recovery code (forgot master password)."`
	Vault           VaultCmd           `cmd:"" help:"Manage vaults."`
	Secret          SecretCmd          `cmd:"" help:"Manage secrets."`
	Sync            SyncCmd            `cmd:"" help:"Sync local cache with the server and flush the offline outbox."`
	Outbox          OutboxCmd          `cmd:"" help:"Inspect the offline change queue (outbox)."`
	Logs            LogsCmd            `cmd:"" help:"Show client logs (from <data-dir>/client.log)."`
	Ping            PingCmd            `cmd:"" help:"Check server connectivity."`
	Version         VersionCmd         `cmd:"" help:"Print client version."`
}
