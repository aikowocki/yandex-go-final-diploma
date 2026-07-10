package cli

import (
	"context"
	"fmt"
	"strings"

	clienti18n "github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
	secretuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
	vaultuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/vault"
)

// SecretCmd — группа команд секретов.
type SecretCmd struct {
	Add  SecretAddCmd  `cmd:"" help:"Add a login/password secret to a vault."`
	List SecretListCmd `cmd:"" help:"List secrets of a vault (without revealing passwords)."`
	Get  SecretGetCmd  `cmd:"" help:"Reveal a secret's password."`
}

type SecretAddCmd struct {
	Vault string `arg:"" help:"Vault name."`
}

func (c *SecretAddCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, secret *secretuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}
	vaultID, err := openVaultByName(ctx, vault, c.Vault)
	if err != nil {
		return err
	}

	title, err := promptLine(l.T("prompt_title"))
	if err != nil {
		return err
	}
	username, err := promptLine(l.T("prompt_username"))
	if err != nil {
		return err
	}
	password, err := promptSecret(l.T("prompt_password"))
	if err != nil {
		return err
	}
	uri, err := promptLine(l.T("prompt_uri"))
	if err != nil {
		return err
	}
	tagsRaw, err := promptLine(l.T("prompt_tags"))
	if err != nil {
		return err
	}
	note, err := promptLine(l.T("prompt_note"))
	if err != nil {
		return err
	}

	if _, err := secret.CreateLoginPassword(ctx, vaultID, secretuc.CreateLoginPasswordInput{
		Title:    title,
		Username: username,
		Password: string(password),
		URI:      uri,
		Tags:     parseTags(tagsRaw),
		Note:     note,
	}); err != nil {
		return err
	}
	fmt.Println(l.T("secret_created"))
	return nil
}

type SecretListCmd struct {
	Vault string `arg:"" help:"Vault name."`
}

func (c *SecretListCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, secret *secretuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}
	vaultID, err := openVaultByName(ctx, vault, c.Vault)
	if err != nil {
		return err
	}

	rows, err := secret.ListRow(ctx, vaultID)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		fmt.Println(l.T("secret_empty"))
		return nil
	}
	for _, r := range rows {
		// Пароль здесь НЕ показывается (Tier 2a). id нужен для secret get.
		fmt.Printf("%s\t%s\t%s\t%s\n", r.ID, r.Row.Title, r.Row.Username, r.Row.URI)
	}
	return nil
}

type SecretGetCmd struct {
	Vault string `arg:"" help:"Vault name."`
	ID    string `arg:"" help:"Secret id (from 'secret list')."`
}

func (c *SecretGetCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, secret *secretuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}
	vaultID, err := openVaultByName(ctx, vault, c.Vault)
	if err != nil {
		return err
	}

	payload, err := secret.GetPayload(ctx, vaultID, c.ID)
	if err != nil {
		return err
	}
	fmt.Println(payload.Payload.Password)
	return nil
}

// parseTags разбивает строку тегов по запятой, обрезает пробелы и отбрасывает пустые.
func parseTags(raw string) []string {
	parts := strings.Split(raw, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			tags = append(tags, t)
		}
	}
	if len(tags) == 0 {
		return nil
	}
	return tags
}
