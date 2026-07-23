package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	clienti18n "github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
	secretuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
	vaultuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/vault"
)

// TOTPCmd — группа команд для секретов типа totp (authenticator-записи, E2E).
type TOTPCmd struct {
	Add    TOTPAddCmd    `cmd:"" help:"Add a TOTP secret (manual fields or paste an otpauth:// URI/QR text)."`
	List   TOTPListCmd   `cmd:"" help:"List TOTP secrets of a vault."`
	Code   TOTPCodeCmd   `cmd:"" help:"Generate the current TOTP code for a secret."`
	Update TOTPUpdateCmd `cmd:"" help:"Update a TOTP secret (optimistic locking; resolves conflicts)."`
}

// TOTPAddCmd — добавление секрета типа totp в папку.
type TOTPAddCmd struct {
	Vault string `arg:"" help:"Vault name."`
}

// Run запрашивает поля TOTP-секрета (вручную или из otpauth:// URI) и создаёт его в папке.
func (c *TOTPAddCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, secret *secretuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}
	vaultID, err := openVaultByName(ctx, vault, c.Vault)
	if err != nil {
		return err
	}

	input, err := promptTOTPInput(l)
	if err != nil {
		return err
	}
	if _, err := secret.CreateTOTP(ctx, vaultID, input); err != nil {
		return err
	}
	fmt.Println(l.T("secret_created"))
	return nil
}

// TOTPListCmd — список секретов типа totp в папке.
type TOTPListCmd struct {
	Vault string `arg:"" help:"Vault name."`
}

// Run выводит список секретов типа totp в указанной папке.
func (c *TOTPListCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, secret *secretuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}
	vaultID, err := openVaultByName(ctx, vault, c.Vault)
	if err != nil {
		return err
	}

	rows, err := secret.ListTOTPRows(ctx, vaultID)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		fmt.Println(l.T("secret_empty"))
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTITLE\tISSUER")
	for _, r := range rows {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", r.ID, r.Row.Title, r.Row.Issuer)
	}
	_ = w.Flush()
	return nil
}

// TOTPCodeCmd генерирует текущий код — секрет расшифровывается только для запрошенной строки
// (минимизация экспозиции, тот же принцип, что и в плане §6.5 для TUI).
type TOTPCodeCmd struct {
	Vault string `arg:"" help:"Vault name."`
	ID    string `arg:"" help:"Secret id (from 'secret totp list')."`
}

// Run расшифровывает секрет TOTP и печатает текущий одноразовый код.
func (c *TOTPCodeCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, secret *secretuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}
	vaultID, err := openVaultByName(ctx, vault, c.Vault)
	if err != nil {
		return err
	}

	d, err := secret.GetTOTPDetail(ctx, vaultID, c.ID)
	if err != nil {
		return err
	}
	code, err := secretuc.GenerateTOTPCode(d.Payload)
	if err != nil {
		return err
	}
	fmt.Println(code)
	return nil
}

// TOTPUpdateCmd — редактирование секрета totp с оптимистичной блокировкой по версии.
type TOTPUpdateCmd struct {
	Vault string `arg:"" help:"Vault name."`
	ID    string `arg:"" help:"Secret id (from 'secret totp list')."`
}

// Run обновляет секрет totp, разрешая конфликты версии при необходимости.
func (c *TOTPUpdateCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, secret *secretuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}
	vaultID, err := openVaultByName(ctx, vault, c.Vault)
	if err != nil {
		return err
	}

	baseVersion, ok, err := localTypedVersion(ctx, secret, vaultID, c.ID, totpSecretType)
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println(l.T("secret_not_found_local"))
		return nil
	}

	input, err := promptTOTPInput(l)
	if err != nil {
		return err
	}
	conflict, err := secret.UpdateTOTP(ctx, vaultID, c.ID, baseVersion, input)
	if err != nil {
		return err
	}
	if conflict != nil {
		return resolveGenericConflictInteractive(ctx, secret, l, conflict)
	}
	fmt.Println(l.T("secret_updated"))
	return nil
}

// promptTOTPInput — если пользователь вставляет строку, начинающуюся с "otpauth://" (обычно
// скопированную ссылку/QR-контент), поля парсятся автоматически (issuer/account/secret/algo/
// digits/period); иначе — обычный ручной ввод каждого поля.
func promptTOTPInput(l *clienti18n.Localizer) (secretuc.CreateTOTPInput, error) {
	raw, err := promptLine(l.T("prompt_totp_uri_or_secret"))
	if err != nil {
		return secretuc.CreateTOTPInput{}, err
	}

	var input secretuc.CreateTOTPInput
	if strings.HasPrefix(strings.TrimSpace(raw), "otpauth://") {
		input, err = secretuc.ParseOTPAuthURI(raw)
		if err != nil {
			return secretuc.CreateTOTPInput{}, err
		}
		fmt.Println(l.T("totp_parsed_from_uri"))
	} else {
		input.Secret = raw
		input.Issuer, err = promptLine(l.T("prompt_issuer"))
		if err != nil {
			return secretuc.CreateTOTPInput{}, err
		}
		input.Account, err = promptLine(l.T("prompt_account"))
		if err != nil {
			return secretuc.CreateTOTPInput{}, err
		}
	}

	title, err := promptLine(l.T("prompt_title"))
	if err != nil {
		return secretuc.CreateTOTPInput{}, err
	}
	if title != "" {
		input.Title = title
	} else if input.Title == "" {
		input.Title = input.Issuer
	}

	tagsRaw, err := promptLine(l.T("prompt_tags"))
	if err != nil {
		return secretuc.CreateTOTPInput{}, err
	}
	input.Tags = parseTags(tagsRaw)

	note, err := promptLine(l.T("prompt_note"))
	if err != nil {
		return secretuc.CreateTOTPInput{}, err
	}
	input.Note = note

	return input, nil
}
