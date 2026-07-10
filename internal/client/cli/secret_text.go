package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	clienti18n "github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
	secretuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
	vaultuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/vault"
)

// TextCmd — группа команд для секретов типа text (произвольные текстовые заметки).
type TextCmd struct {
	Add    TextAddCmd    `cmd:"" help:"Add a text secret to a vault."`
	List   TextListCmd   `cmd:"" help:"List text secrets of a vault."`
	Show   TextShowCmd   `cmd:"" help:"Show a text secret's full card (incl. body)."`
	Update TextUpdateCmd `cmd:"" help:"Update a text secret (optimistic locking; resolves conflicts)."`
}

type TextAddCmd struct {
	Vault string `arg:"" help:"Vault name."`
}

func (c *TextAddCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, secret *secretuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}
	vaultID, err := openVaultByName(ctx, vault, c.Vault)
	if err != nil {
		return err
	}

	input, err := promptTextInput(l)
	if err != nil {
		return err
	}
	if _, err := secret.CreateText(ctx, vaultID, input); err != nil {
		return err
	}
	fmt.Println(l.T("secret_created"))
	return nil
}

type TextListCmd struct {
	Vault string `arg:"" help:"Vault name."`
}

func (c *TextListCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, secret *secretuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}
	vaultID, err := openVaultByName(ctx, vault, c.Vault)
	if err != nil {
		return err
	}

	rows, err := secret.ListTextRows(ctx, vaultID)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		fmt.Println(l.T("secret_empty"))
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tTAGS")
	for _, r := range rows {
		fmt.Fprintf(w, "%s\t%s\t%s\n", r.ID, r.Row.Title, joinTags(r.Row.Tags))
	}
	w.Flush()
	return nil
}

type TextShowCmd struct {
	Vault string `arg:"" help:"Vault name."`
	ID    string `arg:"" help:"Secret id (from 'secret text list')."`
}

func (c *TextShowCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, secret *secretuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}
	vaultID, err := openVaultByName(ctx, vault, c.Vault)
	if err != nil {
		return err
	}

	d, err := secret.GetTextDetail(ctx, vaultID, c.ID)
	if err != nil {
		return err
	}
	fmt.Printf("%s: %s\n", l.T("label_title"), d.Row.Title)
	if len(d.Row.Tags) > 0 {
		fmt.Printf("%s: %s\n", l.T("label_tags"), joinTags(d.Row.Tags))
	}
	if d.Index.Note != "" {
		fmt.Printf("%s: %s\n", l.T("label_note"), d.Index.Note)
	}
	for _, kv := range d.Index.CustomFields {
		fmt.Printf("%s: %s\n", kv.Key, kv.Value)
	}
	fmt.Printf("%s: %s\n", l.T("label_body"), d.Payload.Body)
	printOTPCodes(l, d.Payload.OTPCodes)
	return nil
}

type TextUpdateCmd struct {
	Vault string `arg:"" help:"Vault name."`
	ID    string `arg:"" help:"Secret id (from 'secret text list')."`
}

func (c *TextUpdateCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, secret *secretuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}
	vaultID, err := openVaultByName(ctx, vault, c.Vault)
	if err != nil {
		return err
	}

	baseVersion, ok, err := localTypedVersion(ctx, secret, vaultID, c.ID, textSecretType)
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println(l.T("secret_not_found_local"))
		return nil
	}

	input, err := promptTextInput(l)
	if err != nil {
		return err
	}
	conflict, err := secret.UpdateText(ctx, vaultID, c.ID, baseVersion, input)
	if err != nil {
		return err
	}
	if conflict != nil {
		return resolveGenericConflictInteractive(ctx, secret, l, conflict)
	}
	fmt.Println(l.T("secret_updated"))
	return nil
}

func promptTextInput(l *clienti18n.Localizer) (secretuc.CreateTextInput, error) {
	title, err := promptLine(l.T("prompt_title"))
	if err != nil {
		return secretuc.CreateTextInput{}, err
	}
	body, err := promptLine(l.T("prompt_body"))
	if err != nil {
		return secretuc.CreateTextInput{}, err
	}
	tagsRaw, err := promptLine(l.T("prompt_tags"))
	if err != nil {
		return secretuc.CreateTextInput{}, err
	}
	note, err := promptLine(l.T("prompt_note"))
	if err != nil {
		return secretuc.CreateTextInput{}, err
	}
	otpCodes, err := promptOTPCodes(l)
	if err != nil {
		return secretuc.CreateTextInput{}, err
	}
	return secretuc.CreateTextInput{Title: title, Body: body, Tags: parseTags(tagsRaw), Note: note, OTPCodes: otpCodes}, nil
}

func joinTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	out := tags[0]
	for _, t := range tags[1:] {
		out += "," + t
	}
	return out
}
