package cli

import "github.com/aikowocki/yandex-go-final-diploma/internal/client/config"

type CLI struct {
	Config config.ClientConfig `embed:""`

	Register        RegisterCmd        `cmd:"" help:"Register a new account and set up encryption."`
	Login           LoginCmd           `cmd:"" help:"Log in and unlock the local session."`
	SetupEncryption SetupEncryptionCmd `cmd:"" name:"setup-encryption" help:"Set up encryption for the current account."`
	Vault           VaultCmd           `cmd:"" help:"Manage vaults."`
	Secret          SecretCmd          `cmd:"" help:"Manage secrets."`
	Ping            PingCmd            `cmd:"" help:"Check server connectivity."`
	Version         VersionCmd         `cmd:"" help:"Print client version."`
}
