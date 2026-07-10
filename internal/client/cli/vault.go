package cli

import (
	"context"
	"errors"
	"fmt"

	clienti18n "github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
	vaultuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/vault"
)

// errVaultNotFound / errVaultAmbiguous — резолв папок по имени в CLI.
var (
	errVaultNotFound  = errors.New("vault not found by name")
	errVaultAmbiguous = errors.New("multiple vaults with this name, use a unique name")
)

// VaultCmd — группа команд папок.
type VaultCmd struct {
	Create VaultCreateCmd `cmd:"" help:"Create a new vault."`
	List   VaultListCmd   `cmd:"" help:"List vaults."`
}

type VaultCreateCmd struct {
	Name string `arg:"" help:"Vault name."`
}

func (c *VaultCreateCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}

	if _, err := vault.Create(ctx, c.Name); err != nil {
		return err
	}
	fmt.Println(l.T("vault_created"))
	return nil
}

type VaultListCmd struct{}

func (c *VaultListCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}

	vaults, err := vault.List(ctx)
	if err != nil {
		return err
	}
	if len(vaults) == 0 {
		fmt.Println(l.T("vault_empty"))
		return nil
	}
	for _, v := range vaults {
		fmt.Printf("%s\t(v%d)\n", v.Name, v.Version)
	}
	return nil
}

// openVaultByName открывает папку (unwrap в сессию через List) и резолвит id по имени.
func openVaultByName(ctx context.Context, vault *vaultuc.UseCase, name string) (string, error) {
	vaults, err := vault.List(ctx)
	if err != nil {
		return "", err
	}

	var id string
	matches := 0
	for _, v := range vaults {
		if v.Name == name {
			id = v.ID
			matches++
		}
	}
	switch {
	case matches == 0:
		return "", errVaultNotFound
	case matches > 1:
		return "", errVaultAmbiguous
	default:
		return id, nil
	}
}
