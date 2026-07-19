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
	syncuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/sync"
	vaultuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/vault"
)

// SecretCmd — группа команд секретов.
type SecretCmd struct {
	Add    SecretAddCmd    `cmd:"" help:"Add a login/password secret to a vault."`
	List   SecretListCmd   `cmd:"" help:"List secrets of a vault (without revealing passwords)."`
	Search SecretSearchCmd `cmd:"" help:"Search secrets of a vault (Tier 2a always; notes as index loads)."`
	Get    SecretGetCmd    `cmd:"" help:"Reveal a secret's password."`
	Show   SecretShowCmd   `cmd:"" help:"Show a secret's full card (all fields, incl. password)."`
	Update SecretUpdateCmd `cmd:"" help:"Update a secret (optimistic locking; resolves conflicts)."`
	Delete SecretDeleteCmd `cmd:"" help:"Delete a secret (soft-delete, optimistic locking)."`
	OTPUse OTPUseCmd       `cmd:"otp-use" help:"Mark a recovery code as used (any secret type)."`
	Text   TextCmd         `cmd:"" help:"Manage type=text secrets (free-form notes)."`
	File   FileCmd         `cmd:"" help:"Manage type=binary secrets (large files, streamed via MinIO)."`
	Card   BankCardCmd     `cmd:"" help:"Manage type=bank_card secrets."`
	TOTP   TOTPCmd         `cmd:"" help:"Manage type=totp secrets (authenticator codes)."`
}

// SecretAddCmd — добавление секрета типа login/password в папку.
type SecretAddCmd struct {
	Vault string `arg:"" help:"Vault name."`
}

// Run запрашивает поля секрета login/password и создаёт его в указанной папке.
func (c *SecretAddCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, secret *secretuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}
	vaultID, err := openVaultByName(ctx, vault, c.Vault)
	if err != nil {
		return err
	}

	input, err := promptLoginPasswordInput(l)
	if err != nil {
		return err
	}

	if _, err := secret.CreateLoginPassword(ctx, vaultID, input); err != nil {
		return err
	}
	fmt.Println(l.T("secret_created"))
	return nil
}

// SecretListCmd — список секретов login/password в папке (без раскрытия паролей).
type SecretListCmd struct {
	Vault   string `arg:"" help:"Vault name."`
	Refresh bool   `help:"Sync with the server before listing." short:"r"`
}

// Run выводит список секретов login/password в указанной папке.
func (c *SecretListCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, secret *secretuc.UseCase, sync *syncuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}
	// По умолчанию список читается из локального кеша (без сети). --refresh форсирует синк.
	if c.Refresh {
		if err := runSync(ctx, sync, l); err != nil {
			return err
		}
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
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTITLE\tUSERNAME\tURI")
	for _, r := range rows {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.ID, r.Row.Title, r.Row.Username, r.Row.URI)
	}
	_ = w.Flush()
	return nil
}

// SecretGetCmd — раскрытие пароля секрета login/password.
type SecretGetCmd struct {
	Vault string `arg:"" help:"Vault name."`
	ID    string `arg:"" help:"Secret id (from 'secret list')."`
}

// Run расшифровывает и печатает пароль секрета login/password.
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

// SecretShowCmd — показ полной карточки секрета login/password (все поля, включая пароль).
type SecretShowCmd struct {
	Vault string `arg:"" help:"Vault name."`
	ID    string `arg:"" help:"Secret id (from 'secret list')."`
}

// Run печатает полную карточку секрета login/password (row + index + payload).
func (c *SecretShowCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, secret *secretuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}
	vaultID, err := openVaultByName(ctx, vault, c.Vault)
	if err != nil {
		return err
	}

	d, err := secret.GetDetail(ctx, vaultID, c.ID)
	if err != nil {
		return err
	}

	// Полная карточка: row (Tier 2a) + index (Tier 2b) + payload (Tier 3).
	fmt.Printf("%s: %s\n", l.T("label_title"), d.Row.Title)
	fmt.Printf("%s: %s\n", l.T("label_username"), d.Row.Username)
	fmt.Printf("%s: %s\n", l.T("label_uri"), d.Row.URI)
	if len(d.Row.Tags) > 0 {
		fmt.Printf("%s: %s\n", l.T("label_tags"), strings.Join(d.Row.Tags, ", "))
	}
	if d.Index.Note != "" {
		fmt.Printf("%s: %s\n", l.T("label_note"), d.Index.Note)
	}
	for _, kv := range d.Index.CustomFields {
		fmt.Printf("%s: %s\n", kv.Key, kv.Value)
	}
	fmt.Printf("%s: %s\n", l.T("label_password"), d.Payload.Password)
	printOTPCodes(l, d.Payload.OTPCodes)
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
